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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/utils"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

// NewInMemoryDB creates a new in-memory database
func NewInMemoryDB(t *testing.T) chaindb.Database {
	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)

	db, err := chaindb.NewBadgerDB(&chaindb.Config{
		DataDir:  testDatadirPath,
		InMemory: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

// GetRuntimeVars returns the testRuntimeFilePath and testRuntimeURL
func GetRuntimeVars(targetRuntime string) (string, string) {
	var testRuntimeFilePath string
	var testRuntimeURL string

	switch targetRuntime {
	case NODE_RUNTIME:
		testRuntimeFilePath, testRuntimeURL = GetAbsolutePath(NODE_RUNTIME_FP), NODE_RUNTIME_URL
	case POLKADOT_RUNTIME:
		testRuntimeFilePath, testRuntimeURL = GetAbsolutePath(POLKADOT_RUNTIME_FP), POLKADOT_RUNTIME_URL
	case HOST_API_TEST_RUNTIME:
		testRuntimeFilePath, testRuntimeURL = GetAbsolutePath(HOST_API_TEST_RUNTIME_FP), HOST_API_TEST_RUNTIME_URL
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

// TestRuntimeNetwork ...
type TestRuntimeNetwork struct{}

// NetworkState ...
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

func generateEd25519Signatures(t *testing.T, n int) []*Signature {
	t.Helper()
	signs := make([]*Signature, n)
	for i := 0; i < n; i++ {
		msg := []byte("Hello")
		key, err := ed25519.GenerateKeypair()
		require.NoError(t, err)

		sign, err := key.Private().Sign(msg)
		require.NoError(t, err)

		signs[i] = &Signature{
			PubKey:    key.Public().Encode(),
			Sign:      sign,
			Msg:       msg,
			KeyTypeID: crypto.Ed25519Type,
		}
	}
	return signs
}

// GenerateRuntimeWasmFile generates all runtime wasm files.
func GenerateRuntimeWasmFile() ([]string, error) {
	var wasmFilePaths []string
	runtimes := []string{HOST_API_TEST_RUNTIME, POLKADOT_RUNTIME, NODE_RUNTIME}
	for _, rt := range runtimes {
		testRuntimeFilePath, testRuntimeURL := GetRuntimeVars(rt)
		wasmFilePaths = append(wasmFilePaths, testRuntimeFilePath)
		_, err := GetRuntimeBlob(testRuntimeFilePath, testRuntimeURL)
		if err != nil {
			return nil, err
		}
	}
	return wasmFilePaths, nil
}

// RemoveFiles removes multiple files.
func RemoveFiles(files []string) error {
	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			return err
		}
	}
	return nil
}
