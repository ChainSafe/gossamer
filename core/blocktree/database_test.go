package blocktree

import (
	"io/ioutil"
	"math/big"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/db"
)

func createTestBlockTree(t *testing.T, genesisBlock *types.Block, depth int, db db.Database) *BlockTree {
	bt := NewBlockTreeFromGenesis(genesisBlock, db)
	previousHash := genesisBlock.Header.Hash()

	for i := 1; i <= depth; i++ {
		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)),
			},
			Body: &types.Body{},
		}

		hash := block.Header.Hash()
		bt.AddBlock(block)
		previousHash = hash
	}

	// node := getNodeFromBlockNumber(depth/3)
	// for i := depth/3; i <= depth; i++ {

	// }

	return bt
}

func TestStoreBlockTree(t *testing.T) {
	dataDir, err := ioutil.TempDir("", "./test_data")
	if err != nil {
		t.Fatal(err)
	}

	testDb, err := db.NewBadgerDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	zeroHash, err := common.HexToHash("0x00")
	if err != nil {
		t.Fatal(err)
	}

	genesisBlock := &types.Block{
		Header: &types.Header{
			ParentHash: zeroHash,
			Number:     big.NewInt(0),
		},
		Body: &types.Body{},
	}

	bt := createTestBlockTree(t, genesisBlock, 10, testDb)

	err = bt.Store()
	if err != nil {
		t.Fatal(err)
	}

	resBt := NewBlockTreeFromGenesis(genesisBlock, testDb)
	err = resBt.Load()

	if !reflect.DeepEqual(bt, resBt) {
		t.Fatalf("Fail: got %v expected %v", resBt, bt)
	}
}
