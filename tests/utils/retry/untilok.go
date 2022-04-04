// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package retry

import (
	"context"
	"fmt"
	"time"
)

// UntilOK retries the function `f` until it returns a true
// value for `ok` or a non nil error.
// It waits `retryWait` after each failed call to `f`.
// If the context `ctx` is canceled, the function returns
// immediately an error stating the number of failed tries,
// for how long it retried and the context error.
func UntilOK(ctx context.Context, retryWait time.Duration,
	f func() (ok bool, err error)) (err error) {
	failedTries := 0
	for ctx.Err() == nil {
		ok, err := f()
		if ok {
			return nil
		} else if err != nil {
			return fmt.Errorf("stop retrying function: %w", err)
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
