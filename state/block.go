package state

import (
	"encoding/binary"
	"encoding/json"
	"math/big"
	"reflect"

	"github.com/ChainSafe/gossamer/consensus/babe"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/blocktree"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
)

type blockState struct {
	bt          *blocktree.BlockTree
	db          *polkadb.BlockDB
	latestBlock types.BlockHeader
}

func NewBlockState(dataDir string) (*blockState, error) {
	blockDb, err := polkadb.NewBlockDB(dataDir)
	if err != nil {
		return nil, err
	}
	return &blockState{
		bt: &blocktree.BlockTree{},
		db: blockDb,
	}, nil
}

var (
	// Data prefixes
	headerPrefix     = []byte("hdr") // headerPrefix + hash -> header
	blockDataPrefix  = []byte("hsh") // blockDataPrefix + hash -> blockData
	babeHeaderPrefix = []byte("hba") // babeHeaderPrefix || epoch || slot -> babeHeader
	blockPrefix     = []byte("blk") // blockPrefix + hash -> block
)

// headerKey = headerPrefix + hash
func headerKey(hash common.Hash) []byte {
	return append(headerPrefix, hash.ToBytes()...)
}

// blockDataKey = blockDataPrefix + hash
func blockDataKey(hash common.Hash) []byte {
	return append(blockDataPrefix, hash.ToBytes()...)
}

// babeHeaderKey = babeHeaderPrefix || epoch || slice
func babeHeaderKey(epoch uint64, slot uint64) []byte {
	epochBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBytes, epoch)
	sliceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sliceBytes, slot)
	combined := append(epochBytes, sliceBytes...)
	return append(babeHeaderPrefix, combined...)
}
// blockKey = blockDataPrefix + hash
func blockKey(hash common.Hash) []byte {
	return append(blockPrefix, hash.ToBytes()...)
}


func (bs *blockState) GetHeader(hash common.Hash) (*types.BlockHeader, error) {
	var result *types.BlockHeader

	data, err := bs.db.Db.Get(headerKey(hash))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, result)
	return result, err
}

func (bs *blockState) GetBlockData(hash common.Hash) (types.BlockData, error) {
	var result types.BlockData

	data, err := bs.db.Db.Get(blockDataKey(hash))
	if err != nil {
		return types.BlockData{}, err
	}

	err = json.Unmarshal(data, &result)

	return result, err
}

func (bs *blockState) GetBabeHeader(epoch uint64, slot uint64) (babe.BabeHeader, error) {
	var result babe.BabeHeader

	data, err := bs.db.Db.Get(babeHeaderKey(epoch, slot))
	if err != nil {
		return babe.BabeHeader{}, err
	}

	err = json.Unmarshal(data, &result)

	return result, err
}

func (bs *blockState) GetLatestBlock() types.BlockHeader {
	return bs.latestBlock
}

func (bs *blockState) GetBlockByHash(hash common.Hash) (types.Block, error) {
	var result types.Block

	data, err := bs.db.Db.Get(blockKey(hash))
	if err != nil {
		return types.Block{}, err
	}
	blockData, err := bs.GetBlockData(hash)
	if err != nil {
		return types.Block{}, nil
	}
	blockBody := blockData.Body
	return types.Block{Header: header, Body: blockBody}, nil
}

func (bs *blockState) GetBlockByNumber(n *big.Int) types.Block {
	// Can't do yet
	return types.Block{}
}

func (bs *blockState) SetHeader(header types.BlockHeader) error {
	hash := header.Hash()

	// Write the encoded header
	bh, err := json.Marshal(header)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(headerKey(hash), bh)
	return err
}

func (bs *blockState) SetBlockData(hash common.Hash, blockData types.BlockData) error {
	// Write the encoded header
	bh, err := json.Marshal(blockData)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(blockDataKey(hash), bh)
	return err
}

func (bs *blockState) SetBabeHeader(epoch uint64, slot uint64, blockData babe.BabeHeader) error {
	// Write the encoded header
	bh, err := json.Marshal(blockData)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(babeHeaderKey(epoch, slot), bh)
	return err
}

func (bs *blockState) AddBlock(block types.Block) error {
	blockHeader := block.Header

	// Set the latest block
	if reflect.DeepEqual(bs.latestBlock, types.Block{}) || blockHeader.Number.Cmp(bs.latestBlock.Header.Number) == 1 {
		bs.latestBlock = block
	}

	// Write the encoded block
	bl, err := json.Marshal(block)
	if err != nil {
		return err
	}

	//Add the block to the DB
	err = bs.db.Db.Put(blockKey(blockHeader.Hash), bl)
	return err
}
