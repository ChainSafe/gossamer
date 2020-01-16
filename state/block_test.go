package state

import (
	"math/big"
	"os"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/core/types"
)

func TestGetBlockByNumber(t *testing.T) {
	dataDir := "../test_data"

	// Create & start a new State service
	stateService := NewService(dataDir)
	stateService.Start()

	// Close the service, and remove dataDir once test is done
	defer stateService.Stop()
	defer func() {
		if err := os.RemoveAll(dataDir); err != nil {
			t.Fatalf("removal of temp directory failed")
		}
	}()

	// Create a header & blockdata
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
	// SetHeader also sets mapping [blockNubmer : hash] in DB
	err := stateService.Block.SetHeader(*blockHeader)
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
