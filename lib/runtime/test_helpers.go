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

package runtime

import (
	"encoding/binary"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	ma "github.com/multiformats/go-multiaddr"
)

// TestAuthorityDataKey is the location of GRANDPA authority data in the storage trie for NODE_RUNTIME
var TestAuthorityDataKey, _ = common.HexToBytes("0x3a6772616e6470615f617574686f726974696573")

// GetRuntimeVars returns the testRuntimeFilePath and testRuntimeURL
func GetRuntimeVars(targetRuntime string) (string, string) {
	var testRuntimeFilePath string
	var testRuntimeURL string

	switch targetRuntime {
	case SUBSTRATE_TEST_RUNTIME:
		testRuntimeFilePath, testRuntimeURL = GetAbsolutePath(SUBSTRATE_TEST_RUNTIME_FP), SUBSTRATE_TEST_RUNTIME_URL
	case NODE_RUNTIME:
		testRuntimeFilePath, testRuntimeURL = GetAbsolutePath(NODE_RUNTIME_FP), NODE_RUNTIME_URL
	case TEST_RUNTIME:
		testRuntimeFilePath, testRuntimeURL = GetAbsolutePath(TESTS_FP), TEST_WASM_URL
	}

	return testRuntimeFilePath, testRuntimeURL
}

// GetAbsolutePath returns the completePath for a given targetDir
func GetAbsolutePath(targetDir string) string {
	dir, err := os.Getwd()
	if err != nil {
		panic("failed to get current working directory")
	}
	return path.Join(dir, targetDir)
}

// GetRuntimeBlob checks if the test wasm @testRuntimeFilePath exists and if not, it fetches it from @testRuntimeURL
func GetRuntimeBlob(testRuntimeFilePath, testRuntimeURL string) (n int64, err error) {
	if utils.PathExists(testRuntimeFilePath) {
		return 0, nil
	}

	out, err := os.Create(testRuntimeFilePath)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = out.Close()
	}()

	/* #nosec */
	resp, err := http.Get(testRuntimeURL)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	n, err = io.Copy(out, resp.Body)
	return n, err
}

type TestRuntimeStorage struct {
	trie *trie.Trie
}

func NewTestRuntimeStorage(tr *trie.Trie) *TestRuntimeStorage {
	if tr == nil {
		tr = trie.NewEmptyTrie()
	}
	return &TestRuntimeStorage{
		trie: tr,
	}
}

func (trs *TestRuntimeStorage) Trie() *trie.Trie {
	return trs.trie
}

func (trs *TestRuntimeStorage) Set(key []byte, value []byte) error {
	return trs.trie.Put(key, value)
}

func (trs *TestRuntimeStorage) Get(key []byte) ([]byte, error) {
	return trs.trie.Get(key)
}

func (trs *TestRuntimeStorage) Root() (common.Hash, error) {
	return trs.trie.Hash()
}

func (trs *TestRuntimeStorage) SetChild(keyToChild []byte, child *trie.Trie) error {
	return trs.trie.PutChild(keyToChild, child)
}

func (trs *TestRuntimeStorage) SetChildStorage(keyToChild, key, value []byte) error {
	return trs.trie.PutIntoChild(keyToChild, key, value)
}

func (trs *TestRuntimeStorage) GetChildStorage(keyToChild, key []byte) ([]byte, error) {
	return trs.trie.GetFromChild(keyToChild, key)
}

func (trs *TestRuntimeStorage) Delete(key []byte) error {
	return trs.trie.Delete(key)
}

func (trs *TestRuntimeStorage) Entries() map[string][]byte {
	return trs.trie.Entries()
}

func (trs *TestRuntimeStorage) SetBalance(key [32]byte, balance uint64) error {
	skey, err := common.BalanceKey(key)
	if err != nil {
		return err
	}

	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, balance)

	return trs.Set(skey, bb)
}

func (trs *TestRuntimeStorage) GetBalance(key [32]byte) (uint64, error) {
	skey, err := common.BalanceKey(key)
	if err != nil {
		return 0, err
	}

	bal, err := trs.Get(skey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(bal), nil
}

func (trs *TestRuntimeStorage) DeleteChildStorage(key []byte) error {
	return trs.trie.DeleteFromChild(key)
}

func (trs *TestRuntimeStorage) ClearChildStorage(keyToChild, key []byte) error {
	return trs.trie.ClearFromChild(keyToChild, key)
}

func (trs *TestRuntimeStorage) KeepAlive() {
	go func() {
		for {
			trs.trie = trs.trie
		}
	}()
}

type TestRuntimeNetwork struct {
}

func (trn *TestRuntimeNetwork) NetworkState() common.NetworkState {
	testAddrs := []ma.Multiaddr(nil)

	// create mock multiaddress
	addr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/7001/p2p/12D3KooWDcCNBqAemRvguPa7rtmsbn2hpgLqAz8KsMMFsF2rdCUP")

	testAddrs = append(testAddrs, addr)

	return common.NetworkState{
		PeerID:     "12D3KooWDcCNBqAemRvguPa7rtmsbn2hpgLqAz8KsMMFsF2rdCUP",
		Multiaddrs: testAddrs,
	}
}
