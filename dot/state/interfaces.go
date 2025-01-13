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

	BestBlock() (*types.Block, error) // can be achieved by calling blockchainDB.Info() fulfills blockchain.HeaderBackend
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (number uint, err error)
	BestBlockStateRoot() (common.Hash, error) // only used within dot/state; can be removed
	GenesisHash() common.Hash

	GetAllDescendants(hash common.Hash) ([]common.Hash, error) // only used within dot/state; can be removed
	GetAllBlocksAtNumber(num uint) ([]common.Hash, error)      // can be removed
	GetBlockBody(hash common.Hash) (*types.Body, error)
	GetBlockStateRoot(bhash common.Hash) (common.Hash, error)
	GetBlockByHash(common.Hash) (*types.Block, error)
	GetBlockByNumber(blockNumber uint) (*types.Block, error)
	GetBlockHashesBySlot(slot uint64) (blockHashes []common.Hash, err error) // can be removed
	GetFinalisedHeader(round, setID uint64) (*types.Header, error)
	GetFinalisedHash(round, setID uint64) (common.Hash, error)
	GetHashesByNumber(blockNumber uint) ([]common.Hash, error) // not sure why we need this, use `GetHashByNumber`?
	GetHashByNumber(blockNumber uint) (common.Hash, error)     // can be retrieved by from HeaderBackend.Hash()
	GetHeader(bhash common.Hash) (*types.Header, error)
	GetHeaderByNumber(num uint) (*types.Header, error)
	GetHighestFinalisedHeader() (*types.Header, error)
	GetHighestFinalisedHash() (common.Hash, error)
	GetHighestRoundAndSetID() (uint64, uint64, error)
	GetJustification(common.Hash) ([]byte, error)

	GetNonFinalisedBlocks() []common.Hash // can probably be achieved by getting last finalized, then traversing children
	GetRoundAndSetID() (uint64, uint64)   // not called, can be removed
	GetSlotForBlock(common.Hash) (uint64, error)

	HasFinalisedBlock(round, setID uint64) (bool, error)
	HasHeader(hash common.Hash) (bool, error)
	HasJustification(hash common.Hash) (bool, error)
	HasHeaderInDatabase(hash common.Hash) (bool, error) // can probably just use `HasHeader`

	SetFinalisedHash(hash common.Hash, round uint64, setID uint64) error
	SetHeader(header *types.Header) error
	SetJustification(hash common.Hash, data []byte) error

	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)                          // can be achieved by retrieving digest from block and decoding
	HandleRuntimeChanges(newState *rtstorage.TrieState, in runtime.Instance, bHash common.Hash) error // will have to be recreated

	CompareAndSetBlockData(bd *types.BlockData) error
	SetReceipt(hash common.Hash, data []byte) error // aux store functionality?
	HasReceipt(hash common.Hash) (bool, error)
	GetReceipt(common.Hash) ([]byte, error)              // pretty sure this is just block data?
	HasMessageQueue(hash common.Hash) (bool, error)      // won't be needed if CompareAndSetBlockData is replaced
	SetMessageQueue(hash common.Hash, data []byte) error // won't be needed if CompareAndSetBlockData is replaced
	GetMessageQueue(common.Hash) ([]byte, error)         // won't be needed if CompareAndSetBlockData is replaced

	IsDescendantOf(parent, child common.Hash) (bool, error)     // use client/api/utils.IsDescendantOf
	LowestCommonAncestor(a, b common.Hash) (common.Hash, error) // use primitives/blockchain.LowestCommonAncestor
	Leaves() []common.Hash                                      // don't think this is called
	NumberIsFinalised(blockNumber uint) (bool, error)

	Range(startHash, endHash common.Hash) (hashes []common.Hash, err error) // should be able to use primitives/blockchain.TreeRoute
	RangeInMemory(start, end common.Hash) ([]common.Hash, error)            // should be able to use primitives/blockchain.TreeRoute

	StoreRuntime(blockHash common.Hash, runtime runtime.Instance)

	FreeImportedBlockNotifierChannel(ch chan *types.Block)
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo

	RegisterRuntimeUpdatedChannel(ch chan<- runtime.Version) (uint32, error) // will need extra work to listen on import events and pass along.  Since it's RPC related we can deprioritize

	IsPaused() bool
	Pause() error
}

var _ blockState = &BlockState{}

type storageState interface {
	TrieState(root *common.Hash) (*rtstorage.TrieState, error)
	// rtstorage.TrieState is where all the mutations are held in 'storageDiffs"
	// TrieState is called like 90% of the time just to pass it into rt.SetContextStorage
	// TrieState has a Trie interface attribute where it's just used to get the hash
	// TrieState also is the type that the runtime host functions interact with to query/store storage changes,
	// iterate through top trie and child trie keys, start and rollback transactions

	// impl: recreate sp_statemachine.OverlayedChanges
	// call backend.StateAt() to get current state, pass in overlayed changes to handle changes
	// create an interface that represents the public API of TrieState
	// fulfill that interface with something that combines TrieBackend and OverlayedCHanges for a given hash

	// substrate notes:
	// sp_externalities::Externalities is the trait that represents the host functionality required by the runtime
	// sp_statemachine::Ext is type that implements sp_externalities::Externalities which raps Overlayed changed, read only backend
	// the #[runtime_interface] macro is used to wrap traits with default implementations where `self` is `sp_statemachine::Ext`

	StoreTrie(*rtstorage.TrieState, *types.Header) error
	// StoreTrie should be deprecated, since it's only used in dot/core/service.go in handleBlock function
	// The trie backend will be updated by adding the block to the client

	GenerateTrieProof(stateRoot common.Hash, keys [][]byte) ([][]byte, error)
	// only used by RPC at the moment, will need to recreate sp_api::CallExecutor::contextual_call
	// which will read the keys and generate a proof.

	GetStorage(root *common.Hash, key []byte) ([]byte, error)
	GetStorageByBlockHash(bhash *common.Hash, key []byte) ([]byte, error)
	GetKeysWithPrefix(root *common.Hash, prefix []byte) ([][]byte, error)
	GetStorageChild(root *common.Hash, keyToChild []byte) (trie.Trie, error)
	GetStorageFromChild(root *common.Hash, keyToChild, key []byte) ([]byte, error)
	GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error)
	StorageRoot() (common.Hash, error) // best block state root, use HeaderBackend

	Entries(root *common.Hash) (map[string][]byte, error) // should be overhauled to iterate through the keys instead of loading everything into memory

	LoadCode(hash *common.Hash) ([]byte, error)
	LoadCodeHash(hash *common.Hash) (common.Hash, error)

	RegisterStorageObserver(o Observer)
	UnregisterStorageObserver(o Observer)

	sync.Locker
}

var _ storageState = &InmemoryStorageState{}

// look into rtstroage.TrieState, you will need to make this an interface
