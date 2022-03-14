// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

//go:generate mockgen -destination=mock_core_test.go -package $GOPACKAGE . BlockState,StorageState,TransactionState,Network,EpochState,CodeSubstitutedState,DigestHandler

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (blockNumber uint, err error)
	BestBlockStateRoot() (common.Hash, error)
	BestBlock() (*types.Block, error)
	AddBlock(*types.Block) error
	GetAllBlocksAtDepth(hash common.Hash) []common.Hash
	GetBlockByHash(common.Hash) (*types.Block, error)
	GetBlockStateRoot(bhash common.Hash) (common.Hash, error)
	GenesisHash() common.Hash
	GetSlotForBlock(common.Hash) (uint64, error)
	GetFinalisedHeader(uint64, uint64) (*types.Header, error)
	GetFinalisedHash(uint64, uint64) (common.Hash, error)
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeImportedBlockNotifierChannel(ch chan *types.Block)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo
	FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo)
	HighestCommonAncestor(a, b common.Hash) (common.Hash, error)
	SubChain(start, end common.Hash) ([]common.Hash, error)
	GetBlockBody(hash common.Hash) (*types.Body, error)
	HandleRuntimeChanges(newState *rtstorage.TrieState, in runtime.Instance, bHash common.Hash) error
	GetRuntime(*common.Hash) (runtime.Instance, error)
	StoreRuntime(common.Hash, runtime.Instance)
}

// StorageState interface for storage state methods
type StorageState interface {
	LoadCode(root *common.Hash) ([]byte, error)
	LoadCodeHash(root *common.Hash) (common.Hash, error)
	TrieState(root *common.Hash) (*rtstorage.TrieState, error)
	StoreTrie(*rtstorage.TrieState, *types.Header) error
	GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error)
	GetStorage(root *common.Hash, key []byte) ([]byte, error)
	GenerateTrieProof(stateRoot common.Hash, keys [][]byte) ([][]byte, error)
	sync.Locker
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

// EpochState is the interface for state.EpochState
type EpochState interface {
	GetEpochForBlock(header *types.Header) (uint64, error)
	SetCurrentEpoch(epoch uint64) error
	GetCurrentEpoch() (uint64, error)
}

// CodeSubstitutedState interface to handle storage of code substitute state
type CodeSubstitutedState interface {
	LoadCodeSubstitutedBlockHash() common.Hash
	StoreCodeSubstitutedBlockHash(hash common.Hash) error
}

// DigestHandler is the interface for the consensus digest handler
type DigestHandler interface {
	HandleDigests(header *types.Header)
}
