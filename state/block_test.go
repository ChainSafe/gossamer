package state

import (
	"io/ioutil"
	"math/big"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/core/types"
)

func TestGetBlockByNumber(t *testing.T) {
	dataDir, err := ioutil.TempDir("", "TestGetBlockByNumber")
	if err != nil {
		t.Error("Failed to create temp folder for TestGetBlockByNumber test", "err", err)
		return
	}
	// Create & start a new State service
	stateService := NewService(dataDir)
	err = stateService.Start()
	if err != nil {
		t.Fatal(err)
	}

	// Create a header & blockData
	blockHeader := &types.BlockHeader{
		Number: big.NewInt(1),
	}
	hash := blockHeader.Hash()

	// BlockBody with fake extrinsics
	blockBody := &types.BlockBody{}
	*blockBody = append(*blockBody, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}...)

	blockData := types.BlockData{
		Hash:   hash,
		Header: blockHeader,
		Body:   blockBody,
	}

	// Set the block's header & blockData in the blockState
	// SetHeader also sets mapping [blockNumber : hash] in DB
	err = stateService.Block.SetHeader(*blockHeader)
	if err != nil {
		t.Fatal(err)
	}

	err = stateService.Block.SetBlockData(hash, blockData)
	if err != nil {
		t.Fatal(err)
	}

	// Get block & check if it's the same as the expectedBlock
	expectedBlock := types.Block{
		Header: blockHeader,
		Body:   blockBody,
	}
	retBlock, err := stateService.Block.GetBlockByNumber(blockHeader.Number)
	if err != nil {
		t.Fatal(err)
	}
	retBlock.Header.Hash()

	if !reflect.DeepEqual(expectedBlock, retBlock) {
		t.Fatalf("Fail: got %+v\nexpected %+v", retBlock, expectedBlock)
	}

}


func TestAddBlock(t *testing.T) {
	dataDir, err := ioutil.TempDir("", "TestGetBlockByNumber")
	if err != nil {
		t.Error("Failed to create temp folder for TestGetBlockByNumber test", "err", err)
		return
	}

	//Create a new blockState
	blockState, err := NewBlockState(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	//Close DB & erase data dir contents
	defer func() {
		err = blockState.db.Db.Close()
		if err != nil {
			t.Fatal("BlockDB close err: ", err)
		}
	}()

	// Create block0 & call AddBlock
	// Create header & blockBody for block0
	header0 := &types.BlockHeader{
		Number: big.NewInt(0),
	}
	blockHash0 := header0.Hash()

	// BlockBody with fake extrinsics
	blockBody0 := types.BlockBody{}
	blockBody0 = append(blockBody0, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}...)

	block0 := types.Block{
		Header: header0,
		Body:   &blockBody0,
	}

	// Add the block0 to the DB
	blockState.AddBlock(block0)

	// Create block1 & call AddBlock
	// Create header & blockdata for block 1
	header1 := &types.BlockHeader{
		Number: big.NewInt(1),
	}
	blockHash1 := header1.Hash()

	// BlockBody with fake extrinsics
	blockBody1 := types.BlockBody{}
	blockBody1 = append(blockBody1, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}...)

	block1 := types.Block{
		Header: header1,
		Body:   &blockBody1,
	}

	// Add the block1 to the DB
	blockState.AddBlock(block1)

	// Get the blocks & check if it's the same as the added blocks
	retBlock, err := blockState.GetBlockByHash(blockHash0)
	if err != nil {
		t.Fatal(err)
	}

	retBlock.Header.Hash()

	if !reflect.DeepEqual(block0, retBlock) {
		t.Fatalf("Fail: got %+v\nexpected %+v", retBlock, block0)
	}

	retBlock, err = blockState.GetBlockByHash(blockHash1)
	if err != nil {
		t.Fatal(err)
	}

	retBlock.Header.Hash()

	if !reflect.DeepEqual(block1, retBlock) {
		t.Fatalf("Fail: got %+v\nexpected %+v", retBlock, block1)
	}

	// Check if latestBlock is set correctly
	if !reflect.DeepEqual(*block1.Header, blockState.latestBlock) {
		t.Fatalf("LatestBlock Fail: got %+v\nexpected %+v", blockState.latestBlock, block1)
	}

}