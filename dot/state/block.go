// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"

	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

var blockPrefix = "block"

const pruneKeyBufferSize = 1000

// BlockState defines fields for manipulating the state of blocks, such as BlockTree,
// BlockDB and Header
type BlockState struct {
	bt        *blocktree.BlockTree
	baseState *BaseState
	dbPath    string
	db        chaindb.Database
	sync.RWMutex
	genesisHash   common.Hash
	lastFinalised common.Hash

	// block notifiers
	imported                       map[chan *types.Block]struct{}
	finalised                      map[chan *types.FinalisationInfo]struct{}
	finalisedLock                  sync.RWMutex
	importedLock                   sync.RWMutex
	runtimeUpdateSubscriptionsLock sync.RWMutex
	runtimeUpdateSubscriptions     map[uint32]chan<- runtime.Version

	pruneKeyCh chan *types.Header
}

// NewBlockState will create a new BlockState backed by the database located at basePath
func NewBlockState(db chaindb.Database, bt *blocktree.BlockTree) (*BlockState, error) {
	if bt == nil {
		return nil, fmt.Errorf("block tree is nil")
	}

	bs := &BlockState{
		bt:                         bt,
		dbPath:                     db.Path(),
		baseState:                  NewBaseState(db),
		db:                         chaindb.NewTable(db, blockPrefix),
		imported:                   make(map[chan *types.Block]struct{}),
		finalised:                  make(map[chan *types.FinalisationInfo]struct{}),
		pruneKeyCh:                 make(chan *types.Header, pruneKeyBufferSize),
		runtimeUpdateSubscriptions: make(map[uint32]chan<- runtime.Version),
	}

	genesisBlock, err := bs.GetBlockByNumber(big.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("failed to get genesis header: %w", err)
	}

	bs.genesisHash = genesisBlock.Header.Hash()
	bs.lastFinalised, err = bs.GetHighestFinalisedHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get last finalised hash: %w", err)
	}

	return bs, nil
}

// NewBlockStateFromGenesis initialises a BlockState from a genesis header, saving it to the database located at basePath
func NewBlockStateFromGenesis(db chaindb.Database, header *types.Header) (*BlockState, error) {
	bs := &BlockState{
		bt:                         blocktree.NewBlockTreeFromRoot(header, db),
		baseState:                  NewBaseState(db),
		db:                         chaindb.NewTable(db, blockPrefix),
		imported:                   make(map[chan *types.Block]struct{}),
		finalised:                  make(map[chan *types.FinalisationInfo]struct{}),
		pruneKeyCh:                 make(chan *types.Header, pruneKeyBufferSize),
		runtimeUpdateSubscriptions: make(map[uint32]chan<- runtime.Version),
		genesisHash:                header.Hash(),
		lastFinalised:              header.Hash(),
	}

	if err := bs.setArrivalTime(header.Hash(), time.Now()); err != nil {
		return nil, err
	}

	if err := bs.SetHeader(header); err != nil {
		return nil, err
	}

	if err := bs.db.Put(headerHashKey(header.Number.Uint64()), header.Hash().ToBytes()); err != nil {
		return nil, err
	}

	if err := bs.SetBlockBody(header.Hash(), types.NewBody([]types.Extrinsic{})); err != nil {
		return nil, err
	}

	if err := bs.db.Put(highestRoundAndSetIDKey, roundAndSetIDToBytes(0, 0)); err != nil {
		return nil, err
	}

	// set the latest finalised head to the genesis header
	if err := bs.SetFinalisedHash(bs.genesisHash, 0, 0); err != nil {
		return nil, err
	}

	return bs, nil
}

var (
	// Data prefixes
	headerPrefix        = []byte("hdr") // headerPrefix + hash -> header
	blockBodyPrefix     = []byte("blb") // blockBodyPrefix + hash -> body
	headerHashPrefix    = []byte("hsh") // headerHashPrefix + encodedBlockNum -> hash
	arrivalTimePrefix   = []byte("arr") // arrivalTimePrefix || hash -> arrivalTime
	receiptPrefix       = []byte("rcp") // receiptPrefix + hash -> receipt
	messageQueuePrefix  = []byte("mqp") // messageQueuePrefix + hash -> message queue
	justificationPrefix = []byte("jcp") // justificationPrefix + hash -> justification
)

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

// DeleteBlock deletes all instances of the block and its related data in the database
func (bs *BlockState) DeleteBlock(hash common.Hash) error {
	if has, _ := bs.HasHeader(hash); has {
		err := bs.db.Del(headerKey(hash))
		if err != nil {
			return err
		}
	}

	if has, _ := bs.HasBlockBody(hash); has {
		err := bs.db.Del(blockBodyKey(hash))
		if err != nil {
			return err
		}
	}

	if has, _ := bs.HasArrivalTime(hash); has {
		err := bs.db.Del(arrivalTimeKey(hash))
		if err != nil {
			return err
		}
	}

	if has, _ := bs.HasReceipt(hash); has {
		err := bs.db.Del(prefixKey(hash, receiptPrefix))
		if err != nil {
			return err
		}
	}

	if has, _ := bs.HasMessageQueue(hash); has {
		err := bs.db.Del(prefixKey(hash, messageQueuePrefix))
		if err != nil {
			return err
		}
	}

	if has, _ := bs.HasJustification(hash); has {
		err := bs.db.Del(prefixKey(hash, justificationPrefix))
		if err != nil {
			return err
		}
	}

	return nil
}

// HasHeader returns if the db contains a header with the given hash
func (bs *BlockState) HasHeader(hash common.Hash) (bool, error) {
	return bs.db.Has(headerKey(hash))
}

// GetHeader returns a BlockHeader for a given hash
func (bs *BlockState) GetHeader(hash common.Hash) (*types.Header, error) {
	result := types.NewEmptyHeader()

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

// GetHashByNumber returns the block hash given the number
func (bs *BlockState) GetHashByNumber(num *big.Int) (common.Hash, error) {
	bh, err := bs.db.Get(headerHashKey(num.Uint64()))
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot get block %d: %w", num, err)
	}

	return common.NewHash(bh), nil
}

// GetHeaderByNumber returns a block header given a number
func (bs *BlockState) GetHeaderByNumber(num *big.Int) (*types.Header, error) {
	bh, err := bs.db.Get(headerHashKey(num.Uint64()))
	if err != nil {
		return nil, fmt.Errorf("cannot get block %d: %w", num, err)
	}

	hash := common.NewHash(bh)
	return bs.GetHeader(hash)
}

// GetBlockByHash returns a block for a given hash
func (bs *BlockState) GetBlockByHash(hash common.Hash) (*types.Block, error) {
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

// GetBlockByNumber returns a block for a given blockNumber
func (bs *BlockState) GetBlockByNumber(num *big.Int) (*types.Block, error) {
	// First retrieve the block hash in a byte array based on the block number from the database
	byteHash, err := bs.db.Get(headerHashKey(num.Uint64()))
	if err != nil {
		return nil, fmt.Errorf("cannot get block %d: %w", num, err)
	}

	// Then find the block based on the hash
	hash := common.NewHash(byteHash)
	block, err := bs.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GetBlockHash returns block hash for a given blockNumber
func (bs *BlockState) GetBlockHash(blockNumber *big.Int) (common.Hash, error) {
	byteHash, err := bs.db.Get(headerHashKey(blockNumber.Uint64()))
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot get block %d: %w", blockNumber, err)
	}

	return common.NewHash(byteHash), nil
}

// SetHeader will set the header into DB
func (bs *BlockState) SetHeader(header *types.Header) error {
	hash := header.Hash()
	// Write the encoded header
	bh, err := scale.Marshal(*header)
	if err != nil {
		return err
	}

	err = bs.db.Put(headerKey(hash), bh)
	if err != nil {
		return err
	}

	return nil
}

// HasBlockBody returns true if the db contains the block body
func (bs *BlockState) HasBlockBody(hash common.Hash) (bool, error) {
	return bs.db.Has(blockBodyKey(hash))
}

// GetBlockBody will return Body for a given hash
func (bs *BlockState) GetBlockBody(hash common.Hash) (*types.Body, error) {
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
	err := bs.setArrivalTime(block.Header.Hash(), arrivalTime)
	if err != nil {
		return err
	}

	prevHead := bs.bt.DeepestBlockHash()

	// add block to blocktree
	err = bs.bt.AddBlock(&block.Header, uint64(arrivalTime.UnixNano()))
	if err != nil {
		return err
	}

	// add the header to the DB
	err = bs.SetHeader(&block.Header)
	if err != nil {
		return err
	}
	hash := block.Header.Hash()

	// set best block key if this is the highest block we've seen
	if hash == bs.BestBlockHash() {
		err = bs.setBestBlockHashKey(hash)
		if err != nil {
			return err
		}
	}

	// only set number->hash mapping for our current chain
	var onChain bool
	if onChain, err = bs.isBlockOnCurrentChain(&block.Header); onChain && err == nil {
		err = bs.db.Put(headerHashKey(block.Header.Number.Uint64()), hash.ToBytes())
		if err != nil {
			return err
		}
	}

	err = bs.SetBlockBody(block.Header.Hash(), &block.Body)
	if err != nil {
		return err
	}

	// check if there was a re-org, if so, re-set the canonical number->hash mapping
	err = bs.handleAddedBlock(prevHead, bs.bt.DeepestBlockHash())
	if err != nil {
		return err
	}

	go bs.notifyImported(block)
	return bs.db.Flush()
}

// handleAddedBlock re-sets the canonical number->hash mapping if there was a chain re-org.
// prev is the previous best block hash before the new block was added to the blocktree.
// curr is the current best block hash.
func (bs *BlockState) handleAddedBlock(prev, curr common.Hash) error {
	ancestor, err := bs.HighestCommonAncestor(prev, curr)
	if err != nil {
		return err
	}

	// if the highest common ancestor of the previous chain head and current chain head is the previous chain head,
	// then the current chain head is the descendant of the previous and thus are on the same chain
	if ancestor == prev {
		return nil
	}

	subchain, err := bs.SubChain(ancestor, curr)
	if err != nil {
		return err
	}

	batch := bs.db.NewBatch()
	for _, hash := range subchain {
		// TODO: set number from ancestor.Number + i ?
		header, err := bs.GetHeader(hash)
		if err != nil {
			return fmt.Errorf("failed to get header in subchain: %w", err)
		}

		err = batch.Put(headerHashKey(header.Number.Uint64()), hash.ToBytes())
		if err != nil {
			return err
		}
	}

	return batch.Flush()
}

// AddBlockToBlockTree adds the given block to the blocktree. It does not write it to the database.
func (bs *BlockState) AddBlockToBlockTree(header *types.Header) error {
	bs.Lock()
	defer bs.Unlock()

	arrivalTime, err := bs.GetArrivalTime(header.Hash())
	if err != nil {
		arrivalTime = time.Now()
	}

	return bs.bt.AddBlock(header, uint64(arrivalTime.UnixNano()))
}

// GetAllBlocksAtDepth returns all hashes with the depth of the given hash plus one
func (bs *BlockState) GetAllBlocksAtDepth(hash common.Hash) []common.Hash {
	return bs.bt.GetAllBlocksAtDepth(hash)
}

func (bs *BlockState) isBlockOnCurrentChain(header *types.Header) (bool, error) {
	bestBlock, err := bs.BestBlockHeader()
	if err != nil {
		return false, err
	}

	// if the new block is ahead of our best block, then it is on our current chain.
	if header.Number.Cmp(bestBlock.Number) > 0 {
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

	return bs.bt.DeepestBlockHash()
}

// BestBlockHeader returns the block header of the current head of the chain
func (bs *BlockState) BestBlockHeader() (*types.Header, error) {
	return bs.GetHeader(bs.BestBlockHash())
}

// BestBlockStateRoot returns the state root of the current head of the chain
func (bs *BlockState) BestBlockStateRoot() (common.Hash, error) {
	header, err := bs.GetHeader(bs.BestBlockHash())
	if err != nil {
		return common.Hash{}, err
	}

	return header.StateRoot, nil
}

// GetBlockStateRoot returns the state root of the given block hash
func (bs *BlockState) GetBlockStateRoot(bhash common.Hash) (common.Hash, error) {
	header, err := bs.GetHeader(bhash)
	if err != nil {
		return common.EmptyHash, err
	}

	return header.StateRoot, nil
}

// BestBlockNumber returns the block number of the current head of the chain
func (bs *BlockState) BestBlockNumber() (*big.Int, error) {
	header, err := bs.GetHeader(bs.BestBlockHash())
	if err != nil {
		return nil, err
	}

	if header == nil {
		return nil, fmt.Errorf("failed to get best block header")
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

func (bs *BlockState) setBestBlockHashKey(hash common.Hash) error {
	return bs.baseState.StoreBestBlockHash(hash)
}

// HasArrivalTime returns true if the db contains the block's arrival time
func (bs *BlockState) HasArrivalTime(hash common.Hash) (bool, error) {
	return bs.db.Has(arrivalTimeKey(hash))
}

// GetArrivalTime returns the arrival time in nanoseconds since the Unix epoch of a block given its hash
func (bs *BlockState) GetArrivalTime(hash common.Hash) (time.Time, error) {
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
func (bs *BlockState) HandleRuntimeChanges(newState *rtstorage.TrieState, rt runtime.Instance, bHash common.Hash) error {
	currCodeHash, err := newState.LoadCodeHash()
	if err != nil {
		return err
	}

	codeHash := rt.GetCodeHash()
	if bytes.Equal(codeHash[:], currCodeHash[:]) {
		bs.StoreRuntime(bHash, rt)
		return nil
	}

	logger.Info("🔄 detected runtime code change, upgrading...", "block", bHash, "previous code hash", codeHash, "new code hash", currCodeHash)
	code := newState.LoadCode()
	if len(code) == 0 {
		return errors.New("new :code is empty")
	}

	codeSubBlockHash := bs.baseState.LoadCodeSubstitutedBlockHash()

	if !codeSubBlockHash.Equal(common.Hash{}) {
		newVersion, err := rt.CheckRuntimeVersion(code) //nolint
		if err != nil {
			return err
		}

		// only update runtime during code substitution if runtime SpecVersion is updated
		previousVersion, _ := rt.Version()
		if previousVersion.SpecVersion() == newVersion.SpecVersion() {
			logger.Info("not upgrading runtime code during code substitution")
			bs.StoreRuntime(bHash, rt)
			return nil
		}

		logger.Info("🔄 detected runtime code change, upgrading...", "block", bHash,
			"previous code hash", codeHash, "new code hash", currCodeHash,
			"previous spec version", previousVersion.SpecVersion(), "new spec version", newVersion.SpecVersion())
	}

	rtCfg := &wasmer.Config{
		Imports: wasmer.ImportsNodeRuntime,
	}

	rtCfg.Storage = newState
	rtCfg.Keystore = rt.Keystore()
	rtCfg.NodeStorage = rt.NodeStorage()
	rtCfg.Network = rt.NetworkService()
	rtCfg.CodeHash = currCodeHash

	if rt.Validator() {
		rtCfg.Role = 4
	}

	instance, err := wasmer.NewInstance(code, rtCfg)
	if err != nil {
		return err
	}

	bs.StoreRuntime(bHash, instance)

	err = bs.baseState.StoreCodeSubstitutedBlockHash(common.Hash{})
	if err != nil {
		return fmt.Errorf("failed to update code substituted block hash: %w", err)
	}

	newVersion, err := rt.Version()
	if err != nil {
		return fmt.Errorf("failed to retrieve runtime version: %w", err)
	}
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

// StoreRuntime stores the runtime for corresponding block hash.
func (bs *BlockState) StoreRuntime(hash common.Hash, rt runtime.Instance) {
	bs.bt.StoreRuntime(hash, rt)
}

// GetNonFinalisedBlocks get all the blocks in the blocktree
func (bs *BlockState) GetNonFinalisedBlocks() []common.Hash {
	return bs.bt.GetAllBlocks()
}
