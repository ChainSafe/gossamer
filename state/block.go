package state

import (
	//"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"

	//"github.com/ChainSafe/gossamer/consensus/babe"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/blocktree"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
)

type blockState struct {
	bt          *blocktree.BlockTree
	db          *polkadb.BlockDB
	latestBlock *types.BlockHeader
}

func NewBlockState(dataDir string, latestHash common.Hash) (*blockState, error) {
	blockDb, err := polkadb.NewBlockDB(dataDir)
	if err != nil {
		return nil, err
	}

	bs := &blockState{
		bt: &blocktree.BlockTree{},
		db: blockDb,
	}

	latestBlock, err := bs.GetHeader(latestHash)
	if err != nil {
		return bs, fmt.Errorf("NewBlockState latestBlock err: %s", err)
	}

	bs.latestBlock = latestBlock
	return bs, nil
}

func NewBlockStateFromGenesis(dataDir string, header *types.BlockHeader) (*blockState, error) {
	blockDb, err := polkadb.NewBlockDB(dataDir)
	if err != nil {
		return nil, err
	}

	bs := &blockState{
		bt: &blocktree.BlockTree{},
		db: blockDb,
	}

	err = bs.SetHeader(*header)
	if err != nil {
		return nil, err
	}

	bs.latestBlock = header
	return bs, nil
}

var (
	// Data prefixes
	headerPrefix    = []byte("hdr") // headerPrefix + hash -> header
	blockDataPrefix = []byte("hsh") // blockDataPrefix + hash -> blockData
	//babeHeaderPrefix = []byte("hba") // babeHeaderPrefix || epoch || slot -> babeHeader
)

// headerKey = headerPrefix + hash
func headerKey(hash common.Hash) []byte {
	return append(headerPrefix, hash.ToBytes()...)
}

// blockDataKey = blockDataPrefix + hash
func blockDataKey(hash common.Hash) []byte {
	return append(blockDataPrefix, hash.ToBytes()...)
}

func (bs *blockState) GetHeader(hash common.Hash) (*types.BlockHeader, error) {
	result := new(types.BlockHeader)

	data, err := bs.db.Db.Get(headerKey(hash))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, result)
	result.Hash()
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

func (bs *blockState) GetLatestBlockHeader() *types.BlockHeader {
	return bs.latestBlock
}

func (bs *blockState) GetBlockByHash(hash common.Hash) (types.Block, error) {
	header, err := bs.GetHeader(hash)
	if err != nil {
		return types.Block{}, nil
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

func (bs *blockState) SetBlockData(hash common.Hash, blockData types.BlockHeader) error {
	// Write the encoded header
	bh, err := json.Marshal(blockData)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(blockDataKey(hash), bh)
	return err
}

func (bs *blockState) AddBlock(block types.BlockHeader) error {
	// Set the latest block
	if block.Number.Cmp(bs.latestBlock.Number) == 1 {
		bs.latestBlock = &block
	}

	//TODO: Implement Add Block
	return nil
}

// TODO: this causes a circular dependency since BABE imports state.
// need to refactor, perhaps by putting babe types in a subpackage.

// // babeHeaderKey = babeHeaderPrefix || epoch || slice
// func babeHeaderKey(epoch uint64, slot uint64) []byte {
// 	epochBytes := make([]byte, 8)
// 	binary.LittleEndian.PutUint64(epochBytes, epoch)
// 	sliceBytes := make([]byte, 8)
// 	binary.LittleEndian.PutUint64(sliceBytes, slot)
// 	combined := append(epochBytes, sliceBytes...)
// 	return append(babeHeaderPrefix, combined...)
// }

// func (bs *blockState) GetBabeHeader(epoch uint64, slot uint64) (babe.BabeHeader, error) {
// 	var result babe.BabeHeader

// 	data, err := bs.db.Db.Get(babeHeaderKey(epoch, slot))
// 	if err != nil {
// 		return babe.BabeHeader{}, err
// 	}

// 	err = json.Unmarshal(data, &result)

// 	return result, err
// }

// func (bs *blockState) SetBabeHeader(epoch uint64, slot uint64, blockData babe.BabeHeader) error {
// 	// Write the encoded header
// 	bh, err := json.Marshal(blockData)
// 	if err != nil {
// 		return err
// 	}

// 	err = bs.db.Db.Put(babeHeaderKey(epoch, slot), bh)
// 	return err
// }
