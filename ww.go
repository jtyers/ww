package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rivo/tview"
)

var ErrInterrupted = fmt.Errorf("interrupted")

var StatusSuccess = Status{"[#00ff00]", "success"}
var StatusFailed = Status{"[#ff0000]", "failed"}
var StatusRunning = Status{"[#cacaca]", "running"}

type Status struct {
	colorCode string
	name      string
}

type WWConfig struct {
	// Command is the command to execute.
	Command string

	// Args is the args to pass to the command.
	Args []string

	// Trigger is the WWTrigger used to trigger re-executions. Might be nil if the user only wants
	// the command to run once.
	Trigger WWTrigger
}

type WWState struct {
	// Instance of the tview Application that controls rendering to the terminal and associated event loop.
	app *tview.Application

	// The main textView containing the output of executed commands.
	textView *tview.TextView

	// The grid
	grid *tview.Grid

	// The header cell in the grid
	header *tview.TextView
	status *tview.TextView

	// A channel used to interrupt the configured WWTrigger
	interruptChan chan error

	// If true, stop the Application on next change. Used to escape from within the executeLoop, which
	// starts execution before Application has started running.
	stopOnNextChange bool

	// Stores the Command used to execute - this is here to track the current state of execution
	Command *exec.Cmd
}

// WW is the main struct controlling what we do and display.
type WW struct {
	// User-specified configuration of this WW instance
	config WWConfig

	// State of this WW instance. Deliberately not a pointer since it is never used elsewhere.
	state WWState
}

// Init sets up the WW instance's UI.
func (w *WW) Init() {
	w.state = WWState{
		app: tview.NewApplication(),

		grid: tview.NewGrid().
			SetRows(1, 0).
			SetColumns(0, 10).
			SetBorders(false),

		header: tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignLeft).
			SetText("Hello!"),

		status: tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignRight).
			SetText("status"),

		textView: tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetWordWrap(true).
			SetChangedFunc(func() {
				if w.state.stopOnNextChange {
					w.state.app.Stop()
				} else {
					w.state.app.Draw()
				}
			}),

		interruptChan: make(chan error),
	}

	w.state.grid.AddItem(w.state.header, 0, 0, 1, 1, 5, 0, true)
	w.state.grid.AddItem(w.state.status, 0, 1, 1, 1, 5, 20, false)
	w.state.grid.AddItem(w.state.textView, 1, 0, 1, 2, 0, 0, false)
}

type WWCommandEvent struct {
	// Has the start time of the command
}

func (w *WW) UpdateStatus(status Status, header string) {
	w.state.app.QueueUpdateDraw(func() {
		w.state.header.SetText(status.colorCode + tview.Escape(header))
		w.state.status.SetText(status.colorCode + tview.Escape(status.name))
	})
}

func (w *WW) Run() error {
	// Kick off a goroutine that consumes events from the command and updates the TextView/Header
	// accordingly.

	stdoutChan := make(chan string, 5) // buffered, so we can start writing before readers start reading
	stderrChan := make(chan string, 5)
	evtChan := make(chan *os.ProcessState, 5)

	cmdNameAndArgs := tview.Escape(fmt.Sprintf("%s %s", w.config.Command, strings.Join(w.config.Args, " ")))

	beginExecuteCommand := func() {
		if err := w.executeOnce(stdoutChan, stderrChan, evtChan); err != nil {
			w.UpdateStatus(StatusFailed, err.Error())
		}
	}

	go beginExecuteCommand()

	go func() {
		for {
			select {
			case stdout := <-stdoutChan:
				fmt.Fprintf(w.state.textView, stdout)
			case stderr := <-stderrChan:
				fmt.Fprintf(w.state.textView, stderr)
			}
		}
	}()

	go func() {
		for {
			newState := <-evtChan
			if newState != nil && newState.Exited() {

				if newState.Success() {
					w.UpdateStatus(StatusSuccess, cmdNameAndArgs)
				} else {
					w.UpdateStatus(StatusFailed, cmdNameAndArgs)
				}

				// now run triggers, if configured
				if w.config.Trigger != nil {
					// Wait for trigger to fire
					t := <-w.config.Trigger.WaitForTrigger(w.state.interruptChan)

					if !t {
						break // if trigger was interrupted (i.e. did not return true), quit the loop
					}

					w.state.app.QueueUpdateDraw(func() {
						w.state.textView.Clear()
					})

					beginExecuteCommand()

				} else {
					fmt.Fprint(w.state.textView, "\n[red]ww [yellow]Press Ctrl+C to exit\n")
					break
				}

			} else {
				w.UpdateStatus(StatusRunning, cmdNameAndArgs)
			}
		}
	}()

	if err := w.state.app.SetRoot(w.state.grid, true).EnableMouse(true).Run(); err != nil {
		return err
	}

	return nil
}

func (w *WW) Stop() {
	w.state.app.Stop()
}

func (w *WW) executeOnce(stdoutChan chan string, stderrChan chan string, evtChan chan *os.ProcessState) error {

	cmd := exec.Command(w.config.Command, w.config.Args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed opening stdout: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed opening stderr: %v", err)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start: %v", err)
	}

	evtChan <- cmd.ProcessState

	stdoutClose := make(chan bool, 1)
	stderrClose := make(chan bool, 1)

	scannerReader := func(pipe io.Reader, c chan string, closeChan chan bool) {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			switch scanner.Err() {
			case nil:
				c <- fmt.Sprintln(tview.Escape(scanner.Text()))
			case io.EOF:
				// do nowt (exit goroutine)
				closeChan <- true
			default:
				die(w, "read: %v", err)
			}
		}
	}

	go scannerReader(stdout, stdoutChan, stdoutClose)
	go scannerReader(stderr, stderrChan, stderrClose)

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// process failed, so not an error, simply send state down the event channel

			// FIXME sleep momentarily to allow reads/prints of stdout/stderr to complete
			time.Sleep(time.Millisecond * 100)

			evtChan <- exitErr.ProcessState

			return nil

		} else {
			die(w, "failed waiting for cmd: %v", err)
			return nil
		}
	}

	evtChan <- cmd.ProcessState

	return nil
}

func die(ww *WW, msg string, args ...interface{}) {
	ww.Stop()
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}

func main() {
	config := WWConfig{}

	var useInterval int
	flag.IntVar(&useInterval, "n", 0, "specify number of seconds to run command")
	flag.IntVar(&useInterval, "interval", 0, "specify number of seconds to run command")

	flag.Parse()

	args := flag.Args()
	config.Command = args[0]
	config.Args = args[1:]

	if useInterval > 0 {
		Interval, err := time.ParseDuration(fmt.Sprintf("%ds", useInterval))
		if err != nil {
			die(nil, "invalid --interval: %v", err)
		}

		config.Trigger = &IntervalWWTrigger{Interval}
	}

	ww := &WW{config: config}
	ww.Init()

	if err := ww.Run(); err != nil {
		die(ww, "%v", err)
	}
}
