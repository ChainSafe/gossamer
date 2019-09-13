package polkadb

import (
	"github.com/ChainSafe/gossamer/core"
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	log "github.com/ChainSafe/log15"
)

func (bd *BlockDB) SetBestHash(hash common.Hash) {
	bestHash := hash.ToBytes()
	err := bd.Db.Put([]byte("bestHash"), bestHash)
	if err != nil {
		log.Crit("error", err)
	}
}

func (bd *BlockDB) GetBestHash() common.Hash {
	h, err := bd.Db.Get([]byte("bestHash"))
	if err != nil {
		log.Crit("error", err)
	}
	return common.BytesToHash(h)
}

func (bd *BlockDB) SetBestNumber(hash common.Hash) {
	bestHash := hash.ToBytes()
	err := bd.Db.Put([]byte("bestHash"), bestHash)
	if err != nil {
		log.Crit("error", err)
	}
}

func (bd *BlockDB) GetBestNumber() common.Hash {
	h, err := bd.Db.Get([]byte("bestHash"))
	if err != nil {
		log.Crit("error", err)
	}
	return common.BytesToHash(h)
}

func (bd *BlockDB) SetBlockData(blockData core.BlockData) {
	hash := blockData.Hash.ToBytes()
	data := common.ToBytes(blockData)
	err := bd.Db.Put(data, hash)
	if err != nil {
		log.Crit("error", err)
	}
}

func (bd *BlockDB) SetBlockHeader(header core.BlockHeader) {
	hash := header.Hash.ToBytes()
	data := common.ToBytes(header)
	err := bd.Db.Put(data, hash)
	if err != nil {
		log.Crit("error", err)
	}
}

func (bd *BlockDB) SetBlockHash(blockNumber *big.Int, hash common.Hash) {
	h := hash.ToBytes()
	bn := common.ToBytes(blockNumber)
	err := bd.Db.Put(bn, h)
	if err != nil {
		log.Crit("error", err)
	}
}
