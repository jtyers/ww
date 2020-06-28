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

	// FIXME change this to a callback interface (see channels-are-hell article saved in Pocket)
	WaitForTrigger(interruptChan <-chan error) (triggerChan <-chan bool, statusChan <-chan string)
}

// IntervalWWTrigger is a WWTrigger that re-executes at a time interval.
type IntervalWWTrigger struct {
	Interval time.Duration
}

func (i *IntervalWWTrigger) WaitForTrigger(interruptChan <-chan error) (<-chan bool, <-chan string) {
	c := make(chan bool)

	// When creating the status channel we set it up as buffered, so that we can write to
	// it without needing to wait for a reader to be set up at the other end. Otherwise,
	// we'll hang in scenarios where the caller doesn't need status updates (eg tests!).
	//
	// Set to Interval + 1 to ensure there is just enough buffer to contain all our status
	// updates.
	s := make(chan string, int(i.Interval.Seconds())+1)

	go func() {
		defer close(c)
		defer close(s)

		for j := i.Interval.Seconds() - 1; j >= 0; j-- {
			select {
			case <-interruptChan:
				c <- false
				return
			case <-time.After(time.Second):
				s <- fmt.Sprintf("(running in %0.fs)", j)
			}
		}

		c <- true
	}()

	return c, s
}
