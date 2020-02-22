package blocktree

import (
	"io/ioutil"
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/db"
)

func createTestBlockTree(t *testing.T, genesisBlock *types.Block, depth int, db db.Database) *BlockTree {
	bt := NewBlockTreeFromGenesis(genesisBlock, db)
	previousHash := genesisBlock.Header.Hash()

	// branch tree randomly
	type testBranch struct {
		hash  Hash
		depth *big.Int
	}

	branches := []testBranch{}
	r := *rand.New(rand.NewSource(rand.Int63()))

	// create base tree
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

		isBranch := r.Intn(2)
		if isBranch == 1 {
			branches = append(branches, testBranch{
				hash:  hash,
				depth: bt.GetNode(hash).depth,
			})
		}
	}

	// create tree branches
	for _, branch := range branches {
		for i := int(branch.depth.Uint64()); i <= depth; i++ {
			block := &types.Block{
				Header: &types.Header{
					ParentHash: branch.hash,
					Number:     big.NewInt(int64(i)),
				},
				Body: &types.Body{},
			}

			hash := block.Header.Hash()
			bt.AddBlock(block)
			previousHash = hash
		}
	}

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
