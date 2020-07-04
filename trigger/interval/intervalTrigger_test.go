package interval

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWaitForTrigger(t *testing.T) {
	var tests = []struct {
		name              string
		interval          time.Duration
		interruptInterval time.Duration
		expectedResult    bool
	}{
		{
			"should return true when no interrupt",
			1 * time.Second,
			0,
			true,
		},
		{
			"should return false when interrupted before trigger fires",
			2 * time.Second,
			1 * time.Second,
			false,
		},
		{
			"should return true when interrupted after trigger fires",
			1 * time.Second,
			2 * time.Second,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			trigger := IntervalWWTrigger{Interval: test.interval}

			interruptChan := make(chan error)

			if test.interruptInterval > 0 {
				go func() {
					<-time.After(test.interruptInterval)
					interruptChan <- fmt.Errorf("some error")
				}()
			}

			// when
			triggerChan, _ := trigger.WaitForTrigger(interruptChan)

			result := <-triggerChan

			// then

			require.Equal(t, test.expectedResult, result)
		})
	}
}
