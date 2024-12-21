// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

type GrandpaDatabase interface {
	GetPutDeleter
	NewPrefixIterator(prefix []byte) (database.Iterator, error)
}

// GetPutDeleter has methods to get, put and delete key values.
type GetPutDeleter interface {
	GetPutter
	Deleter
}

// BlockStateDatabase is the database interface for the block state.
type BlockStateDatabase interface {
	GetPutDeleter
	Haser
	NewBatcher
}

// GetPutter has methods to get and put key values.
type GetPutter interface {
	Getter
	Putter
}

// GetterPutterNewBatcher has methods to get values and create a
// new batch.
type GetterPutterNewBatcher interface {
	Getter
	Putter
	NewBatcher
}

// Getter gets a value corresponding to the given key.
type Getter interface {
	Get(key []byte) (value []byte, err error)
}

// Putter puts a value at the given key and returns an error.
type Putter interface {
	Put(key []byte, value []byte) error
}

// Deleter deletes a value at the given key and returns an error.
type Deleter interface {
	Del(key []byte) error
}

// Haser checks if a value exists at the given key and returns an error.
type Haser interface {
	Has(key []byte) (has bool, err error)
}

// NewBatcher creates a new database batch.
type NewBatcher interface {
	NewBatch() database.Batch
}

// BabeConfigurer returns the babe configuration of the runtime.
type BabeConfigurer interface {
	BabeConfiguration() (*types.BabeConfiguration, error)
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}

type blockState interface {
	AddBlock(*types.Block) error
	AddBlockWithArrivalTime(block *types.Block, arrivalTime time.Time) error
	BestBlock() (*types.Block, error)
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (number uint, err error)
	BestBlockStateRoot() (common.Hash, error)
	CompareAndSetBlockData(bd *types.BlockData) error
	GenesisHash() common.Hash

	GetAllDescendants(hash common.Hash) ([]common.Hash, error)
	GetAllBlocksAtNumber(num uint) ([]common.Hash, error)
	GetBlockBody(hash common.Hash) (*types.Body, error)
	GetBlockStateRoot(bhash common.Hash) (common.Hash, error)
	GetBlockByHash(common.Hash) (*types.Block, error)
	GetBlockByNumber(blockNumber uint) (*types.Block, error)
	GetBlockHashesBySlot(slot uint64) (blockHashes []common.Hash, err error)
	GetFinalisedHeader(round, setID uint64) (*types.Header, error)
	GetFinalisedHash(round, setID uint64) (common.Hash, error)

	GetHashesByNumber(blockNumber uint) ([]common.Hash, error)
	GetHashByNumber(blockNumber uint) (common.Hash, error)
	GetHeader(bhash common.Hash) (*types.Header, error)
	GetHeaderByNumber(num uint) (*types.Header, error)
	GetHighestFinalisedHeader() (*types.Header, error)
	GetHighestFinalisedHash() (common.Hash, error)
	GetHighestRoundAndSetID() (uint64, uint64, error)
	GetJustification(common.Hash) ([]byte, error)
	GetMessageQueue(common.Hash) ([]byte, error)
	GetNonFinalisedBlocks() []common.Hash
	GetReceipt(common.Hash) ([]byte, error)
	GetRoundAndSetID() (uint64, uint64)
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
	GetSlotForBlock(common.Hash) (uint64, error)
	HandleRuntimeChanges(newState *rtstorage.TrieState, in runtime.Instance, bHash common.Hash) error
	HasFinalisedBlock(round, setID uint64) (bool, error)
	HasHeader(hash common.Hash) (bool, error)
	HasJustification(hash common.Hash) (bool, error)
	HasHeaderInDatabase(hash common.Hash) (bool, error)

	HasMessageQueue(hash common.Hash) (bool, error)
	SetMessageQueue(hash common.Hash, data []byte) error

	IsDescendantOf(parent, child common.Hash) (bool, error)
	LowestCommonAncestor(a, b common.Hash) (common.Hash, error)
	Leaves() []common.Hash
	NumberIsFinalised(blockNumber uint) (bool, error)
	Range(startHash, endHash common.Hash) (hashes []common.Hash, err error)
	RangeInMemory(start, end common.Hash) ([]common.Hash, error)
	SetFinalisedHash(hash common.Hash, round uint64, setID uint64) error
	SetHeader(header *types.Header) error
	SetJustification(hash common.Hash, data []byte) error
	SetReceipt(hash common.Hash, data []byte) error
	HasReceipt(hash common.Hash) (bool, error)
	StoreRuntime(blockHash common.Hash, runtime runtime.Instance)

	FreeImportedBlockNotifierChannel(ch chan *types.Block)
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo

	RegisterRuntimeUpdatedChannel(ch chan<- runtime.Version) (uint32, error)

	IsPaused() bool
	Pause() error
}

var _ blockState = &BlockState{}

type storageState interface {
	TrieState(root *common.Hash) (*rtstorage.TrieState, error)
	StoreTrie(*rtstorage.TrieState, *types.Header) error
	GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error)
	GenerateTrieProof(stateRoot common.Hash, keys [][]byte) ([][]byte, error)
	GetStorage(root *common.Hash, key []byte) ([]byte, error)
	GetStorageByBlockHash(bhash *common.Hash, key []byte) ([]byte, error)
	StorageRoot() (common.Hash, error)
	Entries(root *common.Hash) (map[string][]byte, error)
	GetKeysWithPrefix(root *common.Hash, prefix []byte) ([][]byte, error)
	GetStorageChild(root *common.Hash, keyToChild []byte) (trie.Trie, error)
	GetStorageFromChild(root *common.Hash, keyToChild, key []byte) ([]byte, error)
	LoadCode(hash *common.Hash) ([]byte, error)
	LoadCodeHash(hash *common.Hash) (common.Hash, error)
	RegisterStorageObserver(o Observer)
	UnregisterStorageObserver(o Observer)

	sync.Locker
}

var _ storageState = &InmemoryStorageState{}

// look into rtstroage.TrieState, you will need to make this an interface
