package blocktree

import (
	"io/ioutil"
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"

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
			Digest:     types.NewDigest(),
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
			digest := types.NewDigest()
			err := digest.Add(types.ConsensusDigest{
				ConsensusEngineID: types.BabeEngineID,
				Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
			})
			if err != nil {
				return nil, nil
			}
			header := &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)),
				Digest:     digest,
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

	db, err := utils.SetupDatabase(testDatadirPath, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	return db
}
