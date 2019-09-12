package polkadb

import (
	"github.com/ChainSafe/gossamer/core"
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	log "github.com/ChainSafe/log15"
)

func (c *BlockDB) SetBestHash(hash common.Hash) {
	bestHash := hash.ToBytes()
	err := c.Db.Put([]byte("bestHash"), bestHash)
	if err != nil {
		log.Crit("error", err)
	}
}

func (c *BlockDB) GetBestHash() common.Hash {
	h, err := c.Db.Get([]byte("bestHash"))
	if err != nil {
		log.Crit("error", err)
	}
	return common.BytesToHash(h)
}

func (c *BlockDB) SetBestNumber(hash common.Hash) {
	bestHash := hash.ToBytes()
	err := c.Db.Put([]byte("bestHash"), bestHash)
	if err != nil {
		log.Crit("error", err)
	}
}

func (c *BlockDB) GetBestNumber() common.Hash {
	h, err := c.Db.Get([]byte("bestHash"))
	if err != nil {
		log.Crit("error", err)
	}
	return common.BytesToHash(h)
}

func (c *BlockDB) SetBlockData(blockData core.BlockData) {
	hash := blockData.Hash.ToBytes()
	bd := common.ToBytes(blockData)
	err := c.Db.Put(bd, hash)
	if err != nil {
		log.Crit("error", err)
	}
}

func (c *BlockDB) SetBlockHeader(header core.BlockHeader) {
	hash := header.Hash.ToBytes()
	bd := common.ToBytes(header)
	err := c.Db.Put(bd, hash)
	if err != nil {
		log.Crit("error", err)
	}
}

func (c *BlockDB) SetBlockHash(blockNumber *big.Int, hash common.Hash) {
	h := hash.ToBytes()
	bn := common.ToBytes(blockNumber)
	err := c.Db.Put(bn, h)
	if err != nil {
		log.Crit("error", err)
	}
}
