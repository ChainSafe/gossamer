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
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

// TestAuthorityDataKey is the location of authority data in the storage trie
var TestAuthorityDataKey, _ = common.HexToBytes("0x3a6772616e6470615f617574686f726974696573")

// NewTestRuntime will create a new runtime (polkadot/test)
func NewTestRuntime(t *testing.T, targetRuntime string) *Runtime {
	return NewTestRuntimeWithTrie(t, targetRuntime, nil)
}

// NewTestRuntimeWithTrie will create a new runtime (polkadot/test) with the supplied trie as the storage
func NewTestRuntimeWithTrie(t *testing.T, targetRuntime string, tt *trie.Trie) *Runtime {
	testRuntimeFilePath, testRuntimeURL, importsFunc := GetRuntimeVars(targetRuntime)

	_, err := GetRuntimeBlob(testRuntimeFilePath, testRuntimeURL)
	require.Nil(t, err, "Fail: could not get runtime", "targetRuntime", targetRuntime)

	rs := NewTestRuntimeStorage(tt)

	fp, err := filepath.Abs(testRuntimeFilePath)
	require.Nil(t, err, "could not create testRuntimeFilePath", "targetRuntime", targetRuntime)

	cfg := &Config{
		Storage:  rs,
		Keystore: keystore.NewKeystore(),
		Imports:  importsFunc,
		LogLvl:   log.LvlInfo,
	}

	r, err := NewRuntimeFromFile(fp, cfg)
	require.Nil(t, err, "Got error when trying to create new VM", "targetRuntime", targetRuntime)
	require.NotNil(t, r, "Could not create new VM instance", "targetRuntime", targetRuntime)

	return r
}

// exportRuntime writes the runtime to a file as a hex string.
func exportRuntime(t *testing.T, targetRuntime string, outFp string) {
	testRuntimeFilePath, testRuntimeURL, _ := GetRuntimeVars(targetRuntime)

	_, err := GetRuntimeBlob(testRuntimeFilePath, testRuntimeURL)
	require.Nil(t, err, "Fail: could not get runtime", "targetRuntime", targetRuntime)

	fp, err := filepath.Abs(testRuntimeFilePath)
	require.NoError(t, err, "could not create testRuntimeFilePath", "targetRuntime", targetRuntime)

	bytes, err := wasm.ReadBytes(fp)
	require.NoError(t, err)

	str := fmt.Sprintf("0x%x", bytes)

	out, err := filepath.Abs(outFp)
	require.NoError(t, err)

	file, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)

	_, err = file.WriteString(str)
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)
}

//nolint
const (
	SUBSTRATE_TEST_RUNTIME     = "substrate_test_runtime"
	SUBSTRATE_TEST_RUNTIME_FP  = "substrate_test_runtime.compact.wasm"
	SUBSTRATE_TEST_RUNTIME_URL = "https://github.com/noot/substrate/blob/add-blob-042920/target/wasm32-unknown-unknown/release/wbuild/substrate-test-runtime/substrate_test_runtime.compact.wasm?raw=true"

	NODE_RUNTIME     = "node_runtime"
	NODE_RUNTIME_FP  = "node_runtime.compact.wasm"
	NODE_RUNTIME_URL = "https://github.com/noot/substrate/blob/noot/legacy/target/wasm32-unknown-unknown/release/wbuild/node-runtime/node_runtime.compact.wasm?raw=true"

	TEST_RUNTIME  = "test_runtime"
	TESTS_FP      = "test_wasm.wasm"
	TEST_WASM_URL = "https://github.com/ChainSafe/gossamer-test-wasm/blob/noot/target/wasm32-unknown-unknown/release/test_wasm.wasm?raw=true"

	SIMPLE_WASM_FP     = "simple.wasm"
	SIMPLE_RUNTIME_URL = "https://github.com//wasmerio/go-ext-wasm/blob/master/wasmer/test/testdata/examples/simple.wasm?raw=true"
)

// GetAbsolutePath returns the completePath for a given targetDir
func GetAbsolutePath(targetDir string) string {
	dir, err := os.Getwd()
	if err != nil {
		panic("failed to get current working directory")
	}
	return path.Join(dir, targetDir)
}

// GetRuntimeVars returns the testRuntimeFilePath and testRuntimeURL
func GetRuntimeVars(targetRuntime string) (string, string, func() (*wasm.Imports, error)) {
	var testRuntimeFilePath string
	var testRuntimeURL string
	var registerImports func() (*wasm.Imports, error)

	switch targetRuntime {
	case SUBSTRATE_TEST_RUNTIME:
		registerImports = RegisterImports_TestRuntime
		testRuntimeFilePath, testRuntimeURL = GetAbsolutePath(SUBSTRATE_TEST_RUNTIME_FP), SUBSTRATE_TEST_RUNTIME_URL
	case NODE_RUNTIME:
		registerImports = RegisterImports_NodeRuntime
		testRuntimeFilePath, testRuntimeURL = GetAbsolutePath(NODE_RUNTIME_FP), NODE_RUNTIME_URL
	case TEST_RUNTIME:
		registerImports = RegisterImports_NodeRuntime
		testRuntimeFilePath, testRuntimeURL = GetAbsolutePath(TESTS_FP), TEST_WASM_URL
	default:
		registerImports = RegisterImports_NodeRuntime
	}

	return testRuntimeFilePath, testRuntimeURL, registerImports
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

// TestRuntimeStorage holds trie pointer
type TestRuntimeStorage struct {
	trie *trie.Trie
}

// NewTestRuntimeStorage creates new instance of TestRuntimeStorage
func NewTestRuntimeStorage(tr *trie.Trie) *TestRuntimeStorage {
	if tr == nil {
		tr = trie.NewEmptyTrie()
	}
	return &TestRuntimeStorage{
		trie: tr,
	}
}

// TrieAsString is a dummy test func
func (trs TestRuntimeStorage) TrieAsString() string {
	return trs.trie.String()
}

// SetStorage is a dummy test func
func (trs TestRuntimeStorage) SetStorage(key []byte, value []byte) error {
	return trs.trie.Put(key, value)
}

// GetStorage is a dummy test func
func (trs TestRuntimeStorage) GetStorage(key []byte) ([]byte, error) {
	return trs.trie.Get(key)
}

// StorageRoot is a dummy test func
func (trs TestRuntimeStorage) StorageRoot() (common.Hash, error) {
	return trs.trie.Hash()
}

// SetStorageChild is a dummy test func
func (trs TestRuntimeStorage) SetStorageChild(keyToChild []byte, child *trie.Trie) error {
	return trs.trie.PutChild(keyToChild, child)
}

// SetStorageIntoChild is a dummy test func
func (trs TestRuntimeStorage) SetStorageIntoChild(keyToChild, key, value []byte) error {
	return trs.trie.PutIntoChild(keyToChild, key, value)
}

// GetStorageFromChild is a dummy test func
func (trs TestRuntimeStorage) GetStorageFromChild(keyToChild, key []byte) ([]byte, error) {
	return trs.trie.GetFromChild(keyToChild, key)
}

// ClearStorage is a dummy test func
func (trs TestRuntimeStorage) ClearStorage(key []byte) error {
	return trs.trie.Delete(key)
}

// Entries is a dummy test func
func (trs TestRuntimeStorage) Entries() map[string][]byte {
	return trs.trie.Entries()
}

// SetBalance sets the balance for an account with the given public key
func (trs TestRuntimeStorage) SetBalance(key [32]byte, balance uint64) error {
	skey, err := common.BalanceKey(key)
	if err != nil {
		return err
	}

	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, balance)

	return trs.SetStorage(skey, bb)
}

// GetBalance gets the balance for an account with the given public key
func (trs TestRuntimeStorage) GetBalance(key [32]byte) (uint64, error) {
	skey, err := common.BalanceKey(key)
	if err != nil {
		return 0, err
	}

	bal, err := trs.GetStorage(skey)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(bal), nil
}
