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
	"github.com/jtyers/ww/trigger"
	"github.com/rivo/tview"
)

var ErrInterrupted = fmt.Errorf("interrupted")

var StatusSuccess = Status{"[#002200:#008800]", tcell.ColorDarkGreen, "success"}
var StatusFailed = Status{"[#ffdddd:#880000]", tcell.ColorDarkRed, "failed"}
var StatusRunning = Status{"[#aaaaaa]", tcell.ColorGray, "running"}

type Status struct {
	colorCode       string
	backgroundColor tcell.Color
	name            string
}

type WWConfig struct {
	// Command is the command to execute.
	Command string

	// Args is the args to pass to the command.
	Args []string

	// Trigger is the WWTrigger used to trigger re-executions. Might be nil if the user only wants
	// the command to run once.
	Trigger trigger.WWTrigger

	Highlighter *Highlighter
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
			SetTextAlign(tview.AlignLeft),

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
		w.state.header.Box.SetBackgroundColor(w.state.Status.backgroundColor)
		w.state.status.Box.SetBackgroundColor(w.state.Status.backgroundColor)

		w.state.header.SetText(w.state.Status.colorCode + cmdNameAndArgs + " " + w.state.StatusText)
		w.state.status.SetText(w.state.Status.colorCode + tview.Escape(w.state.Status.name))
	})
}

func (w *WW) waitForTriggersOrExit() {
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
					w.state.app.QueueUpdate(func() {
						w.state.textView.Clear()
					})

					w.beginExecuteCommand()

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
			return
		}

	} else {
		fmt.Fprint(w.state.textView, "\n[red]ww [yellow]Press Ctrl+C to exit\n")
		return
	}
}

func (w *WW) beginExecuteCommand() {
	if err := w.executeOnce(
		func(stdout string) {
			fmt.Fprint(w.state.textView, w.config.Highlighter.Highlight(stdout))
		},
		func(stderr string) {
			fmt.Fprint(w.state.textView, w.config.Highlighter.Highlight(stderr))
		},
		func(psc ProcessStatusChange) {
			// This loops around, pulling status updates from evtChan, and updating the UI accordingly.
			//
			// Note that output from the command being executed is *NOT* processed by this goroutine;
			// executeOnce() has its own goroutines that read from those pipes and print directly to the
			// textView.

			switch psc.Status {
			case ProcessStatusStarted:
				w.UpdateStatus(StatusRunning, "")

			case ProcessStatusSucceeded:
				w.UpdateStatus(StatusSuccess, fmt.Sprintf("(last run %s)", time.Now().Format("15:04:05")))
				w.waitForTriggersOrExit()

			case ProcessStatusFailed:
				w.UpdateStatus(StatusFailed, fmt.Sprintf("(exited with %d)", psc.State.ExitCode()))
				w.waitForTriggersOrExit()
			}
		},
	); err != nil {
		w.UpdateStatus(StatusFailed, err.Error())
	}
}

func (w *WW) Run() error {
	// Kick off a goroutine that consumes events from the command and updates the TextView/Header
	// accordingly.

	go w.beginExecuteCommand()

	if err := w.state.app.SetRoot(w.state.grid, true).EnableMouse(true).Run(); err != nil {
		return err
	}

	return nil
}

func (w *WW) Stop() {
	w.state.app.Stop()
}

const (
	// ProcessStatusStarted is when the process has been started (but has not yet finished)
	ProcessStatusStarted = 1 // use non-zero so the zero value is not conflated with this

	// ProcessStatusSucceeded is when the process has exited with a zero exit code
	ProcessStatusSucceeded = 2

	// ProcessStatusFailed is when the process has exited with a non-zero exit code
	ProcessStatusFailed = 3
)

type ProcessStatusChange struct {
	State *os.ProcessState

	// Status indicates the new status of the process. See ProcessStatus* constants for possible values.
	Status int
}

// Execute the configured command, calling the given callbacks as we receive output from the process.
//
// Will only return an error if there is an error prior to launching the process. If the process itself
// encounters an error, that will be signalled via a stateChangeCallback.
func (w *WW) executeOnce(stdoutCallback func(string), stderrCallback func(string), stateChangeCallback func(ProcessStatusChange)) error {
	// According to the godoc, we should not call Wait() before we've finished reading stdout/stderr, since Wait will close those pipes
	// as soon as the command has completed. However, our reading (and detection ofEOF) is inside goroutines, so this callback is here
	// to detect EOF on both streams, then call Wait() to close the pipes and clean up.
	maybeWaitForCommand := func(cmd *exec.Cmd) {
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				// process failed, so not an error, simply send state down the event channel

				// FIXME sleep momentarily to allow reads/prints of stdout/stderr to complete
				time.Sleep(time.Millisecond * 100)

				stateChangeCallback(ProcessStatusChange{State: exitErr.ProcessState, Status: ProcessStatusFailed})

			} else {
				if err.Error() == "wait: no child processes" {
					// we ignore this error specifically; it seems to occur if the process exits before we Wait()
				} else {
					die(w, "failed waiting for cmd: %v", err)
				}
			}
		}

		// FIXME sleep momentarily to allow reads/prints of stdout/stderr to complete
		time.Sleep(time.Millisecond * 200)

		stateChangeCallback(ProcessStatusChange{State: cmd.ProcessState, Status: ProcessStatusSucceeded})
	}

	scannerReader := func(pipe io.Reader, dataCallback func(string), onEofCallback func()) {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			dataCallback(fmt.Sprintln(tview.Escape(scanner.Text()))) // add \n as Scanner stripped it off

		}
		if err := scanner.Err(); err != nil {
			die(w, "read: %v", err)
		}

		onEofCallback()
	}

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

	stateChangeCallback(ProcessStatusChange{State: cmd.ProcessState, Status: ProcessStatusStarted})

	stdoutEofChan := make(chan bool, 1)
	stderrEofChan := make(chan bool, 1)

	go scannerReader(stdout, stdoutCallback, func() { stdoutEofChan <- true })
	go scannerReader(stderr, stderrCallback, func() { stderrEofChan <- true })

	<-stdoutEofChan
	<-stderrEofChan

	close(stdoutEofChan)
	close(stderrEofChan)

	maybeWaitForCommand(cmd)

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
