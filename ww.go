package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

var ErrInterrupted = fmt.Errorf("interrupted")

var StatusSuccess = Status{"[#002200:#008800]", "success"}
var StatusFailed = Status{"[#ffdddd:#880000]", "failed"}
var StatusRunning = Status{"[#aaaaaa]", "running"}

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

	// Stores the Command used to execute - this is here to track the current state of execution
	Command *exec.Cmd

	Status Status

	StatusText string
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
			SetTextColor(tcell.ColorDefault).
			SetRegions(true).
			SetWordWrap(true).
			SetChangedFunc(func() {
				w.state.app.Draw()
			}),

		interruptChan: make(chan error),
	}

	w.state.grid.AddItem(w.state.header, 0, 0, 1, 1, 5, 0, false)
	w.state.grid.AddItem(w.state.status, 0, 1, 1, 1, 5, 20, false)
	w.state.grid.AddItem(w.state.textView, 1, 0, 1, 2, 0, 0, true)
}

func (w *WW) UpdateStatus(status Status, header string) {
	w.state.Status = status
	w.state.StatusText = header

	cmdNameAndArgs := tview.Escape(fmt.Sprintf("%s %s", w.config.Command, strings.Join(w.config.Args, " ")))

	w.state.app.QueueUpdateDraw(func() {
		w.state.header.SetText(w.state.Status.colorCode + cmdNameAndArgs + " " + w.state.StatusText)
		w.state.status.SetText(w.state.Status.colorCode + tview.Escape(w.state.Status.name))
	})
}

func (w *WW) Run() error {
	// Kick off a goroutine that consumes events from the command and updates the TextView/Header
	// accordingly.

	stdoutChan := make(chan string, 5) // buffered, so we can start writing before readers start reading
	stderrChan := make(chan string, 5)
	evtChan := make(chan *os.ProcessState, 5)

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
		// This loops around, pulling status updates from evtChan, and updating the UI accordingly.
		//
		// Note that output from the command being executed is *NOT* processed by this goroutine;
		// executeOnce() has its own goroutines that read from those pipes and print directly to the
		// textView.

		for {
			newState := <-evtChan
			if newState != nil && newState.Exited() {

				if newState.Success() {
					w.UpdateStatus(StatusSuccess, fmt.Sprintf("(last run %s)", time.Now().Format("15:04:05")))
				} else {
					w.UpdateStatus(StatusFailed, fmt.Sprintf("(exited with %d)", newState.ExitCode()))
				}

				// now run triggers, if configured
				if w.config.Trigger != nil {
					// Wait for trigger to fire
					triggerChan, statusChan := w.config.Trigger.WaitForTrigger(w.state.interruptChan)

					outerbreak := false

					// This loop is here to WaitForTrigger, but we also need to loop and select to catch
					// updates coming in on statusChan and call UpdateStatus for them.
					for {
						select {
						case statusUpdate := <-statusChan:
							w.UpdateStatus(w.state.Status, statusUpdate)

						case trigger := <-triggerChan:
							if trigger {
								w.state.app.QueueUpdateDraw(func() {
									w.state.textView.Clear()
								})

								beginExecuteCommand()

							} else {
								outerbreak = true // if trigger was interrupted (i.e. did not return true), quit the loop
							}
						}

						if outerbreak {
							outerbreak = false
							break
						}
					}

					if outerbreak {
						outerbreak = false
						break
					}

				} else {
					fmt.Fprint(w.state.textView, "\n[red]ww [yellow]Press Ctrl+C to exit\n")
					break
				}

			} else {
				w.UpdateStatus(StatusRunning, "")
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
			default:
				if err != nil {
					die(w, "read: %v", err)
				}
			case nil:
				c <- fmt.Sprintln(tview.Escape(scanner.Text()))
			case io.EOF:
				// do nowt (exit goroutine)
				closeChan <- true
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
	ww := &WW{config: parseArgs()}
	ww.Init()

	if err := ww.Run(); err != nil {
		die(ww, "%v", err)
	}
}
