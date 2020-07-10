package main

import ()

// WWDisplay represents a type of display for WW.
type WWDisplay interface {
	// Init should perform whatever initialisation is needed for the display, including initial render.
	Init(config WWConfig) error

	Stop() error

	// Called when process state changes
	UpdateStatus(status Status, header string, cmdNameAndArgs string)

	// Called when we receive data on stdout
	OnStdout(data string)

	// Called when we receive data on stder
	OnStderr(data string)
}
