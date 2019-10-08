package rawdb

import (
	"encoding/json"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
	log "github.com/ChainSafe/log15"
)

// check checks to see if there an error if so writes err + message to terminal
func check(e error, msg string) {
	if e != nil {
		log.Crit(msg, "err", e)
	}
}

// SetHeader stores a block header into the KV-store; key is headerPrefix + hash
func SetHeader(db polkadb.Writer, header *types.BlockHeader) {
	hash := header.Hash

	// Write the encoded header
	bh, err := json.Marshal(header)
	check(err, "Failed to encode header to bytes")

	err = db.Put(headerKey(hash), bh)
	check(err, "Failed to store header")
}

// GetHeader retrieves block header from KV-store using headerKey
func GetHeader(db polkadb.Reader, hash common.Hash) *types.BlockHeader {
	var result *types.BlockHeader
	data, err := db.Get(headerKey(hash))
	check(err, "Failed to retrieve block header")

	err = json.Unmarshal(data, &result)
	check(err, "Failed to unmarshal block header")
	return result
}

// SetBlockData writes blockData to KV-store; key is blockDataPrefix + hash
func SetBlockData(db polkadb.Writer, blockData *types.BlockData) {
	hash := blockData.Hash

	// Write the encoded header
	bh, err := json.Marshal(blockData)
	check(err, "Failed to encode blockData to bytes")

	err = db.Put(blockDataKey(hash), bh)
	check(err, "Failed to store blockData")
}

// GetBlockData retrieves blockData from KV-store using blockDataKey
func GetBlockData(db polkadb.Reader, hash common.Hash) *types.BlockData {
	var result *types.BlockData
	data, err := db.Get(blockDataKey(hash))
	check(err, "Failed to retrieve block header")

	err = json.Unmarshal(data, &result)
	check(err, "Failed to unmarshal block header")
	return result
}
