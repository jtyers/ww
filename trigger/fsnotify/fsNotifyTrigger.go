package fsnotify

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/jtyers/ww/trigger/fsnotify/dirwalk"
)

type FsNotifyTrigger struct {
	watcher *fsnotify.Watcher
}

func NewFsNotifyTrigger(directory string, excludeNames []string) (*FsNotifyTrigger, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create watcher: %v", err)
	}

	paths, err := dirwalk.WalkDirectory(directory, excludeNames, nil, true)
	if err != nil {
		return nil, fmt.Errorf("walk: %v", err)
	}

	for _, path := range paths {
		watcher.Add(path)
	}

	trigger := FsNotifyTrigger{watcher}
	return &trigger, nil
}

func (t *FsNotifyTrigger) WaitForTrigger(interruptChan <-chan error) (<-chan bool, <-chan string) {
	triggerChan := make(chan bool)

	// Must be buffered.
	statusChan := make(chan string, 10)

	go func() {
		statusChan <- "waiting for changes..."
		for {
			select {
			case event, ok := <-t.watcher.Events:
				if !ok {
					continue
				}

				if event.Op&fsnotify.Write == fsnotify.Write || event.Op == fsnotify.Create || event.Op == fsnotify.Remove || event.Op == fsnotify.Rename || event.Op == fsnotify.Chmod {
					triggerChan <- true
					break
				}

			case err, ok := <-t.watcher.Errors:
				if !ok {
					statusChan <- fmt.Sprintf("watch error (%v)", err)
				}

			case _ = <-interruptChan:
				t.watcher.Close() // close watcher and quit goroutine
				break
			}
		}

	}()

	return triggerChan, statusChan
}
