// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import "testing"

func tryIntegrationTest(t *testing.T) {
	subsystemToOverseer := make(chan<- any)
	candidateBacking := New(subsystemToOverseer)
	candidateBacking.BlockState = nil // use mock block state

	
}
