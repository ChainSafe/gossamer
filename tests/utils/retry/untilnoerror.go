// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package retry

import (
	"context"
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

		waitAfterFail(ctx, retryWait, &failedTries)
	}

	return makeError(failedTries, retryWait, ctx.Err())
}
