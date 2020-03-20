// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
)

// helper method to create and start test state service
func newTestService(t *testing.T) (state *Service) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	state = NewService(testDir)

	return state
}

func newTestMemDBService() *Service {
	state := NewService("")
	state.UseMemDB()
	return state
}

func TestService_Start(t *testing.T) {
	state := newTestService(t)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), trie.EmptyHash, trie.EmptyHash, [][]byte{})
	if err != nil {
		t.Fatal(err)
	}

	tr := trie.NewEmptyTrie()

	err = state.Initialize(genesisHeader, tr)
	if err != nil {
		t.Fatal(err)
	}

	err = state.Start()
	if err != nil {
		t.Fatal(err)
	}

	state.Stop()
}

func TestMemDB_Start(t *testing.T) {
	state := newTestMemDBService()

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), trie.EmptyHash, trie.EmptyHash, [][]byte{})
	if err != nil {
		t.Fatal(err)
	}

	tr := trie.NewEmptyTrie()

	err = state.Initialize(genesisHeader, tr)
	if err != nil {
		t.Fatal(err)
	}

	err = state.Start()
	if err != nil {
		t.Fatal(err)
	}

	state.Stop()
}

func addBlocksToState(blockState *BlockState, depth int) {
	previousHash := blockState.BestBlockHash()

	// branch tree randomly
	type testBranch struct {
		hash  common.Hash
		depth int
	}

	branches := []testBranch{}
	r := *rand.New(rand.NewSource(rand.Int63()))

	// create base tree
	for i := 1; i <= depth; i++ {
		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)),
				StateRoot:  trie.EmptyHash,
			},
			Body: &types.Body{},
		}

		hash := block.Header.Hash()
		blockState.AddBlock(block)
		previousHash = hash

		isBranch := r.Intn(2)
		if isBranch == 1 {
			branches = append(branches, testBranch{
				hash:  hash,
				depth: i,
			})
		}
	}

	// create tree branches
	for _, branch := range branches {
		for i := branch.depth; i <= depth; i++ {
			block := &types.Block{
				Header: &types.Header{
					ParentHash: previousHash,
					Number:     big.NewInt(int64(i)),
					StateRoot:  trie.EmptyHash,
				},
				Body: &types.Body{},
			}

			hash := block.Header.Hash()
			blockState.AddBlock(block)
			previousHash = hash
		}
	}
}

func TestService_BlockTree(t *testing.T) {
	testDir := utils.NewTestDir(t)

	// removes all data directories created within test directory
	defer utils.RemoveTestDir(t)

	state := NewService(testDir)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), trie.EmptyHash, trie.EmptyHash, [][]byte{})
	if err != nil {
		t.Fatal(err)
	}

	tr := trie.NewEmptyTrie()
	err = state.Initialize(genesisHeader, tr)
	if err != nil {
		t.Fatal(err)
	}

	err = state.Start()
	if err != nil {
		t.Fatal(err)
	}

	// add blocks to state
	addBlocksToState(state.Block, 10)

	state.Stop()

	state2 := NewService(testDir)

	err = state2.Start()
	if err != nil {
		t.Fatal(err)
	}

	state2.Stop()

	if !reflect.DeepEqual(state.Block.BestBlockHash(), state2.Block.BestBlockHash()) {
		t.Fatalf("Fail: got %s expected %s", state.Block.BestBlockHash(), state2.Block.BestBlockHash())
	}
}

// func TestTrie_StoreAndLoadFromDB(t *testing.T) {
// 	trie := trie.NewEmptyTrie()

// 	rt := generateRandomTests(1000)
// 	var val []byte
// 	for _, test := range rt {
// 		err = trie.Put(test.key, test.value)
// 		if err != nil {
// 			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
// 		}

// 		val, err = trie.Get(test.key)
// 		if err != nil {
// 			t.Errorf("Fail to get key %x: %s", test.key, err.Error())
// 		} else if !bytes.Equal(val, test.value) {
// 			t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
// 		}
// 	}

// 	err := trie.StoreInDB()
// 	if err != nil {
// 		t.Fatalf("Fail: could not write trie to DB: %s", err)
// 	}

// 	encroot, err := trie.Hash()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	expected := &Trie{root: trie.root}

// 	trie.root = nil
// 	err = trie.LoadFromDB(encroot)
// 	if err != nil {
// 		t.Errorf("Fail: could not load trie from DB: %s", err)
// 	}

// 	if strings.Compare(expected.String(), trie.String()) != 0 {
// 		t.Errorf("Fail: got\n %s expected\n %s", expected.String(), trie.String())
// 	}

// 	if !reflect.DeepEqual(expected.root, trie.root) {
// 		t.Errorf("Fail: got\n %s expected\n %s", expected.String(), trie.String())
// 	}
// }

func TestStoreAndLoadHash(t *testing.T) {
	trie, err := trie.NewEmptyTrie()
	if err != nil {
		t.Fatal(err)
	}

	defer trie.closeDb()

	tests := []trie.Test{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0xf2, 0x3}, value: []byte("f")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
		{key: []byte{0x07}, value: []byte("ramen")},
		{key: []byte{0}, value: nil},
	}

	for _, test := range tests {
		err = trie.Put(test.key, test.value)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = trie.StoreHash()
	if err != nil {
		t.Fatal(err)
	}

	hash, err := trie.LoadHash()
	if err != nil {
		t.Fatal(err)
	}

	expected, err := trie.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if hash != expected {
		t.Fatalf("Fail: got %x expected %x", hash, expected)
	}
}

// func TestStoreAndLoadGenesisData(t *testing.T) {
// 	trie, err := newTrie()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	defer trie.closeDb()

// 	bootnodes := common.StringArrayToBytes([]string{
// 		"/ip4/127.0.0.1/tcp/7001/p2p/12D3KooWHHzSeKaY8xuZVzkLbKFfvNgPPeKhFBGrMbNzbm5akpqu",
// 		"/ip4/127.0.0.1/tcp/7001/p2p/12D3KooWHHzSeKaY8xuZVzkLbKFfvNgPPeKhFBGrMbNzbm5akpqu",
// 	})

// 	expected := &genesis.Data{
// 		Name:       "gossamer",
// 		ID:         "gossamer",
// 		Bootnodes:  bootnodes,
// 		ProtocolID: "/gossamer/test/0",
// 	}

// 	err = trie.db.StoreGenesisData(expected)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	gen, err := trie.db.LoadGenesisData()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if !reflect.DeepEqual(gen, expected) {
// 		t.Fatalf("Fail: got %v expected %v", gen, expected)
// 	}
// }
