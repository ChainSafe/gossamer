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
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/core/types"
	babetypes "github.com/ChainSafe/gossamer/lib/babe/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/database"
)

var blockPrefix = []byte("block")

// BlockDB stores block's in an underlying Database
type BlockDB struct {
	db database.Database
}

// Put appends `block` to the key and sets the key-value pair in the db
func (blockDB *BlockDB) Put(key, value []byte) error {
	key = append(blockPrefix, key...)
	return blockDB.db.Put(key, value)
}

// Get appends `block` to the key and retrieves the value from the db
func (blockDB *BlockDB) Get(key []byte) ([]byte, error) {
	key = append(blockPrefix, key...)
	return blockDB.db.Get(key)
}

// BlockState defines fields for manipulating the state of blocks, such as BlockTree, BlockDB and Header
type BlockState struct {
	bt                 *blocktree.BlockTree
	db                 *BlockDB
	lock               sync.RWMutex
	genesisHash        common.Hash
	highestBlockHeader *types.Header
}

// NewBlockDB instantiates a badgerDB instance for storing relevant BlockData
func NewBlockDB(db database.Database) *BlockDB {
	return &BlockDB{
		db,
	}
}

// NewBlockState will create a new BlockState backed by the database located at dataDir
func NewBlockState(db database.Database, bt *blocktree.BlockTree) (*BlockState, error) {
	if bt == nil {
		return nil, fmt.Errorf("block tree is nil")
	}

	bs := &BlockState{
		bt: bt,
		db: NewBlockDB(db),
	}

	bs.genesisHash = bt.GenesisHash()
	var err error
	bs.highestBlockHeader, err = bs.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	return bs, nil
}

// NewBlockStateFromGenesis initializes a BlockState from a genesis header, saving it to the database located at dataDir
func NewBlockStateFromGenesis(db database.Database, header *types.Header) (*BlockState, error) {
	bs := &BlockState{
		bt: blocktree.NewBlockTreeFromGenesis(header, db),
		db: NewBlockDB(db),
	}

	err := bs.setArrivalTime(header.Hash(), uint64(time.Now().Unix()))
	if err != nil {
		return nil, err
	}

	err = bs.SetHeader(header)
	if err != nil {
		return nil, err
	}

	err = bs.SetBlockBody(header.Hash(), types.NewBody([]byte{}))

	if err != nil {
		return nil, err
	}

	bs.genesisHash = header.Hash()

	return bs, nil
}

var (
	// Data prefixes
	headerPrefix        = []byte("hdr") // headerPrefix + hash -> header
	babeHeaderPrefix    = []byte("hba") // babeHeaderPrefix || epoch || slot -> babeHeader
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

// GetHeader returns a BlockHeader for a given hash
func (bs *BlockState) GetHeader(hash common.Hash) (*types.Header, error) {
	result := new(types.Header)

	data, err := bs.db.Get(headerKey(hash))
	if err != nil {
		return nil, err
	}

	err = result.Decode(data)
	if err != nil {
		return nil, err
	}

	if reflect.DeepEqual(result, new(types.Header)) {
		return nil, fmt.Errorf("header does not exist")
	}

	result.Hash()
	return result, err
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
func (bs *BlockState) GetBlockByNumber(blockNumber *big.Int) (*types.Block, error) {
	// First retrieve the block hash in a byte array based on the block number from the database
	byteHash, err := bs.db.Get(headerHashKey(blockNumber.Uint64()))
	if err != nil {
		return nil, fmt.Errorf("cannot get block %d: %s", blockNumber, err)
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

	// if this is the highest block we've seen, save it
	if bs.highestBlockHeader == nil {
		bs.highestBlockHeader = header
	} else if bs.highestBlockHeader.Number.Cmp(header.Number) == -1 {
		bs.highestBlockHeader = header
	}

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

	err := bs.db.Put(blockBodyKey(hash), body.AsOptional().Value)
	return err
}

// CompareAndSetBlockData will compare empty fields and set all elements in a block data to db
func (bs *BlockState) CompareAndSetBlockData(bd *types.BlockData) error {
	var existingData = new(types.BlockData)

	if bd.Header != nil && (existingData.Header == nil || (!existingData.Header.Exists() && bd.Header.Exists())) {
		existingData.Header = bd.Header
		header, err := types.NewHeaderFromOptional(existingData.Header)
		if err != nil && header != nil {
			err = bs.SetHeader(header)
			if err != nil {
				return err
			}
		}
	}

	if bd.Body != nil && (existingData.Body == nil || (!existingData.Body.Exists && bd.Body.Exists)) {
		existingData.Body = bd.Body
		err := bs.SetBlockBody(bd.Hash, types.NewBody(existingData.Body.Value))
		if err != nil {
			return err
		}
	}

	if bd.Receipt != nil && (existingData.Receipt == nil || (!existingData.Receipt.Exists() && bd.Receipt.Exists())) {
		existingData.Receipt = bd.Receipt
		err := bs.SetReceipt(bd.Hash, existingData.Receipt.Value())
		if err != nil {
			return err
		}
	}

	if bd.MessageQueue != nil && (existingData.MessageQueue == nil || (!existingData.MessageQueue.Exists() && bd.MessageQueue.Exists())) {
		existingData.MessageQueue = bd.MessageQueue
		err := bs.SetMessageQueue(bd.Hash, existingData.MessageQueue.Value())
		if err != nil {
			return err
		}
	}

	if bd.Justification != nil && (existingData.Justification == nil || (!existingData.Justification.Exists() && bd.Justification.Exists())) {
		existingData.Justification = bd.Justification
		err := bs.SetJustification(bd.Hash, existingData.Justification.Value())
		if err != nil {
			return err
		}
	}

	return nil
}

// AddBlock adds a block to the blocktree and the DB with arrival time as current unix time
func (bs *BlockState) AddBlock(block *types.Block) error {
	return bs.AddBlockWithArrivalTime(block, uint64(time.Now().Unix()))
}

// AddBlockWithArrivalTime adds a block to the blocktree and the DB with the given arrival time
func (bs *BlockState) AddBlockWithArrivalTime(block *types.Block, arrivalTime uint64) error {
	err := bs.setArrivalTime(block.Header.Hash(), arrivalTime)
	if err != nil {
		return err
	}

	// add block to blocktree
	err = bs.bt.AddBlock(block, arrivalTime)
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

	err = bs.SetBlockBody(block.Header.Hash(), types.NewBody(block.Body.AsOptional().Value))
	if err != nil {
		return err
	}
	return err
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

	_, err = bs.SubChain(header.Hash(), bestBlock.Hash())
	if err != nil {
		// subchain function will error if the new block is not a precessor of our best block,
		// thus it is not on our current chain
		return false, nil
	}

	return true, nil
}

// HighestBlockHash returns the hash of the block with the highest number we have received
// This block may not necessarily be in the blocktree.
// TODO: can probably remove this once BlockResponses are implemented
func (bs *BlockState) HighestBlockHash() common.Hash {
	return bs.highestBlockHeader.Hash()
}

// HighestBlockNumber returns the largest block number we have seen
// This block may not necessarily be in the blocktree.
// TODO: can probably remove this once BlockResponses are implemented
func (bs *BlockState) HighestBlockNumber() *big.Int {
	return bs.highestBlockHeader.Number
}

// BestBlockHash returns the hash of the head of the current chain
func (bs *BlockState) BestBlockHash() common.Hash {
	return bs.bt.DeepestBlockHash()
}

// BestBlockHeader returns the block header of the current head of the chain
func (bs *BlockState) BestBlockHeader() (*types.Header, error) {
	return bs.GetHeader(bs.BestBlockHash())
}

// BestBlockNumber returns the block number of the current head of the chain
func (bs *BlockState) BestBlockNumber() (*big.Int, error) {
	header, err := bs.GetHeader(bs.BestBlockHash())
	if err != nil {
		return nil, err
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

	if len(header.Digest) == 0 {
		return 0, fmt.Errorf("chain head missing digest")
	}

	preDigestBytes := header.Digest[0]

	digestItem, err := types.DecodeDigestItem(preDigestBytes)
	if err != nil {
		return 0, err
	}

	preDigest, ok := digestItem.(*types.PreRuntimeDigest)
	if !ok {
		return 0, fmt.Errorf("first digest item is not pre-digest")
	}

	babeHeader := new(babetypes.BabeHeader)
	err = babeHeader.Decode(preDigest.Data)
	if err != nil {
		return 0, fmt.Errorf("cannot decode babe header from pre-digest: %s", err)
	}

	return babeHeader.SlotNumber, nil
}

// SubChain returns the sub-blockchain between the starting hash and the ending hash using the block tree
func (bs *BlockState) SubChain(start, end common.Hash) ([]common.Hash, error) {
	if bs.bt == nil {
		return nil, fmt.Errorf("blocktree is nil")
	}

	return bs.bt.SubBlockchain(start, end)
}

func (bs *BlockState) setBestBlockHashKey(hash common.Hash) error {
	return StoreBestBlockHash(bs.db.db, hash)
}

// GetArrivalTime returns the arrival time of a block given its hash
func (bs *BlockState) GetArrivalTime(hash common.Hash) (uint64, error) {
	arrivalTime, err := bs.db.db.Get(arrivalTimeKey(hash))
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(arrivalTime), nil
}

func (bs *BlockState) setArrivalTime(hash common.Hash, arrivalTime uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, arrivalTime)
	return bs.db.db.Put(arrivalTimeKey(hash), buf)
}

// babeHeaderKey = babeHeaderPrefix || epoch || slice
func babeHeaderKey(epoch uint64, slot uint64) []byte {
	epochBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBytes, epoch)
	sliceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sliceBytes, slot)
	combined := append(epochBytes, sliceBytes...)
	return append(babeHeaderPrefix, combined...)
}

// GetBabeHeader retrieves a BabeHeader from the database
func (bs *BlockState) GetBabeHeader(epoch uint64, slot uint64) (*babetypes.BabeHeader, error) {
	result := new(babetypes.BabeHeader)

	data, err := bs.db.Get(babeHeaderKey(epoch, slot))
	if err != nil {
		return nil, err
	}

	err = result.Decode(data)

	return result, err
}

// SetBabeHeader sets a BabeHeader in the database
func (bs *BlockState) SetBabeHeader(epoch uint64, slot uint64, bh *babetypes.BabeHeader) error {
	// Write the encoded header
	enc := bh.Encode()

	return bs.db.Put(babeHeaderKey(epoch, slot), enc)
}
