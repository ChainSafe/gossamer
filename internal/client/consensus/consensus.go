// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package consensus

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
