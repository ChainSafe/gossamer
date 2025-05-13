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

// Block data origin.
type BlockOrigin uint

const (
	// Genesis block built into the client.
	GenesisBlockOrigin BlockOrigin = iota
	// Block is part of the initial sync with the network.
	NetworkInitialSyncBlockOrigin
	// Block was broadcasted on the network.
	NetworkBroadcastBlockOrigin
	// Block that was received from the network and validated in the consensus process.
	ConsensusBroadcastBlockOrigin
	// Block that was collated by this node.
	OwnBlockOrigin
	// Block was imported from a file.
	FileBlockOrigin
)
