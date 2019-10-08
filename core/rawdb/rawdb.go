package rawdb

import (
	"bytes"
	"encoding/json"
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
	"math/big"

	log "github.com/ChainSafe/log15"
)

func check(e error, msg string) {
	if e != nil {
		log.Warn(msg, "err", e)
	}
}

// SetHeader stores a block header into the database
func SetHeader(db polkadb.Writer, header *types.BlockHeader) {
	hash := header.Hash

	// Write the encoded header
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(header); err != nil {
		log.Crit("error encoding header to bytes", "err", err)
	}

	if err := db.Put(headerKey(hash), buf.Bytes()); err != nil {
		log.Crit("Failed to store header", "err", err)
	}
}

func GetHeader(db polkadb.Reader, hash common.Hash) *types.BlockHeader {
	var result *types.BlockHeader
	data, err := db.Get(headerKey(hash))
	check(err, "Failed to retrieve block header")

	err = json.Unmarshal(data, &result)
	check(err, "Failed to unmarshal block header")
	return result
}

// BLOCK WRITES

func SetBlockData(db polkadb.Writer, blockData *types.BlockData) {}
func SetBlockHash(db polkadb.Writer, num *big.Int) {}
func SetBestHash(db polkadb.Writer, hash *common.Hash) {}
func SetBestNumber(db polkadb.Writer, num *big.Int) {}

// BLOCK READS

func GetBlockHeader(db polkadb.Reader, hash *common.Hash) *types.BlockHeader { return &types.BlockHeader{}}
func GetBlockData(db polkadb.Reader, hash *common.Hash) *types.BlockData { return &types.BlockData{}}
func GetBlockHash(db polkadb.Reader, num *big.Int) *common.Hash { return &common.Hash{}}
func GetBestHash(db polkadb.Reader) *common.Hash { return &common.Hash{}}
func GetBestNumber(db polkadb.Reader) *big.Int { return &big.Int{}}
