// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

// / Block data origin.
type BlockOrigin uint

const (
	// Genesis block built into the client.
	BlockOriginGenesis BlockOrigin = iota
	// Block is part of the initial sync with the network.
	BlockOriginNetworkInitialSync
	// Block was broadcasted on the network.
	BlockOriginNetworkBroadcast
	// Block that was received from the network and validated in the consensus process.
	BlockOriginConsensusBroadcast
	// Block that was collated by this node.
	BlockOriginOwn
	// Block was imported from a file.
	BlockOriginFile
)

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
