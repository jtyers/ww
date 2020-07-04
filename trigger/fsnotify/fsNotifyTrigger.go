package fsnotify

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/jtyers/ww/trigger/fsnotify/dirwalk"
)

type FsNotifyTrigger struct {
	directory    string
	excludeNames []string
}

func NewFsNotifyTrigger(directory string, excludeNames []string) (*FsNotifyTrigger, error) {
	trigger := FsNotifyTrigger{directory, excludeNames}
	return &trigger, nil
}

func (t *FsNotifyTrigger) WaitForTrigger(interruptChan <-chan error) (<-chan bool, <-chan string) {
	// Since fsnotify's API only allows you to create a watcher (and that watcher then
	// imediately starts watching stuff), we have to create a fresh watcher every time
	// we wait for the trigger, and also do the related directory walking, since you cannot
	// ever re-use a watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(fmt.Errorf("could not create watcher: %v", err))
	}

	paths, err := dirwalk.WalkDirectory(t.directory, t.excludeNames, nil, true)
	if err != nil {
		//return nil, fmt.Errorf("walk: %v", err)
		return nil, nil
	}

	for _, path := range paths {
		watcher.Add(path)
	}

	triggerChan := make(chan bool)

	// Must be buffered.
	statusChan := make(chan string, 10)

	go func() {
		defer close(triggerChan)
		defer close(statusChan)

		statusChan <- "waiting for changes..."
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					continue
				}

				if event.Op&fsnotify.Write == fsnotify.Write || event.Op == fsnotify.Create || event.Op == fsnotify.Remove || event.Op == fsnotify.Rename || event.Op == fsnotify.Chmod {
					fmt.Printf("change detected in %v\n", event.Name)
					triggerChan <- true
					watcher.Close() // close watcher and quit goroutine
					return
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					statusChan <- fmt.Sprintf("watch error (%v)", err)
				}

			case _ = <-interruptChan:
				watcher.Close() // close watcher and quit goroutine
				return
			}
		}
	}()

	return triggerChan, statusChan
}
