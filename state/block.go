package state

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sync"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/common/optional"
	babetypes "github.com/ChainSafe/gossamer/consensus/babe/types"
	"github.com/ChainSafe/gossamer/core/blocktree"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/db"
)

// BlockDB stores block's in an underlying Database
type BlockDB struct {
	Db db.Database
}

// BlockState defines fields for manipulating the state of blocks, such as BlockTree, BlockDB and Header
type BlockState struct {
	bt   *blocktree.BlockTree
	db   *BlockDB
	lock sync.RWMutex
}

// NewBlockDB instantiates a badgerDB instance for storing relevant BlockData
func NewBlockDB(dataDir string) (*BlockDB, error) {
	db, err := db.NewBadgerDB(dataDir)
	if err != nil {
		return nil, err
	}

	return &BlockDB{
		Db: db,
	}, nil
}

// NewBlockState will create a new BlockState backed by the database located at dataDir
func NewBlockState(db *BlockDB, bt *blocktree.BlockTree) (*BlockState, error) {
	if bt == nil {
		return nil, fmt.Errorf("block tree is nil")
	}

	bs := &BlockState{
		bt: bt,
		db: db,
	}

	return bs, nil
}

// NewBlockStateFromGenesis initializes a BlockState from a genesis header, saving it to the database located at dataDir
func NewBlockStateFromGenesis(dataDir string, header *types.Header) (*BlockState, error) {
	blockDb, err := NewBlockDB(dataDir)
	if err != nil {
		return nil, err
	}

	bs := &BlockState{
		bt: blocktree.NewBlockTreeFromGenesis(header, blockDb.Db),
		db: blockDb,
	}

	err = bs.SetHeader(header)
	if err != nil {
		return nil, err
	}

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
func (bs *BlockState) GetHeader(hash common.Hash) (*types.Header, error) {
	result := new(types.Header)

	data, err := bs.db.Db.Get(headerKey(hash))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, result)
	if reflect.DeepEqual(result, new(types.Header)) {
		return nil, fmt.Errorf("header does not exist")
	}

	result.Hash()
	return result, err
}

// GetBlockData returns a BlockData for a given hash
func (bs *BlockState) GetBlockData(hash common.Hash) (*types.BlockData, error) {
	result := new(types.BlockData)

	data, err := bs.db.Db.Get(blockDataKey(hash))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, result)
	if err != nil {
		return result, err
	}

	if result.Header == nil {
		result.Header = optional.NewHeader(false, nil)
	}

	if result.Body == nil {
		result.Body = optional.NewBody(false, nil)
	}

	if result.Receipt == nil {
		result.Receipt = optional.NewBytes(false, nil)
	}

	if result.MessageQueue == nil {
		result.MessageQueue = optional.NewBytes(false, nil)
	}

	if result.Justification == nil {
		result.Justification = optional.NewBytes(false, nil)
	}

	return result, nil
}

// GetBlockByHash returns a block for a given hash
func (bs *BlockState) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	header, err := bs.GetHeader(hash)
	if err != nil {
		return nil, err
	}

	blockData, err := bs.GetBlockData(hash)
	if err != nil {
		return nil, err
	}

	body, err := types.NewBodyFromOptional(blockData.Body)
	if err != nil {
		return nil, err
	}
	return &types.Block{Header: header, Body: body}, nil
}

// GetBlockByNumber returns a block for a given blockNumber
func (bs *BlockState) GetBlockByNumber(blockNumber *big.Int) (*types.Block, error) {
	// First retrieve the block hash in a byte array based on the block number from the database
	byteHash, err := bs.db.Db.Get(headerHashKey(blockNumber.Uint64()))
	if err != nil {
		return nil, fmt.Errorf("cannot get block %d: %s", blockNumber, err)
	}

	// Then find the block based on the hash
	hash := common.NewHash(byteHash)
	block, err := bs.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// SetHeader will set the header into DB
func (bs *BlockState) SetHeader(header *types.Header) error {
	bs.lock.Lock()
	defer bs.lock.Unlock()

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

// SetBlock will add a block to the DB
func (bs *BlockState) SetBlock(block *types.Block) error {
	// Add the blockHeader to the DB
	err := bs.SetHeader(block.Header)
	if err != nil {
		return err
	}

	blockData := &types.BlockData{
		Hash:   block.Header.Hash(),
		Header: block.Header.AsOptional(),
		Body:   block.Body.AsOptional(),
	}
	return bs.SetBlockData(blockData)
}

// SetBlockData will set the block data using given hash and blockData into DB
func (bs *BlockState) SetBlockData(blockData *types.BlockData) error {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	// Write the encoded header
	bh, err := json.Marshal(blockData)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(blockDataKey(blockData.Hash), bh)
	return err
}

// AddBlock adds a block to the blocktree and the DB
func (bs *BlockState) AddBlock(block *types.Block) error {
	// add block to blocktree
	err := bs.bt.AddBlock(block)
	if err != nil {
		return err
	}

	// add the header to the DB
	err = bs.SetHeader(block.Header)
	if err != nil {
		return err
	}
	hash := block.Header.Hash()

	// add block data to the DB
	bd := &types.BlockData{
		Hash:   hash,
		Header: block.Header.AsOptional(),
		Body:   block.Body.AsOptional(),
	}
	err = bs.SetBlockData(bd)
	return err
}

// ChainHead returns the hash of the head of the current chain
// rename BestBlockHash ?
func (bs *BlockState) ChainHead() common.Hash {
	return bs.bt.DeepestBlockHash()
}

// ChainHeadAsHeader returns the block header of the current head of the chain
// rename BestBlockHeader?
func (bs *BlockState) ChainHeadAsHeader() (*types.Header, error) {
	return bs.GetHeader(bs.ChainHead())
}

// SubChain returns the sub-blockchain between the starting hash and the ending hash using the block tree
func (bs *BlockState) SubChain(start, end common.Hash) []common.Hash {
	return bs.bt.SubBlockchain(start, end)
}

// ComputeSlotForBlock returns the slot number for a given block
// TODO: can move this out of the blocktree into BABE
func (bs *BlockState) ComputeSlotForBlock(block *types.Block, slotDuration uint64) uint64 {
	return bs.bt.ComputeSlotForBlock(block, slotDuration)
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

// GetBabeHeader retrieves a BabeHeader from the database
func (bs *BlockState) GetBabeHeader(epoch uint64, slot uint64) (*babetypes.BabeHeader, error) {
	result := new(babetypes.BabeHeader)

	data, err := bs.db.Db.Get(babeHeaderKey(epoch, slot))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, result)

	return result, err
}

// SetBabeHeader sets a BabeHeader in the database
func (bs *BlockState) SetBabeHeader(epoch uint64, slot uint64, bh *babetypes.BabeHeader) error {
	// Write the encoded header
	enc, err := json.Marshal(bh)
	if err != nil {
		return err
	}

	err = bs.db.Db.Put(babeHeaderKey(epoch, slot), enc)
	return err
}
