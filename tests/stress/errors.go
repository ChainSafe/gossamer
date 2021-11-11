// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package stress

import (
	"errors"
)

//nolint
var (
	errFinalizedBlockMismatch = errors.New("node finalised head hashes don't match")
	errNoFinalizedBlock       = errors.New("did not finalise block for round")
	errNoBlockAtNumber        = errors.New("no blocks found for given number")
	errBlocksAtNumberMismatch = errors.New("different blocks found for given number")
	errChainHeadMismatch      = errors.New("node chain head hashes don't match")
)
