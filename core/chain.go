package core

import (
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	log "github.com/ChainSafe/log15"
)

type Chain struct {
	db *polkadb.BlockDB
}

func (c *Chain) SetBestHash(hash common.Hash) {
	bestHash := hash.ToBytes()
	err := c.Db.Put([]byte("bestHash"), bestHash)
	if err != nil {
		log.Crit("error", err)
	}
}
func (bd *polkadb.BlockDB) GetBestHash() common.Hash {
	h, err := bd.Db.Get([]byte("bestHash"))
	if err != nil {
		log.Crit("error", err)
	}
	return common.BytesToHash(h)
}
func (bd *polkadb.BlockDB) SetBestNumber(hash common.Hash) {
	bestHash := hash.ToBytes()
	err := bd.Db.Put([]byte("bestHash"), bestHash)
	if err != nil {
		log.Crit("error", err)
	}
}
func (bd *polkadb.BlockDB) GetBestNumber() common.Hash {
	h, err := bd.Db.Get([]byte("bestHash"))
	if err != nil {
		log.Crit("error", err)
	}
	return common.BytesToHash(h)
}
func (bd *polkadb.BlockDB) SetBlockData(blockData types.BlockData) {
	hash := blockData.Hash.ToBytes()
	data := common.ToBytes(blockData)
	err := bd.Db.Put(data, hash)
	if err != nil {
		log.Crit("error", err)
	}
}
func (bd *polkadb.BlockDB) SetBlockHeader(header types.BlockHeader) {
	hash := header.Hash.ToBytes()
	data := common.ToBytes(header)
	err := bd.Db.Put(data, hash)
	if err != nil {
		log.Crit("error", err)
	}
}
func (bd *polkadb.BlockDB) SetBlockHash(blockNumber *big.Int, hash common.Hash) {
	h := hash.ToBytes()
	bn := common.ToBytes(blockNumber)
	err := bd.Db.Put(bn, h)
	if err != nil {
		log.Crit("error", err)
	}
}
