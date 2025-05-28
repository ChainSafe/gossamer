// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	primitives_runtime "github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
)

const (
	blockPrefix = "block"
)

var (
	headerPrefix        = []byte("hdr") // headerPrefix + hash -> header
	blockBodyPrefix     = []byte("blb") // blockBodyPrefix + hash -> body
	headerHashPrefix    = []byte("hsh") // headerHashPrefix + encodedBlockNum -> hash
	arrivalTimePrefix   = []byte("arr") // arrivalTimePrefix || hash -> arrivalTime
	ReceiptPrefix       = []byte("rcp") // ReceiptPrefix + hash -> receipt
	MessageQueuePrefix  = []byte("mqp") // MessageQueuePrefix + hash -> message queue
	JustificationPrefix = []byte("jcp") // JustificationPrefix + hash -> justification
	firstSlotNumberKey  = []byte("fsn") // firstSlotNumberKey -> First slot number

	errNilBlockTree = errors.New("blocktree is nil")
	errNilBlockBody = errors.New("block body is nil")

	syncedBlocksGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_syncer",
		Name:      "blocks_synced_total",
		Help:      "total number of blocks synced",
	})
)

type BlockState interface {
	AddBlock(
		*types.Block,
		*overlayedchanges.OverlayedChanges[hash.H256, primitives_runtime.BlakeTwo256],
		*storage.StateVersion,
	) error
	AddBlockWithArrivalTime(block *types.Block, arrivalTime time.Time) error

	BestBlock() (*types.Block, error)
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (number uint, err error)
	GenesisHash() common.Hash

	GetBlockBody(hash common.Hash) (*types.Body, error)
	GetBlockStateRoot(bhash common.Hash) (common.Hash, error)
	GetBlockByHash(common.Hash) (*types.Block, error)
	GetBlockByNumber(blockNumber uint) (*types.Block, error)
	GetFinalisedHeader(round, setID uint64) (*types.Header, error)
	GetHashesByNumber(blockNumber uint) ([]common.Hash, error) // not sure why we need this, use `GetHashByNumber`?
	GetHashByNumber(blockNumber uint) (common.Hash, error)
	GetHeader(bhash common.Hash) (*types.Header, error)
	GetHeaderByNumber(num uint) (*types.Header, error)
	GetHighestFinalisedHeader() (*types.Header, error)
	GetHighestFinalisedHash() (common.Hash, error)
	GetHighestRoundAndSetID() (uint64, uint64, error)
	GetJustification(common.Hash) ([]byte, error)
	GetFirstNonOriginSlotNumber() (uint64, error)
	GetReceipt(hash common.Hash) ([]byte, error)
	GetMessageQueue(hash common.Hash) ([]byte, error)
	GetTries() *Tries
	GetBlockHashesBySlot(slotNum uint64) ([]common.Hash, error)
	GetAllBlocksAtNumber(num uint) ([]common.Hash, error)

	GetNonFinalisedBlocks() []common.Hash
	GetSlotForBlock(common.Hash) (uint64, error)

	HasFinalisedBlock(round, setID uint64) (bool, error)
	HasHeader(hash common.Hash) (bool, error)
	HasJustification(hash common.Hash) (bool, error)
	HasHeaderInDatabase(hash common.Hash) (bool, error)

	GetLastFinalized() common.Hash
	SetFirstNonOriginSlotNumber(slotNumber uint64) error
	SetFinalisedHash(hash common.Hash, round uint64, setID uint64, finalizeAncestors bool) error
	SetFinalizedHeader(header *types.Header) error
	SetHeader(header *types.Header) error
	SetJustification(hash common.Hash, data []byte) error
	SetHighestRoundAndSetID(round, setID uint64) error
	GetRoundAndSetID() (uint64, uint64)

	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
	UnregisterRuntimeUpdatedChannel(id uint32) bool
	HandleRuntimeChanges(newState rtstorage.TrieState, in runtime.Instance, bHash common.Hash) error

	CompareAndSetBlockData(bd *types.BlockData) error

	IsDescendantOf(parent, child common.Hash) (bool, error)
	LowestCommonAncestor(a, b common.Hash) (common.Hash, error)
	NumberIsFinalised(blockNumber uint) (bool, error)

	BlocktreeAsString() string
	Leaves() []common.Hash

	Range(startHash, endHash common.Hash) (hashes []common.Hash, err error)
	RangeInMemory(start, end common.Hash) ([]common.Hash, error)

	StoreRuntime(blockHash common.Hash, runtime runtime.Instance)

	FreeImportedBlockNotifierChannel(ch chan *types.Block)
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo

	RegisterRuntimeUpdatedChannel(ch chan<- runtime.Version) (uint32, error)

	Rewind(toBlock uint) error

	SetBlockTree(blocktree *blocktree.BlockTree)

	IsPaused() bool
	Pause() error
}

// DefaultBlockState contains the historical block data of the blockchain, including block headers and bodies.
// It wraps the blocktree (which contains unfinalised blocks) and the database (which contains finalised blocks).
type DefaultBlockState struct {
	bt                *blocktree.BlockTree
	baseState         *BaseState
	dbPath            string
	db                BlockStateDatabase
	lock              sync.RWMutex
	genesisHash       common.Hash
	lastFinalised     common.Hash
	lastRound         uint64
	lastSetID         uint64
	unfinalisedBlocks *hashToBlockMap
	tries             *Tries

	// State variables
	pausedLock sync.RWMutex
	pause      chan struct{}

	// block notifiers
	imported                       map[chan *types.Block]struct{}
	finalised                      map[chan *types.FinalisationInfo]struct{}
	finalisedLock                  sync.RWMutex
	importedLock                   sync.RWMutex
	runtimeUpdateSubscriptionsLock sync.RWMutex
	runtimeUpdateSubscriptions     map[uint32]chan<- runtime.Version

	telemetry Telemetry
}

// NewDefaultBlockState will create a new BlockState backed by the database located at basePath
func NewDefaultBlockState(db database.Database, trs *Tries, telemetry Telemetry) (*DefaultBlockState, error) {
	bs := &DefaultBlockState{
		dbPath:                     db.Path(),
		baseState:                  NewBaseState(db),
		db:                         database.NewTable(db, blockPrefix),
		unfinalisedBlocks:          newHashToBlockMap(),
		tries:                      trs,
		imported:                   make(map[chan *types.Block]struct{}),
		finalised:                  make(map[chan *types.FinalisationInfo]struct{}),
		runtimeUpdateSubscriptions: make(map[uint32]chan<- runtime.Version),
		telemetry:                  telemetry,
		pause:                      make(chan struct{}),
	}

	gh, err := bs.db.Get(headerHashKey(0))
	if err != nil {
		return nil, fmt.Errorf("cannot get block 0: %w", err)
	}
	genesisHash := common.NewHash(gh)

	header, err := bs.GetHighestFinalisedHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to get last finalised header: %w", err)
	}

	bs.genesisHash = genesisHash
	bs.lastFinalised = header.Hash()
	bs.bt = blocktree.NewBlockTreeFromRoot(header)
	return bs, nil
}

// NewDefaultBlockStateFromGenesis initialises a BlockState from a genesis header,
// saving it to the database located at basePath
func NewDefaultBlockStateFromGenesis(db database.Database, trs *Tries, header *types.Header,
	telemetryMailer Telemetry) (*DefaultBlockState, error) {
	bs := &DefaultBlockState{
		bt:                         blocktree.NewBlockTreeFromRoot(header),
		baseState:                  NewBaseState(db),
		db:                         database.NewTable(db, blockPrefix),
		unfinalisedBlocks:          newHashToBlockMap(),
		tries:                      trs,
		imported:                   make(map[chan *types.Block]struct{}),
		finalised:                  make(map[chan *types.FinalisationInfo]struct{}),
		runtimeUpdateSubscriptions: make(map[uint32]chan<- runtime.Version),
		genesisHash:                header.Hash(),
		lastFinalised:              header.Hash(),
		telemetry:                  telemetryMailer,
		pause:                      make(chan struct{}),
	}

	if err := bs.setArrivalTime(header.Hash(), time.Now()); err != nil {
		return nil, err
	}

	if err := bs.SetHeader(header); err != nil {
		return nil, err
	}

	if err := bs.db.Put(headerHashKey(uint64(header.Number)), header.Hash().ToBytes()); err != nil {
		return nil, err
	}

	if err := bs.SetBlockBody(header.Hash(), types.NewBody([]types.Extrinsic{})); err != nil {
		return nil, err
	}

	bs.genesisHash = header.Hash()
	bs.lastFinalised = header.Hash()

	if err := bs.db.Put(HighestRoundAndSetIDKey, RoundAndSetIDToBytes(0, 0)); err != nil {
		return nil, err
	}

	// set the latest finalised head to the genesis header
	if err := bs.SetFinalisedHash(bs.genesisHash, 0, 0, true); err != nil {
		return nil, err
	}

	return bs, nil
}

func NewDefaultBlockStateForStateImport(db database.Database) *DefaultBlockState {
	return &DefaultBlockState{
		bt:                blocktree.NewEmptyBlockTree(),
		db:                database.NewTable(db, blockPrefix),
		unfinalisedBlocks: newHashToBlockMap(),
	}
}

// Pause pauses the service ie. halts block production
func (bs *DefaultBlockState) Pause() error {
	bs.pausedLock.Lock()
	defer bs.pausedLock.Unlock()

	if bs.IsPaused() {
		return nil
	}

	close(bs.pause)
	return nil
}

// IsPaused returns if the service is paused or not (ie. producing blocks)
func (bs *DefaultBlockState) IsPaused() bool {
	select {
	case <-bs.pause:
		return true
	default:
		return false
	}
}

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8) // encoding results in 8 bytes
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// headerKey = headerPrefix + hash
func headerKey(hash common.Hash) []byte {
	return append(headerPrefix, hash.ToBytes()...)
}

// headerHashKey = headerHashPrefix + num (uint64 big endian)
func headerHashKey(number uint64) []byte {
	return append(headerHashPrefix, encodeBlockNumber(number)...)
}

// blockBodyKey = blockBodyPrefix + hash
func blockBodyKey(hash common.Hash) []byte {
	return append(blockBodyPrefix, hash.ToBytes()...)
}

// arrivalTimeKey = arrivalTimePrefix + hash
func arrivalTimeKey(hash common.Hash) []byte {
	return append(arrivalTimePrefix, hash.ToBytes()...)
}

// GenesisHash returns the hash of the genesis block
func (bs *DefaultBlockState) GenesisHash() common.Hash {
	return bs.genesisHash
}

// HasHeader returns true if the hash is part of the unfinalised blocks in-memory or
// persisted in the database.
func (bs *DefaultBlockState) HasHeader(hash common.Hash) (bool, error) {
	if bs.unfinalisedBlocks.getBlock(hash) != nil {
		return true, nil
	}

	return bs.db.Has(headerKey(hash))
}

// HasHeaderInDatabase returns true if the database contains a header with the given hash
func (bs *DefaultBlockState) HasHeaderInDatabase(hash common.Hash) (bool, error) {
	return bs.db.Has(headerKey(hash))
}

func (bs *DefaultBlockState) GetLastFinalized() common.Hash {
	return bs.lastFinalised
}

func (bs *DefaultBlockState) GetTries() *Tries {
	return bs.tries
}

// GetHeader returns a BlockHeader for a given hash
func (bs *DefaultBlockState) GetHeader(hash common.Hash) (header *types.Header, err error) {
	header = bs.unfinalisedBlocks.getBlockHeader(hash)
	if header != nil {
		return header, nil
	}

	if bs.db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	if has, _ := bs.HasHeader(hash); !has {
		return nil, database.ErrNotFound
	}

	data, err := bs.db.Get(headerKey(hash))
	if err != nil {
		return nil, err
	}

	result := types.NewEmptyHeader()
	err = scale.Unmarshal(data, result)
	if err != nil {
		return nil, err
	}

	if result.Empty() {
		return nil, database.ErrNotFound
	}

	result.Hash()
	return result, nil
}

// GetHashByNumber returns the block hash on our best chain with the given number
func (bs *DefaultBlockState) GetHashByNumber(num uint) (common.Hash, error) {
	hash, err := bs.bt.GetHashByNumber(num)
	if err == nil {
		return hash, nil
	} else if !errors.Is(err, blocktree.ErrNumLowerThanRoot) {
		return common.Hash{}, fmt.Errorf("failed to get hash from blocktree: %w", err)
	}

	// if error is ErrNumLowerThanRoot, number has already been finalised, so check db
	bh, err := bs.db.Get(headerHashKey(uint64(num)))
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot get block %d: %w", num, err)
	}

	return common.NewHash(bh), nil
}

// GetHashesByNumber returns the block hashes with the given number
func (bs *DefaultBlockState) GetHashesByNumber(blockNumber uint) ([]common.Hash, error) {
	inMemoryBlockHashes := bs.bt.GetHashesAtNumber(blockNumber)
	if len(inMemoryBlockHashes) == 0 {
		bh, err := bs.db.Get(headerHashKey(uint64(blockNumber)))
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return []common.Hash{}, nil
			}
			return []common.Hash{}, fmt.Errorf("cannot get block by its number %d: %w", blockNumber, err)
		}

		return []common.Hash{common.NewHash(bh)}, nil
	}

	return inMemoryBlockHashes, nil
}

// GetAllDescendants gets all the descendants for a given block hash (including itself), by first checking in memory
// and, if not found, reading from the block state database.
func (bs *DefaultBlockState) GetAllDescendants(hash common.Hash) ([]common.Hash, error) {
	allDescendants, err := bs.bt.GetAllDescendants(hash)
	if err != nil && !errors.Is(err, blocktree.ErrNodeNotFound) {
		return nil, err
	}

	if err == nil {
		return allDescendants, nil
	}

	allDescendants = []common.Hash{hash}

	header, err := bs.GetHeader(hash)
	if err != nil {
		return nil, fmt.Errorf("getting header: %w", err)
	}

	nextBlockHashes, err := bs.GetHashesByNumber(header.Number + 1)
	if err != nil {
		return nil, fmt.Errorf("getting hashes by number: %w", err)
	}

	for _, nextBlockHash := range nextBlockHashes {
		nextHeader, err := bs.GetHeader(nextBlockHash)
		if err != nil {
			return nil, fmt.Errorf("getting header from block hash %s: %w", nextBlockHash, err)
		}
		// next block is not a descendant of the block for the given hash
		if nextHeader.ParentHash != hash {
			return []common.Hash{hash}, nil
		}

		nextDescendants, err := bs.bt.GetAllDescendants(nextBlockHash)
		if err != nil && !errors.Is(err, blocktree.ErrNodeNotFound) {
			return nil, fmt.Errorf("getting all descendants: %w", err)
		}
		if err == nil {
			allDescendants = append(allDescendants, nextDescendants...)
			return allDescendants, nil
		}

		nextDescendants, err = bs.GetAllDescendants(nextBlockHash)
		if err != nil {
			return nil, err
		}

		allDescendants = append(allDescendants, nextDescendants...)
	}

	return allDescendants, nil
}

// GetBlockHashesBySlot gets all block hashes that were produced in the given slot.
func (bs *DefaultBlockState) GetBlockHashesBySlot(slotNum uint64) ([]common.Hash, error) {
	highestFinalisedHash, err := bs.GetHighestFinalisedHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get highest finalised hash: %w", err)
	}

	descendants, err := bs.GetAllDescendants(highestFinalisedHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get descendants: %w", err)
	}

	var blocksWithGivenSlot []common.Hash

	for _, desc := range descendants {
		descSlot, err := bs.GetSlotForBlock(desc)
		if errors.Is(err, types.ErrGenesisHeader) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("could not get slot for block: %w", err)
		}

		if descSlot == slotNum {
			blocksWithGivenSlot = append(blocksWithGivenSlot, desc)
		}
	}

	return blocksWithGivenSlot, nil
}

// GetHeaderByNumber returns the block header on our best chain with the given number
func (bs *DefaultBlockState) GetHeaderByNumber(num uint) (*types.Header, error) {
	hash, err := bs.GetHashByNumber(num)
	if err != nil {
		return nil, err
	}

	return bs.GetHeader(hash)
}

// GetBlockByNumber returns the block on our best chain with the given number
func (bs *DefaultBlockState) GetBlockByNumber(num uint) (*types.Block, error) {
	hash, err := bs.GetHashByNumber(num)
	if err != nil {
		return nil, err
	}

	block, err := bs.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GetBlockByHash returns a block for a given hash
func (bs *DefaultBlockState) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	bs.lock.RLock()
	defer bs.lock.RUnlock()

	block := bs.unfinalisedBlocks.getBlock(hash)
	if block != nil {
		return block, nil
	}

	header, err := bs.GetHeader(hash)
	if err != nil {
		return nil, err
	}

	blockBody, err := bs.GetBlockBody(hash)
	if err != nil {
		return nil, err
	}

	return &types.Block{Header: *header, Body: *blockBody}, nil
}

// SetHeader will set the header into DB
func (bs *DefaultBlockState) SetHeader(header *types.Header) error {
	bh, err := scale.Marshal(*header)
	if err != nil {
		return err
	}

	return bs.db.Put(headerKey(header.Hash()), bh)
}

// HasBlockBody returns true if the db contains the block body
func (bs *DefaultBlockState) HasBlockBody(hash common.Hash) (bool, error) {
	bs.lock.RLock()
	defer bs.lock.RUnlock()

	if bs.unfinalisedBlocks.getBlock(hash) != nil {
		return true, nil
	}

	return bs.db.Has(blockBodyKey(hash))
}

// GetBlockBody will return Body for a given hash
func (bs *DefaultBlockState) GetBlockBody(hash common.Hash) (body *types.Body, err error) {
	body = bs.unfinalisedBlocks.getBlockBody(hash)
	if body != nil {
		return body, nil
	}

	data, err := bs.db.Get(blockBodyKey(hash))
	if err != nil {
		return nil, err
	}

	return types.NewBodyFromBytes(data)
}

// SetBlockBody will add a block body to the db
func (bs *DefaultBlockState) SetBlockBody(hash common.Hash, body *types.Body) error {
	encodedBody, err := scale.Marshal(*body)
	if err != nil {
		return err
	}

	return bs.db.Put(blockBodyKey(hash), encodedBody)
}

// SetFirstNonOriginSlotNumber saves the first non-origin slot number into the DB
func (bs *DefaultBlockState) SetFirstNonOriginSlotNumber(slotNumber uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, slotNumber)
	return bs.db.Put(firstSlotNumberKey, buf)
}

// GetFirstNonOriginSlotNumber returns the slot number of the first non origin block
func (s *DefaultBlockState) GetFirstNonOriginSlotNumber() (uint64, error) {
	slotVal, err := s.db.Get(firstSlotNumberKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	val := binary.LittleEndian.Uint64(slotVal)
	return val, nil
}

// CompareAndSetBlockData will compare empty fields and set all elements in a block data to db
func (bs *DefaultBlockState) CompareAndSetBlockData(bd *types.BlockData) error {
	hasReceipt, _ := bs.HasReceipt(bd.Hash)
	if bd.Receipt != nil && !hasReceipt {
		err := bs.SetReceipt(bd.Hash, *bd.Receipt)
		if err != nil {
			return err
		}
	}

	hasMessageQueue, _ := bs.HasMessageQueue(bd.Hash)
	if bd.MessageQueue != nil && !hasMessageQueue {
		err := bs.SetMessageQueue(bd.Hash, *bd.MessageQueue)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddBlock adds a block to the blocktree and the DB with arrival time as current unix time
func (bs *DefaultBlockState) AddBlock(
	block *types.Block,
	_ *overlayedchanges.OverlayedChanges[hash.H256, primitives_runtime.BlakeTwo256],
	_ *storage.StateVersion,
) error {
	bs.lock.Lock()
	defer bs.lock.Unlock()
	return bs.AddBlockWithArrivalTime(block, time.Now())
}

// AddBlockWithArrivalTime adds a block to the blocktree and the DB with the given arrival time
func (bs *DefaultBlockState) AddBlockWithArrivalTime(block *types.Block, arrivalTime time.Time) error {
	if block.Body == nil {
		return errNilBlockBody
	}

	// add block to blocktree
	if err := bs.bt.AddBlock(&block.Header, arrivalTime); err != nil {
		return err
	}

	bs.unfinalisedBlocks.store(block)
	go bs.notifyImported(block)
	return nil
}

// GetAllBlocksAtNumber returns all unfinalised blocks with the given number
func (bs *DefaultBlockState) GetAllBlocksAtNumber(num uint) ([]common.Hash, error) {
	return bs.bt.GetHashesAtNumber(num), nil
}

func (bs *DefaultBlockState) isBlockOnCurrentChain(header *types.Header) (bool, error) {
	bestBlock, err := bs.BestBlockHeader()
	if err != nil {
		return false, err
	}

	// if the new block is ahead of our best block, then it is on our current chain.
	if header.Number > bestBlock.Number {
		return true, nil
	}

	is, err := bs.IsDescendantOf(header.Hash(), bestBlock.Hash())
	if err != nil {
		return false, err
	}

	if !is {
		return false, nil
	}

	return true, nil
}

// BestBlockHash returns the hash of the head of the current chain
func (bs *DefaultBlockState) BestBlockHash() common.Hash {
	if bs.bt == nil {
		return common.Hash{}
	}

	return bs.bt.BestBlockHash()
}

// BestBlockHeader returns the block header of the current head of the chain
func (bs *DefaultBlockState) BestBlockHeader() (*types.Header, error) {
	header, err := bs.GetHeader(bs.BestBlockHash())
	if err != nil {
		return nil, fmt.Errorf("cannot get header of best block: %w", err)
	}
	syncedBlocksGauge.Set(float64(header.Number))
	return header, nil
}

// BestBlockStateRoot returns the state root of the current head of the chain
func (bs *DefaultBlockState) BestBlockStateRoot() (common.Hash, error) {
	header, err := bs.BestBlockHeader()
	if err != nil {
		return common.Hash{}, err
	}

	return header.StateRoot, nil
}

// GetBlockStateRoot returns the state root of the given block hash
func (bs *DefaultBlockState) GetBlockStateRoot(bhash common.Hash) (
	hash common.Hash, err error) {
	header, err := bs.GetHeader(bhash)
	if err != nil {
		return hash, err
	}

	return header.StateRoot, nil
}

// BestBlockNumber returns the block number of the current head of the chain
func (bs *DefaultBlockState) BestBlockNumber() (blockNumber uint, err error) {
	header, err := bs.BestBlockHeader()
	if err != nil {
		return 0, err
	}

	if header == nil {
		return 0, fmt.Errorf("failed to get best block header")
	}

	return header.Number, nil
}

// BestBlock returns the current head of the chain
func (bs *DefaultBlockState) BestBlock() (*types.Block, error) {
	return bs.GetBlockByHash(bs.BestBlockHash())
}

// GetSlotForBlock returns the slot for a block
func (bs *DefaultBlockState) GetSlotForBlock(hash common.Hash) (uint64, error) {
	header, err := bs.GetHeader(hash)
	if err != nil {
		return 0, fmt.Errorf("getting header for hash %s: %w", hash, err)
	}

	return types.GetSlotFromHeader(header)
}

var ErrEmptyHeader = errors.New("empty header")

func (bs *DefaultBlockState) loadHeaderFromDatabase(hash common.Hash) (header *types.Header, err error) {
	startHeaderData, err := bs.db.Get(headerKey(hash))
	if err != nil {
		return nil, fmt.Errorf("querying database: %w", err)
	}

	header = types.NewEmptyHeader()
	err = scale.Unmarshal(startHeaderData, header)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling start header: %w", err)
	}

	if header.Empty() {
		return nil, fmt.Errorf("%w: %s", ErrEmptyHeader, hash)
	}

	return header, nil
}

// Range returns the sub-blockchain between the starting hash and the
// ending hash using both block tree and database
func (bs *DefaultBlockState) Range(startHash, endHash common.Hash) (hashes []common.Hash, err error) {
	if startHash == endHash {
		hashes = []common.Hash{startHash}
		return hashes, nil
	}

	endHeader, err := bs.loadHeaderFromDatabase(endHash)
	if errors.Is(err, database.ErrNotFound) ||
		errors.Is(err, ErrEmptyHeader) {
		// end hash is not in the database so we should lookup the
		// block that could be in memory and in the database as well
		return bs.retrieveRange(startHash, endHash)
	} else if err != nil {
		return nil, fmt.Errorf("retrieving end hash from database: %w", err)
	}

	// end hash was found in the database, that means all the blocks
	// between start and end can be found in the database
	return bs.retrieveRangeFromDatabase(startHash, endHeader)
}

func (bs *DefaultBlockState) retrieveRange(startHash, endHash common.Hash) (hashes []common.Hash, err error) {
	inMemoryHashes, err := bs.bt.Range(startHash, endHash)
	if err != nil {
		return nil, fmt.Errorf("retrieving range from in-memory blocktree: %w", err)
	}

	firstItem := inMemoryHashes[0]

	// if the first item is equal to the startHash that means we got the range
	// from the in-memory blocktree
	if firstItem == startHash {
		return inMemoryHashes, nil
	}

	// since we got as many blocks as we could from
	// the block tree but still missing blocks to
	// fulfil the range we should lookup in the
	// database for the remaining ones, the first item in the hashes array
	// must be the block tree root that is also placed in the database
	//  so we will start from its parent since it is already in the array
	blockTreeRootHeader, err := bs.loadHeaderFromDatabase(firstItem)
	if err != nil {
		return nil, fmt.Errorf("loading block tree root from database: %w", err)
	}

	startingAtParentHeader, err := bs.loadHeaderFromDatabase(blockTreeRootHeader.ParentHash)
	if err != nil {
		return nil, fmt.Errorf("loading header of parent of the root from database: %w", err)
	}

	inDatabaseHashes, err := bs.retrieveRangeFromDatabase(startHash, startingAtParentHeader)
	if err != nil {
		return nil, fmt.Errorf("retrieving range from database: %w", err)
	}

	return append(inDatabaseHashes, inMemoryHashes...), nil
}

var ErrStartHashMismatch = errors.New("start hash mismatch")
var ErrStartGreaterThanEnd = errors.New("start greater than end")

// retrieveRangeFromDatabase takes the start and the end and will retrieve all block in between
// where all blocks (start and end inclusive) are supposed to be placed at database
func (bs *DefaultBlockState) retrieveRangeFromDatabase(startHash common.Hash,
	endHeader *types.Header) (hashes []common.Hash, err error) {
	startHeader, err := bs.loadHeaderFromDatabase(startHash)
	if err != nil {
		return nil, fmt.Errorf("range start should be in database: %w", err)
	}

	if startHeader.Number > endHeader.Number {
		return nil, fmt.Errorf("%w", ErrStartGreaterThanEnd)
	}

	// blocksInRange is the difference between the end number to start number
	// but the difference doesn't include the start item so we add 1
	blocksInRange := endHeader.Number - startHeader.Number + 1

	hashes = make([]common.Hash, blocksInRange)

	lastPosition := blocksInRange - 1

	inLoopHash := endHeader.Hash()
	for currentPosition := int(lastPosition); currentPosition >= 0; currentPosition-- {
		hashes[currentPosition] = inLoopHash

		inLoopHeader, err := bs.loadHeaderFromDatabase(inLoopHash)
		if err != nil {
			return nil, fmt.Errorf("retrieving hash %s from database: %w", inLoopHash.Short(), err)
		}

		inLoopHash = inLoopHeader.ParentHash
	}

	// here we ensure that we finished up the loop with the hash we used as start
	// with the same hash as the startHash
	if hashes[0] != startHash {
		return nil, fmt.Errorf("%w: expecting %s, found: %s", ErrStartHashMismatch, startHash.Short(), inLoopHash.Short())
	}

	return hashes, nil
}

// RangeInMemory returns the sub-blockchain between the starting hash and the ending hash using the block tree
func (bs *DefaultBlockState) RangeInMemory(start, end common.Hash) ([]common.Hash, error) {
	if bs.bt == nil {
		return nil, fmt.Errorf("%w", errNilBlockTree)
	}

	return bs.bt.RangeInMemory(start, end)
}

// IsDescendantOf returns true if child is a descendant of parent, false otherwise.
// it returns an error if parent or child are not in the blocktree.
func (bs *DefaultBlockState) IsDescendantOf(ancestor, descendant common.Hash) (bool, error) {
	if bs.bt == nil {
		return false, fmt.Errorf("%w", errNilBlockTree)
	}

	isDescendant, err := bs.bt.IsDescendantOf(ancestor, descendant)
	if err != nil {
		descendantHeader, err2 := bs.GetHeader(descendant)
		if err2 != nil {
			return false, fmt.Errorf("getting header: %w", err2)
		}

		ancestorHeader, err2 := bs.GetHeader(ancestor)
		if err2 != nil {
			return false, fmt.Errorf("getting header: %w", err2)
		}

		for current := descendantHeader; current.Number > ancestorHeader.Number; {
			if current.ParentHash == ancestor {
				return true, nil
			}
			current, err2 = bs.GetHeader(current.ParentHash)
			if err2 != nil {
				return false, fmt.Errorf("getting header: %w", err2)
			}
		}

		return false, nil
	}

	return isDescendant, nil
}

// LowestCommonAncestor returns the lowest common ancestor between two blocks in the tree.
func (bs *DefaultBlockState) LowestCommonAncestor(a, b common.Hash) (common.Hash, error) {
	return bs.bt.LowestCommonAncestor(a, b)
}

// Leaves returns the leaves of the blocktree as an array
func (bs *DefaultBlockState) Leaves() []common.Hash {
	return bs.bt.Leaves()
}

// BlocktreeAsString returns the blocktree as a string
func (bs *DefaultBlockState) BlocktreeAsString() string {
	return bs.bt.String()
}

// GetArrivalTime returns the arrival time in nanoseconds since the Unix epoch of a block given its hash
func (bs *DefaultBlockState) GetArrivalTime(hash common.Hash) (time.Time, error) {
	at, err := bs.bt.GetArrivalTime(hash)
	if err == nil {
		return at, nil
	}

	arrivalTime, err := bs.db.Get(arrivalTimeKey(hash))
	if err != nil {
		return time.Time{}, err
	}

	ns := binary.LittleEndian.Uint64(arrivalTime)
	return time.Unix(0, int64(ns)), nil
}

func (bs *DefaultBlockState) setArrivalTime(hash common.Hash, arrivalTime time.Time) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(arrivalTime.UnixNano()))
	return bs.db.Put(arrivalTimeKey(hash), buf)
}

// HandleRuntimeChanges handles the update in runtime.
func (bs *DefaultBlockState) HandleRuntimeChanges(newState rtstorage.TrieState,
	parentRuntimeInstance runtime.Instance, bHash common.Hash) error {
	currCodeHash, err := newState.LoadCodeHash()
	if err != nil {
		return err
	}

	parentCodeHash := parentRuntimeInstance.GetCodeHash()

	// if the parent code hash is the same as the new code hash
	// we do nothing since we don't want to store duplicate runtimes
	// for different hashes
	if bytes.Equal(parentCodeHash[:], currCodeHash[:]) {
		return nil
	}

	logger.Infof("🔄 detected runtime code change, upgrading with block %s from previous code hash %s to new code hash %s...", //nolint:lll
		bHash, parentCodeHash, currCodeHash)
	code := newState.LoadCode()
	if len(code) == 0 {
		return errors.New("new :code is empty")
	}

	codeSubBlockHash := bs.baseState.LoadCodeSubstitutedBlockHash()

	if codeSubBlockHash != (common.Hash{}) {
		newVersion, err := wazero_runtime.GetRuntimeVersion(code)
		if err != nil {
			return err
		}

		// only update runtime during code substitution if runtime SpecVersion is updated
		previousVersion, err := parentRuntimeInstance.Version()
		if err != nil {
			return err
		}

		if previousVersion.SpecVersion == newVersion.SpecVersion {
			logger.Info("not upgrading runtime code during code substitution")
			bs.StoreRuntime(bHash, parentRuntimeInstance)
			return nil
		}

		logger.Infof(
			"🔄 detected runtime code change, upgrading with block %s from previous code hash %s and spec %d to new code hash %s and spec %d...", //nolint:lll
			bHash, parentCodeHash, previousVersion.SpecVersion, currCodeHash, newVersion.SpecVersion)
	}

	rtCfg := wazero_runtime.Config{
		Storage:     newState,
		Keystore:    parentRuntimeInstance.Keystore(),
		NodeStorage: parentRuntimeInstance.NodeStorage(),
		Network:     parentRuntimeInstance.NetworkService(),
		CodeHash:    currCodeHash,
	}

	if parentRuntimeInstance.Validator() {
		rtCfg.Role = 4
	}

	instance, err := wazero_runtime.NewInstance(code, rtCfg)
	if err != nil {
		return err
	}

	bs.StoreRuntime(bHash, instance)

	err = bs.baseState.StoreCodeSubstitutedBlockHash(common.Hash{})
	if err != nil {
		return fmt.Errorf("failed to update code substituted block hash: %w", err)
	}

	newVersion, err := instance.Version()
	if err != nil {
		return err
	}
	go bs.notifyRuntimeUpdated(newVersion)
	return nil
}

// GetRuntime gets the runtime instance pointer for the block hash given.
func (bs *DefaultBlockState) GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error) {
	// we search primarily in the blocktree so we ensure the
	// fork aware property while searching for a runtime, however
	// if there is no runtimes in that fork then we look for the
	// closest ancestor with a runtime instance
	runtimeInstance, err := bs.bt.GetBlockRuntime(blockHash)

	if err != nil {
		// in this case the node is not in the blocktree which mean
		// it is a finalized node already persisted in database
		if errors.Is(err, blocktree.ErrNodeNotFound) {
			panic(err.Error() + " see https://github.com/ChainSafe/gossamer/issues/3066")
		}

		return nil, fmt.Errorf("while getting runtime: %w", err)
	}

	return runtimeInstance, nil
}

// StoreRuntime stores the runtime for corresponding block hash.
func (bs *DefaultBlockState) StoreRuntime(hash common.Hash, rt runtime.Instance) {
	bs.bt.StoreRuntime(hash, rt)
}

// GetNonFinalisedBlocks get all the blocks in the blocktree
func (bs *DefaultBlockState) GetNonFinalisedBlocks() []common.Hash {
	return bs.bt.GetAllBlocks()
}

// SetBlockTree sets the blocktree for the block state
// WARN: this should be used only when state sync finishes and we need to set the new state to resume the node using a
// specific blocktree
func (bs *DefaultBlockState) SetBlockTree(blocktree *blocktree.BlockTree) {
	bs.bt = blocktree
}

func (bs *DefaultBlockState) Rewind(toBlock uint) error {
	num, _ := bs.BestBlockNumber()
	if toBlock > num {
		return fmt.Errorf("cannot rewind, given height is higher than our current height")
	}

	logger.Infof(
		"rewinding state from current height %s to desired height %d...",
		num, toBlock)

	root, err := bs.GetBlockByNumber(toBlock)
	if err != nil {
		return err
	}

	bs.bt = blocktree.NewBlockTreeFromRoot(&root.Header)

	header, err := bs.BestBlockHeader()
	if err != nil {
		return err
	}

	bs.lastFinalised = header.Hash()

	logger.Infof(
		"rewinding state for new height %s and best block hash %s...",
		header.Number, header.Hash())

	return nil
}
