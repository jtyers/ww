package interval

import (
	"fmt"
	"time"
)

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
				s <- fmt.Sprintf("(%0.fs)", j)
			}
		}

		c <- true
	}()

	return c, s
}
