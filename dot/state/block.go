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

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

const (
	blockPrefix = "block"
)

var (
	headerPrefix        = []byte("hdr") // headerPrefix + hash -> header
	blockBodyPrefix     = []byte("blb") // blockBodyPrefix + hash -> body
	headerHashPrefix    = []byte("hsh") // headerHashPrefix + encodedBlockNum -> hash
	arrivalTimePrefix   = []byte("arr") // arrivalTimePrefix || hash -> arrivalTime
	receiptPrefix       = []byte("rcp") // receiptPrefix + hash -> receipt
	messageQueuePrefix  = []byte("mqp") // messageQueuePrefix + hash -> message queue
	justificationPrefix = []byte("jcp") // justificationPrefix + hash -> justification

	errNilBlockBody = errors.New("block body is nil")

	syncedBlocksGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_syncer",
		Name:      "blocks_synced_total",
		Help:      "total number of blocks synced",
	})
)

// BlockState contains the historical block data of the blockchain, including block headers and bodies.
// It wraps the blocktree (which contains unfinalised blocks) and the database (which contains finalised blocks).
type BlockState struct {
	bt        *blocktree.BlockTree
	baseState *BaseState
	dbPath    string
	db        chaindb.Database
	sync.RWMutex
	genesisHash       common.Hash
	lastFinalised     common.Hash
	unfinalisedBlocks *hashToBlockMap
	tries             *Tries
	storageState      *StorageState

	// block notifiers
	imported                       map[chan *types.Block]struct{}
	finalised                      map[chan *types.FinalisationInfo]struct{}
	finalisedLock                  sync.RWMutex
	importedLock                   sync.RWMutex
	runtimeUpdateSubscriptionsLock sync.RWMutex
	runtimeUpdateSubscriptions     map[uint32]chan<- runtime.Version

	telemetry telemetry.Client
}

// NewBlockState will create a new BlockState backed by the database located at basePath
func NewBlockState(db chaindb.Database, trs *Tries, telemetry telemetry.Client, storageState *StorageState) (*BlockState, error) {
	bs := &BlockState{
		dbPath:                     db.Path(),
		baseState:                  NewBaseState(db),
		db:                         chaindb.NewTable(db, blockPrefix),
		unfinalisedBlocks:          newHashToBlockMap(),
		tries:                      trs,
		storageState:               storageState,
		imported:                   make(map[chan *types.Block]struct{}),
		finalised:                  make(map[chan *types.FinalisationInfo]struct{}),
		runtimeUpdateSubscriptions: make(map[uint32]chan<- runtime.Version),
		telemetry:                  telemetry,
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

// NewBlockStateFromGenesis initialises a BlockState from a genesis header,
// saving it to the database located at basePath
func NewBlockStateFromGenesis(db chaindb.Database, trs *Tries, header *types.Header,
	telemetryMailer telemetry.Client) (*BlockState, error) {
	bs := &BlockState{
		bt:                         blocktree.NewBlockTreeFromRoot(header),
		baseState:                  NewBaseState(db),
		db:                         chaindb.NewTable(db, blockPrefix),
		unfinalisedBlocks:          newHashToBlockMap(),
		tries:                      trs,
		imported:                   make(map[chan *types.Block]struct{}),
		finalised:                  make(map[chan *types.FinalisationInfo]struct{}),
		runtimeUpdateSubscriptions: make(map[uint32]chan<- runtime.Version),
		genesisHash:                header.Hash(),
		lastFinalised:              header.Hash(),
		telemetry:                  telemetryMailer,
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

	if err := bs.db.Put(highestRoundAndSetIDKey, roundAndSetIDToBytes(0, 0)); err != nil {
		return nil, err
	}

	// set the latest finalised head to the genesis header
	if err := bs.SetFinalisedHash(bs.genesisHash, 0, 0); err != nil {
		return nil, err
	}

	return bs, nil
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
func (bs *BlockState) GenesisHash() common.Hash {
	return bs.genesisHash
}

// HasHeader returns true if the hash is part of the unfinalised blocks in-memory or
// persisted in the database.
func (bs *BlockState) HasHeader(hash common.Hash) (bool, error) {
	if bs.unfinalisedBlocks.getBlock(hash) != nil {
		return true, nil
	}

	return bs.db.Has(headerKey(hash))
}

// HasHeaderInDatabase returns true if the database contains a header with the given hash
func (bs *BlockState) HasHeaderInDatabase(hash common.Hash) (bool, error) {
	return bs.db.Has(headerKey(hash))
}

// GetHeader returns a BlockHeader for a given hash
func (bs *BlockState) GetHeader(hash common.Hash) (header *types.Header, err error) {
	header = bs.unfinalisedBlocks.getBlockHeader(hash)
	if header != nil {
		return header, nil
	}

	if bs.db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	if has, _ := bs.HasHeader(hash); !has {
		return nil, chaindb.ErrKeyNotFound
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
		return nil, chaindb.ErrKeyNotFound
	}

	result.Hash()
	return result, nil
}

// GetHashByNumber returns the block hash on our best chain with the given number
func (bs *BlockState) GetHashByNumber(num uint) (common.Hash, error) {
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

// GetHeaderByNumber returns the block header on our best chain with the given number
func (bs *BlockState) GetHeaderByNumber(num uint) (*types.Header, error) {
	hash, err := bs.GetHashByNumber(num)
	if err != nil {
		return nil, err
	}

	return bs.GetHeader(hash)
}

// GetBlockByNumber returns the block on our best chain with the given number
func (bs *BlockState) GetBlockByNumber(num uint) (*types.Block, error) {
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
func (bs *BlockState) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	bs.RLock()
	defer bs.RUnlock()

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
func (bs *BlockState) SetHeader(header *types.Header) error {
	bh, err := scale.Marshal(*header)
	if err != nil {
		return err
	}

	return bs.db.Put(headerKey(header.Hash()), bh)
}

// HasBlockBody returns true if the db contains the block body
func (bs *BlockState) HasBlockBody(hash common.Hash) (bool, error) {
	bs.RLock()
	defer bs.RUnlock()

	if bs.unfinalisedBlocks.getBlock(hash) != nil {
		return true, nil
	}

	return bs.db.Has(blockBodyKey(hash))
}

// GetBlockBody will return Body for a given hash
func (bs *BlockState) GetBlockBody(hash common.Hash) (body *types.Body, err error) {
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
func (bs *BlockState) SetBlockBody(hash common.Hash, body *types.Body) error {
	encodedBody, err := scale.Marshal(*body)
	if err != nil {
		return err
	}

	return bs.db.Put(blockBodyKey(hash), encodedBody)
}

// CompareAndSetBlockData will compare empty fields and set all elements in a block data to db
func (bs *BlockState) CompareAndSetBlockData(bd *types.BlockData) error {
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
func (bs *BlockState) AddBlock(block *types.Block) error {
	bs.Lock()
	defer bs.Unlock()
	return bs.AddBlockWithArrivalTime(block, time.Now())
}

// AddBlockWithArrivalTime adds a block to the blocktree and the DB with the given arrival time
func (bs *BlockState) AddBlockWithArrivalTime(block *types.Block, arrivalTime time.Time) error {
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

// AddBlockToBlockTree adds the given block to the blocktree. It does not write it to the database.
// TODO: remove this func and usage from sync (after sync refactor?)
func (bs *BlockState) AddBlockToBlockTree(block *types.Block) error {
	bs.Lock()
	defer bs.Unlock()

	arrivalTime, err := bs.GetArrivalTime(block.Header.Hash())
	if err != nil {
		arrivalTime = time.Now()
	}

	bs.unfinalisedBlocks.store(block)
	return bs.bt.AddBlock(&block.Header, arrivalTime)
}

// GetAllBlocksAtNumber returns all unfinalised blocks with the given number
func (bs *BlockState) GetAllBlocksAtNumber(num uint) ([]common.Hash, error) {
	header, err := bs.GetHeaderByNumber(num)
	if err != nil {
		return nil, err
	}

	return bs.GetAllBlocksAtDepth(header.ParentHash), nil
}

// GetAllBlocksAtDepth returns all hashes with the depth of the given hash plus one
func (bs *BlockState) GetAllBlocksAtDepth(hash common.Hash) []common.Hash {
	return bs.bt.GetAllBlocksAtNumber(hash)
}

func (bs *BlockState) isBlockOnCurrentChain(header *types.Header) (bool, error) {
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
func (bs *BlockState) BestBlockHash() common.Hash {
	if bs.bt == nil {
		return common.Hash{}
	}

	return bs.bt.BestBlockHash()
}

// BestBlockHeader returns the block header of the current head of the chain
func (bs *BlockState) BestBlockHeader() (*types.Header, error) {
	header, err := bs.GetHeader(bs.BestBlockHash())
	if err != nil {
		return nil, fmt.Errorf("cannot get header of best block: %w", err)
	}
	syncedBlocksGauge.Set(float64(header.Number))
	return header, nil
}

// BestBlockStateRoot returns the state root of the current head of the chain
func (bs *BlockState) BestBlockStateRoot() (common.Hash, error) {
	header, err := bs.BestBlockHeader()
	if err != nil {
		return common.Hash{}, err
	}

	return header.StateRoot, nil
}

// GetBlockStateRoot returns the state root of the given block hash
func (bs *BlockState) GetBlockStateRoot(bhash common.Hash) (
	hash common.Hash, err error) {
	header, err := bs.GetHeader(bhash)
	if err != nil {
		return hash, err
	}

	return header.StateRoot, nil
}

// BestBlockNumber returns the block number of the current head of the chain
func (bs *BlockState) BestBlockNumber() (blockNumber uint, err error) {
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
func (bs *BlockState) BestBlock() (*types.Block, error) {
	return bs.GetBlockByHash(bs.BestBlockHash())
}

// GetSlotForBlock returns the slot for a block
func (bs *BlockState) GetSlotForBlock(hash common.Hash) (uint64, error) {
	header, err := bs.GetHeader(hash)
	if err != nil {
		return 0, err
	}

	return types.GetSlotFromHeader(header)
}

// SubChain returns the sub-blockchain between the starting hash and the ending hash using the block tree
func (bs *BlockState) SubChain(start, end common.Hash) ([]common.Hash, error) {
	if bs.bt == nil {
		return nil, fmt.Errorf("blocktree is nil")
	}

	return bs.bt.SubBlockchain(start, end)
}

// IsDescendantOf returns true if child is a descendant of parent, false otherwise.
// it returns an error if parent or child are not in the blocktree.
func (bs *BlockState) IsDescendantOf(parent, child common.Hash) (bool, error) {
	if bs.bt == nil {
		return false, fmt.Errorf("blocktree is nil")
	}

	return bs.bt.IsDescendantOf(parent, child)
}

// HighestCommonAncestor returns the block with the highest number that is an ancestor of both a and b
func (bs *BlockState) HighestCommonAncestor(a, b common.Hash) (common.Hash, error) {
	return bs.bt.HighestCommonAncestor(a, b)
}

// Leaves returns the leaves of the blocktree as an array
func (bs *BlockState) Leaves() []common.Hash {
	return bs.bt.Leaves()
}

// BlocktreeAsString returns the blocktree as a string
func (bs *BlockState) BlocktreeAsString() string {
	return bs.bt.String()
}

// GetArrivalTime returns the arrival time in nanoseconds since the Unix epoch of a block given its hash
func (bs *BlockState) GetArrivalTime(hash common.Hash) (time.Time, error) {
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

func (bs *BlockState) setArrivalTime(hash common.Hash, arrivalTime time.Time) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(arrivalTime.UnixNano()))
	return bs.db.Put(arrivalTimeKey(hash), buf)
}

// HandleRuntimeChanges handles the update in runtime.
func (bs *BlockState) HandleRuntimeChanges(newState *rtstorage.TrieState,
	rt runtime.Instance, bHash common.Hash) error {
	currCodeHash, err := newState.LoadCodeHash()
	if err != nil {
		return err
	}

	codeHash := rt.GetCodeHash()
	if bytes.Equal(codeHash[:], currCodeHash[:]) {
		bs.StoreRuntime(bHash, rt)
		return nil
	}

	logger.Infof("🔄 detected runtime code change, upgrading with block %s from previous code hash %s to new code hash %s...", //nolint:lll
		bHash, codeHash, currCodeHash)
	code := newState.LoadCode()
	if len(code) == 0 {
		return errors.New("new :code is empty")
	}

	codeSubBlockHash := bs.baseState.LoadCodeSubstitutedBlockHash()

	if !codeSubBlockHash.Equal(common.Hash{}) {
		newVersion, err := wasmer.GetRuntimeVersion(code)
		if err != nil {
			return err
		}

		// only update runtime during code substitution if runtime SpecVersion is updated
		previousVersion := rt.Version()
		if previousVersion.SpecVersion == newVersion.SpecVersion {
			logger.Info("not upgrading runtime code during code substitution")
			bs.StoreRuntime(bHash, rt)
			return nil
		}

		logger.Infof(
			"🔄 detected runtime code change, upgrading with block %s from previous code hash %s and spec %d to new code hash %s and spec %d...", //nolint:lll
			bHash, codeHash, previousVersion.SpecVersion, currCodeHash, newVersion.SpecVersion)
	}

	rtCfg := wasmer.Config{
		Storage:     newState,
		Keystore:    rt.Keystore(),
		NodeStorage: rt.NodeStorage(),
		Network:     rt.NetworkService(),
		CodeHash:    currCodeHash,
	}

	if rt.Validator() {
		rtCfg.Role = 4
	}

	instance, err := wasmer.NewInstance(code, rtCfg)
	if err != nil {
		return err
	}

	bs.ClearRuntimes()
	bs.StoreRuntime(bHash, instance)

	err = bs.baseState.StoreCodeSubstitutedBlockHash(common.Hash{})
	if err != nil {
		return fmt.Errorf("failed to update code substituted block hash: %w", err)
	}

	newVersion := rt.Version()
	go bs.notifyRuntimeUpdated(newVersion)
	return nil
}

// GetRuntime gets the runtime for the corresponding block hash.
func (bs *BlockState) GetRuntime(hash *common.Hash) (runtime.Instance, error) {
	if hash == nil {
		rt, err := bs.bt.GetBlockRuntime(bs.BestBlockHash())
		if err != nil {
			return nil, err
		}
		return rt, nil
	}

	return bs.bt.GetBlockRuntime(*hash)
}

func (bs *BlockState) GetRuntimeFromDB(blockHash *common.Hash) (instance runtime.Instance, err error) {
	var stateRootHash *common.Hash
	if blockHash != nil {
		stateRootHash, err = bs.storageState.GetStateRootFromBlock(blockHash)
		if err != nil {
			return nil, fmt.Errorf("getting state root from block hash: %w", err)
		}
	}

	trieState, err := bs.storageState.TrieState(stateRootHash)
	if err != nil {
		return nil, fmt.Errorf("getting trie state: %w", err)
	}

	var blockHashValue common.Hash
	if blockHash != nil {
		blockHashValue = *blockHash
	} else {
		blockHashValue = bs.BestBlockHash()
	}
	bs.storageState.LoadCode(&blockHashValue)
	instance, err = blockState.GetRuntime(blockHashValue)
	if err != nil {
		return nil, fmt.Errorf("getting runtime: %w", err)
	}

	instance.SetContextStorage(trieState)
	return instance, nil
}

// StoreRuntime stores the runtime for corresponding block hash.
func (bs *BlockState) StoreRuntime(hash common.Hash, rt runtime.Instance) {
	bs.bt.StoreRuntime(hash, rt)
}

// ClearRuntimes deletes references to runtime instances stored in block tree
func (bs *BlockState) ClearRuntimes() {
	bs.bt.ClearRuntimes()
}

// GetNonFinalisedBlocks get all the blocks in the blocktree
func (bs *BlockState) GetNonFinalisedBlocks() []common.Hash {
	return bs.bt.GetAllBlocks()
}
