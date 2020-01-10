package state

import (
	"encoding/json"
	"math/big"
	"reflect"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/blocktree"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
)

type blockState struct {
	bt          *blocktree.BlockTree
	db          *polkadb.BlockDB
	latestBlock types.BlockHeader
}

func NewBlockState(dataDir string) (*blockState, error) {
	blockDb, err := polkadb.NewBlockDB(dataDir)
	if err != nil {
		return nil, err
	}
	return &blockState{
		bt: &blocktree.BlockTree{},
		db: blockDb,
	}, nil
}

var (
	// Data prefixes
	headerPrefix    = []byte("hdr") // headerPrefix + hash -> header
	blockDataPrefix = []byte("hsh") // blockDataPrefix + hash -> blockData
	blockPrefix     = []byte("blk") // blockPrefix + hash -> block
)

// headerKey = headerPrefix + hash
func headerKey(hash common.Hash) []byte {
	return append(headerPrefix, hash.ToBytes()...)
}

// blockDataKey = blockDataPrefix + hash
func blockDataKey(hash common.Hash) []byte {
	return append(blockDataPrefix, hash.ToBytes()...)
}

// blockKey = blockDataPrefix + hash
func blockKey(hash common.Hash) []byte {
	return append(blockPrefix, hash.ToBytes()...)
}

func (bs *blockState) GetHeader(hash common.Hash) (*types.BlockHeader, error) {
	result := new(types.BlockHeader)

	data, err := bs.db.Db.Get(headerKey(hash))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, result)
	return result, err
}

func (bs *blockState) GetBlockData(hash common.Hash) (types.BlockData, error) {
	var result types.BlockData

	data, err := bs.db.Db.Get(blockDataKey(hash))
	if err != nil {
		return types.BlockData{}, err
	}

	err = json.Unmarshal(data, &result)

	return result, err
}

func (bs *blockState) GetBlockByHash(hash common.Hash) (types.Block, error) {
	header, err := bs.GetHeader(hash)
	if err != nil {
		return types.Block{}, err
	}
	blockData, err := bs.GetBlockData(hash)
	if err != nil {
		return types.Block{}, err
	}
	blockBody := blockData.Body
	return types.Block{Header: header, Body: blockBody}, nil
}

func (bs *blockState) GetBlockByNumber(n *big.Int) types.Block {
	// Can't do yet
	return types.Block{}
}

func (bs *blockState) SetHeader(header types.BlockHeader) error {
	hash, err := header.Hash()
	if err != nil {
		return err
	}

	// Write the encoded header
	bh, err := json.Marshal(header)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(headerKey(hash), bh)
	return err
}

func (bs *blockState) SetBlockData(hash common.Hash, blockData types.BlockData) error {
	// Write the encoded header
	bh, err := json.Marshal(blockData)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(blockDataKey(hash), bh)
	return err
}

// Set the latestBlock in blockState
// Set the Header & BlockData in the DB
func (bs *blockState) AddBlock(block types.Block) error {
	blockHeader := *block.Header

	// Set the latest block
	if reflect.DeepEqual(bs.latestBlock, types.BlockHeader{}) {
		bs.latestBlock = blockHeader
	} else if blockHeader.Number != nil && blockHeader.Number.Cmp(bs.latestBlock.Number) == 1 { // If the new block has a number greater than current latestBlock
		bs.latestBlock = blockHeader
	}

	//Add the blockHeader to the DB
	bs.SetHeader(blockHeader)
	hash, err := blockHeader.Hash()
	if err != nil {
		return err
	}

	blockHeaderWithHash, err := block.Header.WithHash()
	if err != nil {
		return err
	}

	// Create BlockData
	bd := types.BlockData{
		Hash:   hash,
		Header: blockHeaderWithHash,
		Body:   block.Body,
	}
	err = bs.SetBlockData(hash, bd)
	return err
}
