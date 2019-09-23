package rawdb

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
	"math/big"

	log "github.com/ChainSafe/log15"
)

type Chain struct{}

// SetHeader stores a block header into the database and also stores the hash-
// to-number mapping.
func (c *Chain) SetHeader(db polkadb.Writer, header *types.BlockHeader) {
	var (
		hash   = header.Hash
	)

	// Write the encoded header
	data := common.ToBytes(header)

	if err := db.Put(hash.ToBytes(), data); err != nil {
		log.Crit("Failed to store header", "err", err)
	}
}

func (c *Chain) SetBlockData(db polkadb.Writer, blockData *types.BlockData) {}
func (c *Chain) SetBlockHash(db polkadb.Writer, num *big.Int) {}
func (c *Chain) SetBestHash(db polkadb.Writer, hash *common.Hash) {}
func (c *Chain) SetBestNumber(db polkadb.Writer, num *big.Int) {}

// BLOCK READS

func (c *Chain) GetBlockHeader(db polkadb.Reader, hash *common.Hash) *types.BlockHeader { return &types.BlockHeader{}}
func (c *Chain) GetBlockData(db polkadb.Reader, hash *common.Hash) *types.BlockData { return &types.BlockData{}}
func (c *Chain) GetBlockHash(db polkadb.Reader, num *big.Int) *common.Hash { return &common.Hash{}}
func (c *Chain) GetBestHash(db polkadb.Reader) *common.Hash { return &common.Hash{}}
func (c *Chain) GetBestNumber(db polkadb.Reader) *big.Int { return &big.Int{}}


