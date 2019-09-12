package polkadb

import (
	log "github.com/ChainSafe/log15"
	"github.com/ChainSafe/gossamer/common"
)


func (bd *BlockDB) Set(hash common.Hash) {
	bestHash := hash.ToBytes()
	err := bd.Db.Put([]byte("bestHash"), bestHash)
	if err != nil {
		log.Crit("error", err)
	}
}

func (bd *BlockDB) Get() common.Hash {
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