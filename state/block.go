package state

import (
	"encoding/binary"
	"encoding/json"
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/blocktree"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
)

type blockState struct {
	Bt *blocktree.BlockTree
	Db *polkadb.BlockDB
}

func NewBlockState() *blockState {
	return &blockState{
		Bt: &blocktree.BlockTree{},
		Db: &polkadb.BlockDB{},
	}
}

var (
	// Data prefixes
	headerPrefix     = []byte("hdr") // headerPrefix + hash -> header
	blockDataPrefix  = []byte("bld") // blockDataPrefix + hash -> blockData
	headerHashPrefix = []byte("hsh") // headerHashPrefix + encodedBlockNum -> hash
)

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8) // encoding results in 8 bytes
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// headerKey = headerPrefix + hash
func headerKey(hash common.Hash) []byte {
	return append(headerPrefix, hash.ToBytes()...)
}

// headerHashKey = headerHashPrefix + num (uint64 big endian)
func headerHashKey(number uint64) []byte {
	return append(headerHashPrefix, encodeBlockNumber(number)...)
}

// blockDataKey = blockDataPrefix + hash
func blockDataKey(hash common.Hash) []byte {
	return append(blockDataPrefix, hash.ToBytes()...)
}

func (bs *blockState) GetHeader(hash common.Hash) (types.BlockHeaderWithHash, error) {
	var result types.BlockHeaderWithHash

	data, err := bs.Db.Db.Get(headerKey(hash))
	if err != nil {
		return types.BlockHeaderWithHash{}, err
	}

	err = json.Unmarshal(data, &result)

	return result, err
}

func (bs *blockState) GetBlockData(hash common.Hash) (types.BlockData, error) {
	var result types.BlockData

	data, err := bs.Db.Db.Get(blockDataKey(hash))
	if err != nil {
		return types.BlockData{}, err
	}

	err = json.Unmarshal(data, &result)

	return result, err
}

func (bs *blockState) GetLatestBlock() types.BlockHeaderWithHash {
	// Can't do yet
	return types.BlockHeaderWithHash{}

}

func (bs *blockState) GetBlockByHash(hash common.Hash) (types.Block, error) {
	header, err := bs.GetHeader(hash)
	if err != nil {
		return types.Block{}, err
	}
	blockData, err := bs.GetBlockData(hash)
	if err != nil {
		return types.Block{}, err
	}
	blockBody := blockData.Body
	return types.Block{Header: header, Body: *blockBody}, nil
}

func (bs *blockState) GetBlockByNumber(n *big.Int) (types.Block, error) {
	// First retrieve the block hash based on the block number from the database
	hash, err := bs.Db.Db.Get(headerHashKey(n.Uint64()))
	if err != nil {
		return types.Block{}, err
	}

	// Then find the block based on the hash
	endHash := common.NewHash(hash)
	block, err := bs.GetBlockByHash(endHash)
	return block, err
}

func (bs *blockState) SetHeader(header types.BlockHeaderWithHash) error {
	hash := header.Hash

	// Write the encoded header
	bh, err := json.Marshal(header)
	if err != nil {
		return err
	}

	err = bs.Db.Db.Put(headerKey(hash), bh)
	if err != nil {
		return err
	}

	// Add a mapping of [blocknumber : hash] for retrieving the block by number
	err = bs.Db.Db.Put(headerHashKey(header.Number.Uint64()), header.Hash.ToBytes())
	return err
}

func (bs *blockState) SetBlockData(hash common.Hash, blockData types.BlockData) error {
	// Write the encoded header
	bh, err := json.Marshal(blockData)
	if err != nil {
		return err
	}

	err = bs.Db.Db.Put(blockDataKey(hash), bh)
	return err
}
