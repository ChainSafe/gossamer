// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"encoding/json"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

type BlockImportDigestHandler interface {
	HandleDigests(*types.Header) error
}

// BlockState interface for block state methods
type BlockState interface {
	GenesisHash() common.Hash
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	AddBlock(*types.Block) error
	GetHeader(bhash common.Hash) (*types.Header, error)
	GetBlockStateRoot(bhash common.Hash) (common.Hash, error)
	RangeInMemory(start, end common.Hash) ([]common.Hash, error)
	GetBlockBody(hash common.Hash) (*types.Body, error)
	HandleRuntimeChanges(newState *rtstorage.TrieState, in runtime.Instance, bHash common.Hash) error
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
	StoreRuntime(blockHash common.Hash, runtime runtime.Instance)
	LowestCommonAncestor(a, b common.Hash) (common.Hash, error)
}

// StorageState interface for storage state methods
type StorageState interface {
	TrieState(root *common.Hash) (*rtstorage.TrieState, error)
	StoreTrie(*rtstorage.TrieState, *types.Header) error
	GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error)
	GenerateTrieProof(stateRoot common.Hash, keys [][]byte) ([][]byte, error)
	sync.Locker
}

// EpochState is the interface for state.EpochState
type EpochState interface {
	GetEpochForBlock(header *types.Header) (uint64, error)

	// GetSkippedEpochDataRaw returns the raw epoch data for a skipped epoch that is stored in advance
	// of the start of the epoch, also this method will update the epoch number from the
	// skipped epoch to the current epoch
	GetSkippedEpochDataRaw(skippedEpoch, currentEpoch uint64,
		header *types.Header) (*types.EpochDataRaw, error)

	// GetSkippedConfigData returns the config data for a skipped epoch that is stored in advance
	// of the start of the epoch, also this method will update the epoch number from the
	// skipped epoch to the current epoch
	GetSkippedConfigData(skippedEpoch, currentEpoch uint64,
		header *types.Header) (*types.ConfigData, error)
}

// GrandpaState is the interface for the state.GrandpaState
type GrandpaState interface {
	ApplyForcedChanges(importedHeader *types.Header) error
}

// TransactionState is the interface for transaction state methods
type TransactionState interface {
	Push(vt *transaction.ValidTransaction) (common.Hash, error)
	AddToPool(vt *transaction.ValidTransaction) common.Hash
	RemoveExtrinsic(ext types.Extrinsic)
	RemoveExtrinsicFromPool(ext types.Extrinsic)
	PendingInPool() []*transaction.ValidTransaction
	Exists(ext types.Extrinsic) bool
}

// Network is the interface for the network service
type Network interface {
	GossipMessage(network.NotificationsMessage)
	IsSynced() bool
	ReportPeer(change peerset.ReputationChange, p peer.ID)
}

// CodeSubstitutedState interface to handle storage of code substitute state
type CodeSubstitutedState interface {
	StoreCodeSubstitutedBlockHash(hash common.Hash) error
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}

// KeyPair is a key pair to sign messages and from which
// the public key and key type can be obtained.
type KeyPair interface {
	Type() crypto.KeyType
	Sign(msg []byte) ([]byte, error)
	Public() crypto.PublicKey
}
