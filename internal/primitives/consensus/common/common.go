// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

// BlockStatus is block status.
type BlockStatus uint

const (
	// Added to the import queue.
	BlockStatusQueued BlockStatus = iota
	// Already in the blockchain and the state is available.
	BlockStatusInChainWithState
	// In the blockchain, but the state is not available.
	BlockStatusInChainPruned
	// Block or parent is known to be bad.
	BlockStatusKnownBad
	// Not in the queue or the blockchain.
	BlockStatusUnknown
)
