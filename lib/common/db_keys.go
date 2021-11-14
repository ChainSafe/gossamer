// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

var (
	// BestBlockHashKey is the db location the hash of the best (unfinalised) block header.
	BestBlockHashKey = []byte("best_hash")
	// LatestStorageHashKey is the db location of the hash of the latest storage trie.
	LatestStorageHashKey = []byte("latest_storage_hash")
	// FinalizedBlockHashKey is the db location of the hash of the latest finalised block header.
	FinalizedBlockHashKey = []byte("finalised_head")
	// GenesisDataKey is the db location of the genesis data.
	GenesisDataKey = []byte("genesis_data")
	// BlockTreeKey is the db location of the encoded block tree structure.
	BlockTreeKey = []byte("block_tree")
	// LatestFinalizedRoundKey is the key where the last finalised grandpa round is stored
	LatestFinalizedRoundKey = []byte("latest_finalised_round")
	// WorkingStorageHashKey is the storage key that the runtime uses to store the latest working state root.
	WorkingStorageHashKey = []byte("working_storage_hash")
	//NodeNameKey is the storage key to store de current node name and avoid create a new name every initialization
	NodeNameKey = []byte("node_name")
	// PruningKey is the storage key to store the current pruning configuration.
	PruningKey = []byte("prune")
	//CodeSubstitutedBlock is the storage key to store block hash of substituted (if there is currently code substituted)
	CodeSubstitutedBlock = []byte("code_substituted_block")
)
