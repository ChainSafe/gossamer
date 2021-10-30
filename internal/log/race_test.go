package log

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"
)

func Test_Race(t *testing.T) {
	t.Parallel()

	buffer := bytes.NewBuffer(nil)

	parent := New(SetWriter(buffer))

	childA := parent.New()
	childB := parent.New()

	childAA := childA.New()
	childBA := childB.New()

	loggers := []*Logger{
		parent, childA, childB, childAA, childBA,
	}

	readyWait := new(sync.WaitGroup)
	readyWait.Add(len(loggers))

	doneWait := new(sync.WaitGroup)
	doneWait.Add(len(loggers))

	// run for 50ms
	ctxTimerStarted := make(chan struct{})
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		const timeout = 50 * time.Millisecond
		readyWait.Wait()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		close(ctxTimerStarted)
	}()
	defer cancel()

	for _, logger := range loggers {
		go func(logger *Logger) {
			defer doneWait.Done()
			readyWait.Done()
			readyWait.Wait()
			<-ctxTimerStarted

			for ctx.Err() != nil {
				// test relies on the -race detector
				// to detect concurrent writes to the buffer.
				logger.Info("x")
			}
		}(logger)
	}

	doneWait.Wait()
}
