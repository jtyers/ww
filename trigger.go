package main

import (
	"fmt"
	"time"
)

// WWWTrigger is the interface provided by triggers, which tell WW when to re-execute
// commands. Example of triggers are a simple time-based interval ("every X seconds"),
// and filesystem events ("re-run whenever files in the current directory are modified").
//
type WWTrigger interface {
	// WaitForTrigger should wait for the next moment to trigger. It returns a channel and
	// sends `true` on the channel when the trigger should fire. It also returns a statusChan,
	// which can optionally send status updates to provide feedback to the user on trigger
	// progress.
	//
	// The wait might be interrupted, at which point interruptChan will receive an error
	// signalling the reason for the interruption. This allows your trigger to perform
	// cleanup if needed. When a WaitForTrigger() is interrupted in this way, it should
	// send `false` on the returned channel.
	//
	WaitForTrigger(interruptChan <-chan error) (triggerChan <-chan bool, statusChan <-chan string)
}

// IntervalWWTrigger is a WWTrigger that re-executes at a time interval.
type IntervalWWTrigger struct {
	Interval time.Duration
}

func (i *IntervalWWTrigger) WaitForTrigger(interruptChan <-chan error) (<-chan bool, <-chan string) {
	c := make(chan bool)
	s := make(chan string)

	go func() {
		for j := i.Interval.Seconds() - 1; j >= 0; j-- {
			select {
			case <-time.After(time.Second):
				s <- fmt.Sprintf("(running in %0.fs)", j)
			case <-interruptChan:
				c <- false
			}
		}
		c <- true
		close(c)
		close(s)
	}()

	return c, s
}
