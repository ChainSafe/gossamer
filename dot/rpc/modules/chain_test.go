package modules

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

func newChainService(t *testing.T) *state.Service {
	testDir := utils.NewTestDir(t)

	defer utils.RemoveTestDir(t)

	stateSrvc := state.NewService(testDir)
	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), trie.EmptyHash, trie.EmptyHash, [][]byte{})
	require.Nil(t, err)

	tr := trie.NewEmptyTrie()

	stateSrvc.UseMemDB()

	genesisData := new(genesis.Data)

	err = stateSrvc.Initialize(genesisData, genesisHeader, tr)
	require.Nil(t, err)

	err = stateSrvc.Start()
	require.Nil(t, err)

	err = loadTestBlocks(genesisHeader.Hash(), stateSrvc.Block)
	require.Nil(t, err)

	return stateSrvc
}

func loadTestBlocks(gh common.Hash, bs *state.BlockState) error {
	// Create header
	header0 := &types.Header{
		Number:     big.NewInt(0),
		Digest:     [][]byte{},
		ParentHash: gh,
	}
	// Create blockHash
	blockHash0 := header0.Hash()
	// BlockBody with fake extrinsics
	blockBody0 := types.Body{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	block0 := &types.Block{
		Header: header0,
		Body:   &blockBody0,
	}

	err := bs.AddBlock(block0)
	if err != nil {
		return err
	}

	// Create header & blockData for block 1
	header1 := &types.Header{
		Number:     big.NewInt(1),
		Digest:     [][]byte{},
		ParentHash: blockHash0,
	}

	// Create Block with fake extrinsics
	blockBody1 := types.Body{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	block1 := &types.Block{
		Header: header1,
		Body:   &blockBody1,
	}

	// Add the block1 to the DB
	err = bs.AddBlock(block1)
	if err != nil {
		return err
	}

	return nil
}

func TestChainGetHeader_Genesis(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)
	expected := &ChainBlockHeaderResponse{
		ParentHash:     "0x0000000000000000000000000000000000000000000000000000000000000000",
		Number:         big.NewInt(0),
		StateRoot:      "0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314",
		ExtrinsicsRoot: "0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314",
		Digest:         [][]byte{},
	}
	res := &ChainBlockHeaderResponse{}
	req := ChainHashRequest("0xc375f478c6887dbcc2d1a4dbcc25f330b3df419325ece49cddfe5a0555663b7e")
	err := svc.GetHeader(nil, &req, res)
	require.Nil(t, err)

	require.Equal(t, expected, res)
}

func TestChainGetHeader_Latest(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)
	expected := &ChainBlockHeaderResponse{
		ParentHash:     "0xdbfdd87392d9ee52f499610582737daceecf83dc3ad7946fcadeb01c86e1ef75",
		Number:         big.NewInt(1),
		StateRoot:      "0x0000000000000000000000000000000000000000000000000000000000000000",
		ExtrinsicsRoot: "0x0000000000000000000000000000000000000000000000000000000000000000",
		Digest:         [][]byte{},
	}
	res := &ChainBlockHeaderResponse{}
	req := ChainHashRequest("") // empty request should return latest hash
	err := svc.GetHeader(nil, &req, res)
	require.Nil(t, err)

	require.Equal(t, expected, res)
}

func TestChainGetHeader_NotFound(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)

	res := &ChainBlockHeaderResponse{}
	req := ChainHashRequest("0xea374832a2c3997280d2772c10e6e5b0b493ccd3d09c0ab14050320e34076c2c")
	err := svc.GetHeader(nil, &req, res)
	require.EqualError(t, err, "Key not found")
}

func TestChainGetHeader_Error(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)

	res := &ChainBlockHeaderResponse{}
	req := ChainHashRequest("zz")
	err := svc.GetHeader(nil, &req, res)
	require.EqualError(t, err, "could not byteify non 0x prefixed string")
}

func TestChainGetBlock_Genesis(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)
	header := &ChainBlockHeaderResponse{
		ParentHash:     "0x0000000000000000000000000000000000000000000000000000000000000000",
		Number:         big.NewInt(0),
		StateRoot:      "0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314",
		ExtrinsicsRoot: "0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314",
		Digest:         [][]byte{},
	}
	expected := &ChainBlockResponse{
		Block: ChainBlock{
			Header: *header,
			Body:   nil,
		},
	}

	res := &ChainBlockResponse{}
	req := ChainHashRequest("0xc375f478c6887dbcc2d1a4dbcc25f330b3df419325ece49cddfe5a0555663b7e")
	err := svc.GetBlock(nil, &req, res)
	require.Nil(t, err)

	require.Equal(t, expected, res)
}

func TestChainGetBlock_Latest(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)
	header := &ChainBlockHeaderResponse{
		ParentHash:     "0xdbfdd87392d9ee52f499610582737daceecf83dc3ad7946fcadeb01c86e1ef75",
		Number:         big.NewInt(1),
		StateRoot:      "0x0000000000000000000000000000000000000000000000000000000000000000",
		ExtrinsicsRoot: "0x0000000000000000000000000000000000000000000000000000000000000000",
		Digest:         [][]byte{},
	}
	expected := &ChainBlockResponse{
		Block: ChainBlock{
			Header: *header,
			Body:   nil,
		},
	}

	res := &ChainBlockResponse{}
	req := ChainHashRequest("") // empty request should return latest block
	err := svc.GetBlock(nil, &req, res)
	require.Nil(t, err)

	require.Equal(t, expected, res)
}

func TestChainGetBlock_NoFound(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)

	res := &ChainBlockResponse{}
	req := ChainHashRequest("0xea374832a2c3997280d2772c10e6e5b0b493ccd3d09c0ab14050320e34076c2c")
	err := svc.GetBlock(nil, &req, res)
	require.EqualError(t, err, "Key not found")
}

func TestChainGetBlock_Error(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)

	res := &ChainBlockResponse{}
	req := ChainHashRequest("zz")
	err := svc.GetBlock(nil, &req, res)
	require.EqualError(t, err, "could not byteify non 0x prefixed string")
}

func TestChainGetBlockHash_Latest(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)

	resString := string("")
	res := ChainHashResponse(resString)
	req := ChainBlockNumberRequest(nil)
	err := svc.GetBlockHash(nil, &req, &res)

	require.Nil(t, err)

	require.Equal(t, "0x80d653de440352760f89366c302c02a92ab059f396e2bfbf7f860e6e256cd698", res)
}

func TestChainGetBlockHash_ByNumber(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)

	resString := string("")
	res := ChainHashResponse(resString)
	req := ChainBlockNumberRequest("1")
	err := svc.GetBlockHash(nil, &req, &res)

	require.Nil(t, err)

	require.Equal(t, "0x80d653de440352760f89366c302c02a92ab059f396e2bfbf7f860e6e256cd698", res)
}

func TestChainGetBlockHash_ByHex(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)

	resString := string("")
	res := ChainHashResponse(resString)
	req := ChainBlockNumberRequest("0x01")
	err := svc.GetBlockHash(nil, &req, &res)

	require.Nil(t, err)

	require.Equal(t, "0x80d653de440352760f89366c302c02a92ab059f396e2bfbf7f860e6e256cd698", res)
}

func TestChainGetBlockHash_Array(t *testing.T) {
	chain := newChainService(t)
	svc := NewChainModule(chain.Block)

	resString := string("")
	res := ChainHashResponse(resString)
	nums := make([]interface{}, 2)
	nums[0] = float64(0)     // as number
	nums[1] = string("0x01") // as hex string
	req := ChainBlockNumberRequest(nums)
	err := svc.GetBlockHash(nil, &req, &res)

	require.Nil(t, err)

	require.Equal(t, []string{"0xdbfdd87392d9ee52f499610582737daceecf83dc3ad7946fcadeb01c86e1ef75", "0x80d653de440352760f89366c302c02a92ab059f396e2bfbf7f860e6e256cd698"}, res)
}
