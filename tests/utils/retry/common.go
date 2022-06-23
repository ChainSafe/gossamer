// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package retry

import (
	"context"
	"fmt"
	"time"
)

func waitAfterFail(ctx context.Context, retryWait time.Duration,
	failedTries *int) {
	*failedTries++
	waitCtx, waitCancel := context.WithTimeout(ctx, retryWait)
	<-waitCtx.Done()
	waitCancel()
}

func makeError(failedTries int, retryWait time.Duration, ctxErr error) (err error) {
	totalRetryTime := time.Duration(failedTries) * retryWait
	tryWord := "try"
	if failedTries > 1 {
		tryWord = "tries"
	}
	return fmt.Errorf("failed after %d %s during %s (%w)",
		failedTries, tryWord, totalRetryTime, ctxErr)
}
