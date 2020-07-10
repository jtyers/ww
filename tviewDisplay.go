package main

import (
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// Uses tview to create an ncurses-like scrollable display taking up the whole screen
type TviewDisplay struct {
	config WWConfig

	// Instance of the tview Application that controls rendering to the terminal and associated event loop.
	app *tview.Application

	// The main textView containing the output of executed commands.
	textView *tview.TextView

	// The grid
	grid *tview.Grid

	// The header cell in the grid
	header *tview.TextView
	status *tview.TextView

	clearOnNextOutput bool
}

func (d *TviewDisplay) Init(config WWConfig) error {
	d.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	d.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight).
		SetText("status")

	d.textView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextColor(tcell.ColorDefault).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			d.app.Draw()
		})

	d.grid = tview.NewGrid().
		SetRows(1, 0).
		SetColumns(0, 10).
		SetBorders(false).
		AddItem(d.header, 0, 0, 1, 1, 5, 0, false).
		AddItem(d.status, 0, 1, 1, 1, 5, 20, false).
		AddItem(d.textView, 1, 0, 1, 2, 0, 0, true)

	d.app = tview.NewApplication().
		SetRoot(d.grid, true).
		EnableMouse(true)

	if err := d.app.Run(); err != nil {
		return err
	}

	return nil
}

func (d *TviewDisplay) UpdateStatus(status Status, cmdNameAndArgs string, header string) {
	switch status {
	case StatusTriggered:
		d.clearOnNextOutput = true

	case StatusEnded:
		fmt.Fprint(d.textView, "\n[red]ww [yellow]Press Ctrl+C to exit\n")

	default:
		d.app.QueueUpdateDraw(func() {
			d.header.Box.SetBackgroundColor(status.backgroundColor)
			d.status.Box.SetBackgroundColor(status.backgroundColor)

			d.header.SetText(status.colorCode + cmdNameAndArgs + " " + header)
			d.status.SetText(status.colorCode + tview.Escape(status.name))
		})
	}
}

func (d *TviewDisplay) OnStdout(data string) {
	if d.clearOnNextOutput {
		d.app.QueueUpdate(func() {
			d.textView.Clear()
			d.clearOnNextOutput = false
		})
	}

	fmt.Fprint(d.textView, d.config.Highlighter.Highlight(data))
}

func (d *TviewDisplay) OnStderr(data string) {
	d.OnStdout(data) // same implementation
}

func (d *TviewDisplay) Stop() error {
	d.app.Stop()
	return nil
}
