package state

import (
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
		bt: nil,
		db: polkadb.BlockDB{},
	}
}

func (bs *blockState) GetHeader(hash common.Hash) {
	rawdb.GetHeader()
}

func (bs *blockState) GetBlockData(hash common.Hash) {

}

func (bs *blockState) GetLatestBlock() types.BlockHeader {

}

func(bs *blockState)  GetBlockByHash(hash common.Hash) {

}

func (bs *blockState) GetBlockByNumber(n *big.Int) {

}


func (bs *blockState) SetHeader(header types.BlockHeader) {

}

func (bs *blockState) SetBlockData(hash common.Hash, header types.BlockHeader) {

}
