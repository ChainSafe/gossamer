// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package stress

import "fmt"

var (
	errFinalizedBlockMismatch = fmt.Errorf("node finalised head hashes don't match")
	errNoFinalizedBlock       = fmt.Errorf("did not finalise block for round")
	errChainHeadMismatch      = fmt.Errorf("node chain head hashes don't match")
)
