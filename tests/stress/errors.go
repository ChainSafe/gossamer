// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package stress

import (
	"errors"
)

var (
	errFinalizedBlockMismatch = errors.New("node finalised head hashes don't match")
	errNoFinalizedBlock       = errors.New("did not finalise block for round")
	errNoBlockAtNumber        = errors.New("no blocks found for given number")
	errBlocksAtNumberMismatch = errors.New("different blocks found for given number")
	errChainHeadMismatch      = errors.New("node chain head hashes don't match")
)
