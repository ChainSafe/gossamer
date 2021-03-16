package blocktree

import (
	"io/ioutil"
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/stretchr/testify/require"
)

type testBranch struct {
	hash  Hash
	depth *big.Int
}

func createTestBlockTree(header *types.Header, depth int, db chaindb.Database) (*BlockTree, []testBranch) {
	bt := NewBlockTreeFromRoot(header, db)
	previousHash := header.Hash()

	// branch tree randomly
	branches := []testBranch{}
	r := *rand.New(rand.NewSource(rand.Int63()))

	// create base tree
	for i := 1; i <= depth; i++ {
		header := &types.Header{
			ParentHash: previousHash,
			Number:     big.NewInt(int64(i)),
		}

		hash := header.Hash()
		bt.AddBlock(header, 0)
		previousHash = hash

		isBranch := r.Intn(2)
		if isBranch == 1 {
			branches = append(branches, testBranch{
				hash:  hash,
				depth: bt.getNode(hash).depth,
			})
		}
	}

	// create tree branches
	for _, branch := range branches {
		previousHash = branch.hash

		for i := int(branch.depth.Uint64()); i <= depth; i++ {
			header := &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)),
				Digest:     types.Digest{newMockDigestItem(rand.Intn(256))},
			}

			hash := header.Hash()
			bt.AddBlock(header, 0)
			previousHash = hash
		}
	}

	return bt, branches
}

func TestStoreBlockTree(t *testing.T) {
	db := newInMemoryDB(t)
	bt, _ := createTestBlockTree(testHeader, 10, db)

	err := bt.Store()
	require.NoError(t, err)

	resBt := NewBlockTreeFromRoot(testHeader, db)
	err = resBt.Load()
	require.NoError(t, err)

	if !reflect.DeepEqual(bt.head, resBt.head) {
		t.Fatalf("Fail: got %v expected %v", resBt, bt)
	}

	btLeafMap := bt.leaves.toMap()
	resLeafMap := bt.leaves.toMap()
	if !reflect.DeepEqual(btLeafMap, resLeafMap) {
		t.Fatalf("Fail: got %v expected %v", btLeafMap, resLeafMap)
	}
}
func newInMemoryDB(t *testing.T) chaindb.Database {
	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)

	db, err := chaindb.NewBadgerDB(&chaindb.Config{
		DataDir:  testDatadirPath,
		InMemory: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	return db
}
