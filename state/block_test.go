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

func TestGetBlockByNumber(t *testing.T) {
	dataDir := "../test_data/block"

	//Create a new blockState
	blockState, err := NewBlockState(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	//Close DB & erase data dir contents
	defer func() {
		err = blockState.Db.Db.Close()
		if err != nil {
			t.Fatal("BlockDB close err: ", err)
		}
		if err = os.RemoveAll(dataDir); err != nil {
			fmt.Println("removal of temp directory test_data failed")
		}
	}()

	// Create a header & blockdata
	blockHash := common.NewHash([]byte{0, 1, 2})
	header := types.BlockHeaderWithHash{
		Number: big.NewInt(1),
		Hash:   blockHash,
	}

	// BlockBody with fake extrinsics
	blockBody := types.BlockBody{}
	blockBody = append(blockBody, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}...)

	blockData := types.BlockData{
		Hash:   blockHash,
		Header: &header,
		Body:   &blockBody,
	}

	// Set the block's header & blockData in the blockState
	err = blockState.SetHeader(header)
	if err != nil {
		t.Fatal(err)
	}

	err = blockState.SetBlockData(blockHash, blockData)
	if err != nil {
		t.Fatal(err)
	}

	// Get block & check if it's the same as the expectedBlock
	expectedBlock := types.Block{
		Header: header,
		Body:   blockBody,
	}
	retBlock, err := blockState.GetBlockByNumber(big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedBlock, retBlock) {
		t.Fatalf("Fail: got %+v\nexpected %+v", retBlock, expectedBlock)
	}

}
