package state

import (
	"encoding/json"
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/blocktree"
	"github.com/ChainSafe/gossamer/core/rawdb"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
)

type blockState struct {
	bt *blocktree.BlockTree
	db polkadb.BlockDB
}

func newBlockState() *blockState {
	return &blockState{
		bt: &blocktree.BlockTree{},
		db: polkadb.BlockDB{},
	}
}

// check checks to see if there an error if so writes err + message to terminal
func check(e error) {
	if e != nil {
		panic(e)
	}
}

var (
	// Data prefixes
	headerPrefix    = []byte("hdr") // headerPrefix + hash -> header
	blockDataPrefix = []byte("hsh") // blockDataPrefix + hash -> blockData
)

// headerKey = headerPrefix + hash
func headerKey(hash common.Hash) []byte {
	return append(headerPrefix, hash.ToBytes()...)
}

// blockDataKey = blockDataPrefix + hash
func blockDataKey(hash common.Hash) []byte {
	return append(blockDataPrefix, hash.ToBytes()...)
}

func (bs *blockState) GetHeader(hash common.Hash) types.BlockHeader {
	var result types.BlockHeader

	data, err := bs.bt.Db.Db.Get(headerKey(hash))
	check(err)

	err = json.Unmarshal(data, &result)
	check(err)

	return result
}

func (bs *blockState) GetBlockData(hash common.Hash) types.BlockData {
	var result types.BlockData

	data, err := bs.bt.Db.Db.Get(blockDataKey(hash))
	check(err)

	err = json.Unmarshal(data, &result)
	check(err)

	return result
}

func (bs *blockState) GetLatestBlock() types.BlockHeader {
	// Can't do yet
	return types.BlockHeader{}

}

func (bs *blockState) GetBlockByHash(hash common.Hash) types.Block {
	header := bs.GetHeader(hash)
	blockData := bs.GetBlockData(hash)
	blockBody := blockData.Body
	return types.Block{Header: header, Body: *blockBody}
}

func (bs *blockState) GetBlockByNumber(n *big.Int) types.Block {
	// Can't do yet
	return types.Block{}
}

func (bs *blockState) SetHeader(header types.BlockHeader) {
	rawdb.SetHeader(bs.db.Db, &header)
}

func (bs *blockState) SetBlockData(hash common.Hash, blockData types.BlockData) {
	rawdb.SetBlockData(bs.db.Db, &blockData)
}
