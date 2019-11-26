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
		bt: &blocktree.BlockTree{},
		db: polkadb.BlockDB{},
	}
}

func (bs *blockState) GetHeader(hash common.Hash) types.BlockHeader {
	return rawdb.GetHeader(bs.bt.Db.Db, hash)
}

func (bs *blockState) GetBlockData(hash common.Hash) types.BlockData {
	return rawdb.GetBlockData(bs.bt.Db.Db, hash)
}

func (bs *blockState) GetLatestBlock() types.BlockHeader {
	block := bs.bt.DeepestBlock()
	return block.Header

}

func (bs *blockState) GetBlockByHash(hash common.Hash) types.Block {
	// TODO: return the block given the hash
	return types.Block{}
}

func (bs *blockState) GetBlockByNumber(n *big.Int) types.Block {
	return *bs.bt.GetBlockFromBlockNumber(n)
}

func (bs *blockState) SetHeader(header types.BlockHeader) {
	rawdb.SetHeader(bs.db.Db, &header)
}

func (bs *blockState) SetBlockData(hash common.Hash, blockData types.BlockData) {
	rawdb.SetBlockData(bs.db.Db, &blockData)
}
