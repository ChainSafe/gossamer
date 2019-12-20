package state

import (
	"fmt"
	"math/big"
	"os"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
)

func TestAddBlock(t *testing.T) {
	dataDir := "../test_data/block"

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
		if err = os.RemoveAll(dataDir); err != nil {
			fmt.Println("removal of temp directory test_data failed")
		}
	}()

	// Create block0 & call AddBlock
	// Create a header & blockdata
	blockHash0 := common.NewHash([]byte{0, 1, 2})
	header0 := types.BlockHeaderWithHash{
		Number: big.NewInt(0),
		Hash:   blockHash0,
	}

	// BlockBody with fake extrinsics
	blockBody0 := types.BlockBody{}
	blockBody0 = append(blockBody0, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}...)

	block0 := types.Block{
		Header: header0,
		Body:   blockBody0,
	}

	// Add the block0 to the DB
	blockState.AddBlock(block0)

	// Create block1 & call AddBlock
	// Create a header & blockdata
	blockHash1 := common.NewHash([]byte{1, 2, 3})
	header1 := types.BlockHeaderWithHash{
		Number: big.NewInt(1),
		Hash:   blockHash1,
	}

	// BlockBody with fake extrinsics
	blockBody1 := types.BlockBody{}
	blockBody1 = append(blockBody1, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}...)

	block1 := types.Block{
		Header: header1,
		Body:   blockBody1,
	}

	// Add the block1 to the DB
	blockState.AddBlock(block1)

	// Get the blocks & check if it's the same as the added blocks
	retBlock, err := blockState.GetBlockByHash(blockHash0)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(block0, retBlock) {
		t.Fatalf("Fail: got %+v\nexpected %+v", retBlock, block0)
	}

	retBlock, err = blockState.GetBlockByHash(blockHash1)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(block1, retBlock) {
		t.Fatalf("Fail: got %+v\nexpected %+v", retBlock, block1)
	}

	// Check if latestBlock is set correctly
	if !reflect.DeepEqual(block1, blockState.latestBlock) {
		t.Fatalf("LatestBlock Fail: got %+v\nexpected %+v", blockState.latestBlock, block1)
	}

}
