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
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/ChainSafe/chaindb"
)

var blockPrefix = "block"

const pruneKeyBufferSize = 1000

// BlockState defines fields for manipulating the state of blocks, such as BlockTree, BlockDB and Header
type BlockState struct {
	bt          *blocktree.BlockTree
	baseDB      chaindb.Database
	db          chaindb.Database
	lock        sync.RWMutex
	genesisHash common.Hash

	// block notifiers
	imported      map[byte]chan<- *types.Block
	finalized     map[byte]chan<- *types.Header
	importedLock  sync.RWMutex
	finalizedLock sync.RWMutex

	pruneKeyCh chan *types.Header
}

// NewBlockState will create a new BlockState backed by the database located at basePath
func NewBlockState(db chaindb.Database, bt *blocktree.BlockTree) (*BlockState, error) {
	if bt == nil {
		return nil, fmt.Errorf("block tree is nil")
	}

	bs := &BlockState{
		bt:         bt,
		baseDB:     db,
		db:         chaindb.NewTable(db, blockPrefix),
		imported:   make(map[byte]chan<- *types.Block),
		finalized:  make(map[byte]chan<- *types.Header),
		pruneKeyCh: make(chan *types.Header, pruneKeyBufferSize),
	}

	genesisBlock, err := bs.GetBlockByNumber(big.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("failed to get genesis header: %w", err)
	}

	bs.genesisHash = genesisBlock.Header.Hash()
	return bs, nil
}

// NewBlockStateFromGenesis initializes a BlockState from a genesis header, saving it to the database located at basePath
func NewBlockStateFromGenesis(db chaindb.Database, header *types.Header) (*BlockState, error) {
	bs := &BlockState{
		bt:         blocktree.NewBlockTreeFromRoot(header, db),
		baseDB:     db,
		db:         chaindb.NewTable(db, blockPrefix),
		imported:   make(map[byte]chan<- *types.Block),
		finalized:  make(map[byte]chan<- *types.Header),
		pruneKeyCh: make(chan *types.Header, pruneKeyBufferSize),
	}

	err := bs.setArrivalTime(header.Hash(), time.Now())
	if err != nil {
		return nil, err
	}

	err = bs.SetHeader(header)
	if err != nil {
		return nil, err
	}

	err = bs.db.Put(headerHashKey(header.Number.Uint64()), header.Hash().ToBytes())
	if err != nil {
		return nil, err
	}

	err = bs.SetBlockBody(header.Hash(), types.NewBody([]byte{}))
	if err != nil {
		return nil, err
	}

	bs.genesisHash = header.Hash()

	// set the latest finalized head to the genesis header
	err = bs.SetFinalizedHash(bs.genesisHash, 0, 0)
	if err != nil {
		return nil, err
	}

	err = bs.SetRound(0)
	if err != nil {
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

// finalizedHashKey = hashkey + round + setID (LE encoded)
func finalizedHashKey(round, setID uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, round)
	key := append(common.FinalizedBlockHashKey, buf...)
	binary.LittleEndian.PutUint64(buf, setID)
	return append(key, buf...)
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
	result := new(types.Header)

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

	rw := &bytes.Buffer{}
	_, err = rw.Write(data)
	if err != nil {
		return nil, err
	}

	_, err = result.Decode(rw)
	if err != nil {
		return nil, err
	}

	if reflect.DeepEqual(result, new(types.Header)) {
		return nil, chaindb.ErrKeyNotFound
	}

	result.Hash()
	return result, err
}

// GetHashByNumber returns the block hash given the number
func (bs *BlockState) GetHashByNumber(num *big.Int) (common.Hash, error) {
	bh, err := bs.db.Get(headerHashKey(num.Uint64()))
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot get block %d: %s", num, err)
	}

	return common.NewHash(bh), nil
}

// GetHeaderByNumber returns a block header given a number
func (bs *BlockState) GetHeaderByNumber(num *big.Int) (*types.Header, error) {
	bh, err := bs.db.Get(headerHashKey(num.Uint64()))
	if err != nil {
		return nil, fmt.Errorf("cannot get block %d: %s", num, err)
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
	return &types.Block{Header: header, Body: blockBody}, nil
}

// GetBlockByNumber returns a block for a given blockNumber
func (bs *BlockState) GetBlockByNumber(num *big.Int) (*types.Block, error) {
	// First retrieve the block hash in a byte array based on the block number from the database
	byteHash, err := bs.db.Get(headerHashKey(num.Uint64()))
	if err != nil {
		return nil, fmt.Errorf("cannot get block %d: %s", num, err)
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
func (bs *BlockState) GetBlockHash(blockNumber *big.Int) (*common.Hash, error) {
	// First retrieve the block hash in a byte array based on the block number from the database
	byteHash, err := bs.db.Get(headerHashKey(blockNumber.Uint64()))
	if err != nil {
		return nil, fmt.Errorf("cannot get block %d: %s", blockNumber, err)
	}
	hash := common.NewHash(byteHash)
	return &hash, nil
}

// SetHeader will set the header into DB
func (bs *BlockState) SetHeader(header *types.Header) error {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	hash := header.Hash()

	// Write the encoded header
	bh, err := header.Encode()
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

	return types.NewBody(data), nil
}

// SetBlockBody will add a block body to the db
func (bs *BlockState) SetBlockBody(hash common.Hash, body *types.Body) error {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	err := bs.db.Put(blockBodyKey(hash), body.AsOptional().Value())
	return err
}

// HasFinalizedBlock returns true if there is a finalized block for a given round and setID, false otherwise
func (bs *BlockState) HasFinalizedBlock(round, setID uint64) (bool, error) {
	// get current round
	r, err := bs.GetRound()
	if err != nil {
		return false, err
	}

	// round that is being queried for has not yet finalized
	if round > r {
		return false, fmt.Errorf("round not yet finalized")
	}

	return bs.db.Has(finalizedHashKey(round, setID))
}

// GetFinalizedHeader returns the latest finalized block header
func (bs *BlockState) GetFinalizedHeader(round, setID uint64) (*types.Header, error) {
	h, err := bs.GetFinalizedHash(round, setID)
	if err != nil {
		return nil, err
	}

	header, err := bs.GetHeader(h)
	if err != nil {
		return nil, err
	}

	return header, nil
}

// GetFinalizedHash gets the latest finalized block header
func (bs *BlockState) GetFinalizedHash(round, setID uint64) (common.Hash, error) {
	// get current round
	r, err := bs.GetRound()
	if err != nil {
		return common.Hash{}, err
	}

	// round that is being queried for has not yet finalized
	if round > r {
		return common.Hash{}, fmt.Errorf("round not yet finalized")
	}

	h, err := bs.db.Get(finalizedHashKey(round, setID))
	if err != nil {
		return common.Hash{}, err
	}

	return common.NewHash(h), nil
}

// SetFinalizedHash sets the latest finalized block header
func (bs *BlockState) SetFinalizedHash(hash common.Hash, round, setID uint64) error {
	go bs.notifyFinalized(hash)
	if round > 0 {
		err := bs.SetRound(round)
		if err != nil {
			return err
		}
	}

	pruned := bs.bt.Prune(hash)
	for _, rem := range pruned {
		header, err := bs.GetHeader(rem)
		if err != nil {
			return err
		}

		err = bs.DeleteBlock(rem)
		if err != nil {
			return err
		}

		logger.Trace("pruned block", "hash", rem)
		bs.pruneKeyCh <- header
	}

	return bs.db.Put(finalizedHashKey(round, setID), hash[:])
}

// SetRound sets the latest finalized GRANDPA round in the db
// TODO: this needs to use both setID and round
func (bs *BlockState) SetRound(round uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, round)
	return bs.db.Put(common.LatestFinalizedRoundKey, buf)
}

// GetRound gets the latest finalized GRANDPA round from the db
func (bs *BlockState) GetRound() (uint64, error) {
	r, err := bs.db.Get(common.LatestFinalizedRoundKey)
	if err != nil {
		return 0, err
	}

	round := binary.LittleEndian.Uint64(r)
	return round, nil
}

// CompareAndSetBlockData will compare empty fields and set all elements in a block data to db
func (bs *BlockState) CompareAndSetBlockData(bd *types.BlockData) error {
	hasReceipt, _ := bs.HasReceipt(bd.Hash)
	if bd.Receipt != nil && bd.Receipt.Exists() && !hasReceipt {
		err := bs.SetReceipt(bd.Hash, bd.Receipt.Value())
		if err != nil {
			return err
		}
	}

	hasMessageQueue, _ := bs.HasMessageQueue(bd.Hash)
	if bd.MessageQueue != nil && bd.MessageQueue.Exists() && !hasMessageQueue {
		err := bs.SetMessageQueue(bd.Hash, bd.MessageQueue.Value())
		if err != nil {
			return err
		}
	}

	return nil
}

// AddBlock adds a block to the blocktree and the DB with arrival time as current unix time
func (bs *BlockState) AddBlock(block *types.Block) error {
	return bs.AddBlockWithArrivalTime(block, time.Now())
}

// AddBlockWithArrivalTime adds a block to the blocktree and the DB with the given arrival time
func (bs *BlockState) AddBlockWithArrivalTime(block *types.Block, arrivalTime time.Time) error {
	err := bs.setArrivalTime(block.Header.Hash(), arrivalTime)
	if err != nil {
		return err
	}

	// add block to blocktree
	err = bs.bt.AddBlock(block.Header, uint64(arrivalTime.UnixNano()))
	if err != nil {
		return err
	}

	// add the header to the DB
	err = bs.SetHeader(block.Header)
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
	if onChain, err = bs.isBlockOnCurrentChain(block.Header); onChain && err == nil {
		err = bs.db.Put(headerHashKey(block.Header.Number.Uint64()), hash.ToBytes())
		if err != nil {
			return err
		}
	}

	err = bs.SetBlockBody(block.Header.Hash(), types.NewBody(block.Body.AsOptional().Value()))
	if err != nil {
		return err
	}

	go bs.notifyImported(block)
	return bs.baseDB.Flush()
}

// AddBlockToBlockTree adds the given block to the blocktree. It does not write it to the database.
func (bs *BlockState) AddBlockToBlockTree(header *types.Header) error {
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
	if header.Number.Cmp(bestBlock.Number) == 1 {
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
	return StoreBestBlockHash(bs.baseDB, hash)
}

// HasArrivalTime returns true if the db contains the block's arrival time
func (bs *BlockState) HasArrivalTime(hash common.Hash) (bool, error) {
	return bs.db.Has(arrivalTimeKey(hash))
}

// GetArrivalTime returns the arrival time in nanoseconds since the Unix epoch of a block given its hash
func (bs *BlockState) GetArrivalTime(hash common.Hash) (time.Time, error) {
	arrivalTime, err := bs.baseDB.Get(arrivalTimeKey(hash))
	if err != nil {
		return time.Time{}, err
	}

	ns := binary.LittleEndian.Uint64(arrivalTime)
	return time.Unix(0, int64(ns)), nil
}

func (bs *BlockState) setArrivalTime(hash common.Hash, arrivalTime time.Time) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(arrivalTime.UnixNano()))
	return bs.baseDB.Put(arrivalTimeKey(hash), buf)
}
