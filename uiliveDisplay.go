package main

import (
	"fmt"
	"io"
	"time"

	"github.com/gosuri/uilive"
)

// Uses uilive to render output on the next line of the terminal, overwriting it as it updates (e.g. like docker pull)
type UiLiveDisplay struct {
	config WWConfig

	writer   *uilive.Writer
	iowriter io.Writer

	statusText string
	text       string

	wait chan bool

	clearOnNextOutput bool // set to true after command is triggered
}

func (d *UiLiveDisplay) Init(config WWConfig) error {
	d.writer = uilive.New()
	d.writer.RefreshInterval = time.Millisecond * 250 // reduce cpu usage a LOT (default is 1ms)

	d.iowriter = d.writer.Newline()
	d.writer.Start()

	d.wait = make(chan bool)

	outbreak := false
	for {
		select {
		case <-time.After(250 * time.Millisecond):
			fmt.Fprintf(d.iowriter, "%s%s", d.statusText, d.text)

		case <-d.wait:
			outbreak = true
		}

		if outbreak {
			break
		}
	}

	return nil
}

func (d *UiLiveDisplay) UpdateStatus(status Status, header string, cmdNameAndArgs string) {
	switch status {
	case StatusTriggered:
		// We could clear the screen as soon as cmd is triggered, but for slower commands (e.g. anything
		// involving a network lookup such as `kubectl`) this causes the screen to flicker, so instead we
		// set a flag, and then only clear the screen once we've started receiving output, to reduce flicker.
		d.clearOnNextOutput = true

	case StatusEnded:
		d.text += "\n" + "\n[red]ww [yellow]Press Ctrl+C to exit\n"

	default:
		d.statusText = fmt.Sprintf("%s %s %s\n", cmdNameAndArgs, status.name, header)
	}
}

func (d *UiLiveDisplay) OnStdout(data string) {
	if d.clearOnNextOutput {
		d.text = "" // clear buffer
		d.clearOnNextOutput = false
	}

	d.text += d.config.Highlighter.Highlight(data)
}

func (d *UiLiveDisplay) OnStderr(data string) {
	d.OnStdout(data) // same implementation
}

func (d *UiLiveDisplay) Stop() error {
	d.text += "\n" + "\n[red]ww [yellow]Press Ctrl+C to exit\n"
	d.wait <- true
	d.writer.Stop() // flush and stop rendering
	return nil
}
