// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package retry

import (
	"context"
	"fmt"
	"time"
)

// UntilNoError retries the function `f` until it returns a nil error.
// It waits `retryWait` after each failed call to `f`.
// If the context `ctx` is canceled, the function returns
// immediately an error stating the number of failed tries,
// for how long it retried and the last error returned by `f`.
func UntilNoError(ctx context.Context, retryWait time.Duration,
	f func() (err error)) (err error) {
	failedTries := 0
	for ctx.Err() == nil {
		err = f()
		if err == nil {
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
	return fmt.Errorf("failed after %d %s during %s: %w (%s)",
		failedTries, tryWord, totalRetryTime, err, ctx.Err())
}
