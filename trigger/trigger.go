package trigger

// WWTrigger is the interface provided by triggers, which tell WW when to re-execute
// commands. Example of triggers are a simple time-based interval ("every X seconds"),
// and filesystem events ("re-run whenever files in the current directory are modified").
//
// WWTriggers implementations may also implement the io.Closer interface, in which case
// they'll be Close()d just before ww exits.
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
