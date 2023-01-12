// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

import "context"

// runner runs in a blocking manner.
type runner interface {
	Run(ctx context.Context, ready chan<- struct{}, done chan<- error)
}
