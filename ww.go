package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rivo/tview"
)

var ErrInterrupted = fmt.Errorf("interrupted")

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
			SetColumns(0),

		header: tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter).
			SetText("Hello!"),

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

	w.state.grid.AddItem(w.state.header, 0, 0, 1, 1, 0, 0, true)
	w.state.grid.AddItem(w.state.textView, 1, 0, 1, 1, 0, 0, false)
}

type WWCommandEvent struct {
	// Has the start time of the command
}

func (w *WW) Run() error {
	// Kick off a goroutine that consumes events from the command and updates the TextView/Header
	// accordingly

	if err := w.executeOnce(); err != nil {
		return err
	}

	/*
		w.state.app.QueueUpdate(func() {
			for {
				if err := w.executeOnce(); err != nil {
					die(w, "executeOnce: %v", err)
					break
				}

				if w.config.Trigger != nil {
					// Wait for trigger to fire
					t := <-w.config.Trigger.WaitForTrigger(w.state.interruptChan)

					if !t {
						break // if trigger was interrupted (i.e. did not return true), quit the loop
					}

				} else {
					fmt.Fprint(w.state.textView, "\n[red]ww [yellow]Press Ctrl+C to exit\n")
					break
				}
			}
		})
	*/

	if err := w.state.app.SetRoot(w.state.grid, true).EnableMouse(true).Run(); err != nil {
		return err
	}

	return nil
}

func (w *WW) Stop() {
	w.state.app.Stop()
}

func (w *WW) executeOnce() error {
	w.state.header.SetText("[#00ff00]" + tview.Escape(fmt.Sprintf("%s %s", w.config.Command, strings.Join(w.config.Args, " "))))

	cmd := exec.Command(w.config.Command, w.config.Args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed opening stdout: %v", err)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() { // since we scan by line, we must add the lines back into printed output
		fmt.Fprint(w.state.textView, tview.Escape(scanner.Text()), "\n")
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed reading stdout: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed waiting for cmd: %v", err)
	}

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
	flag.IntVar(&useInterval, "interval", 0, "specify number of seconds to run command")

	flag.Parse()

	args := flag.Args()
	config.Command = args[0]
	config.Args = args[1:len(args)]

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
