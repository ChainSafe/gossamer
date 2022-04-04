package retry

import (
	"context"
	"fmt"
	"time"
)

// UntilOK retries the function `f` until it succeeds.
// It waits `retryWait` after each failed call to `f`.
// If the context `ctx` is canceled, the function returns
// immediately an error stating the number of failed tries,
// for how long it retried and the context error.
func UntilOK(ctx context.Context, retryWait time.Duration,
	f func() (ok bool)) (err error) {
	failedTries := 0
	for ctx.Err() == nil {
		ok := f()
		if ok {
			return nil
		}

		failedTries++
		waitCtx, waitCancel := context.WithTimeout(ctx, retryWait)
		<-waitCtx.Done()
		waitCancel()
	}

	totalRetryTime := time.Duration(failedTries) * retryWait
	tryWord := "try"
	if failedTries > 1 {
		tryWord = "tries"
	}
	return fmt.Errorf("failed after %d %s during %s (%w)",
		failedTries, tryWord, totalRetryTime, ctx.Err())
}
