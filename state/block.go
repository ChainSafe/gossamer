package state

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	babetypes "github.com/ChainSafe/gossamer/consensus/babe/types"
	"github.com/ChainSafe/gossamer/core/blocktree"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
)

type blockState struct {
	bt          *blocktree.BlockTree
	db          *polkadb.BlockDB
	latestBlock *types.BlockHeader
}

// NewBlockState will create a new blockState backed by the database located at dataDir
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
	headerPrefix     = []byte("hdr") // headerPrefix + hash -> header
	babeHeaderPrefix = []byte("hba") // babeHeaderPrefix || epoch || slot -> babeHeader
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

// GetHeader returns a BlockHeader for a given hash
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

// GetBlockData returns a BlockData for a given hash
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

// GetBlockByHash returns a block for a given hash
func (bs *blockState) GetBlockByHash(hash common.Hash) (types.Block, error) {
	header, err := bs.GetHeader(hash)
	if err != nil {
		return types.Block{}, err
	}

	blockData, err := bs.GetBlockData(hash)
	if err != nil {
		return types.Block{}, err
	}

	return types.Block{Header: header, Body: blockData.Body}, nil
}

// GetBlockByNumber returns a block for a given blockNumber
func (bs *blockState) GetBlockByNumber(blockNumber *big.Int) (types.Block, error) {
	// First retrieve the block hash in a byte array based on the block number from the database
	byteHash, err := bs.db.Db.Get(headerHashKey(blockNumber.Uint64()))
	if err != nil {
		return types.Block{}, err
	}

	// Then find the block based on the hash
	hash := common.NewHash(byteHash)
	block, err := bs.GetBlockByHash(hash)
	if err != nil {
		return types.Block{}, err
	}

	return block, nil
}

// SetHeader will set the header into DB
func (bs *blockState) SetHeader(header types.BlockHeader) error {
	hash := header.Hash()

	// Write the encoded header
	bh, err := json.Marshal(header)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(headerKey(hash), bh)
	if err != nil {
		return err
	}

	// Add a mapping of [blocknumber : hash] for retrieving the block by number
	err = bs.db.Db.Put(headerHashKey(header.Number.Uint64()), header.Hash().ToBytes())
	return err
}

// SetBlockData will set the block data using given hash and blockData into DB
func (bs *blockState) SetBlockData(hash common.Hash, blockData types.BlockData) error {
	// Write the encoded header
	bh, err := json.Marshal(blockData)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(blockDataKey(hash), bh)
	return err
}

// AddBlock will add a block into the DB
func (bs *blockState) AddBlock(block types.BlockHeader) error {
	// Set the latest block
	if block.Number.Cmp(bs.latestBlock.Number) == 1 {
		bs.latestBlock = &block
	}

	//TODO: Implement Add Block
	return nil
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

func (bs *blockState) GetBabeHeader(epoch uint64, slot uint64) (*babetypes.BabeHeader, error) {
	result := new(babetypes.BabeHeader)

	data, err := bs.db.Db.Get(babeHeaderKey(epoch, slot))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, result)

	return result, err
}

func (bs *blockState) SetBabeHeader(epoch uint64, slot uint64, blockData *babetypes.BabeHeader) error {
	// Write the encoded header
	bh, err := json.Marshal(blockData)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(babeHeaderKey(epoch, slot), bh)
	return err
}
