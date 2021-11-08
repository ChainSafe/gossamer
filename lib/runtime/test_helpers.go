// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

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
	switch targetRuntime {
	case NODE_RUNTIME:
		return GetAbsolutePath(NODE_RUNTIME_FP), NODE_RUNTIME_URL
	case NODE_RUNTIME_v098:
		return GetAbsolutePath(NODE_RUNTIME_FP_v098), NODE_RUNTIME_URL_v098
	case POLKADOT_RUNTIME_v0910:
		return GetAbsolutePath(POLKADOT_RUNTIME_FP_v0910), POLKADOT_RUNTIME_URL_v0910
	case POLKADOT_RUNTIME:
		return GetAbsolutePath(POLKADOT_RUNTIME_FP), POLKADOT_RUNTIME_URL
	case HOST_API_TEST_RUNTIME:
		return GetAbsolutePath(HOST_API_TEST_RUNTIME_FP), HOST_API_TEST_RUNTIME_URL
	case DEV_RUNTIME:
		return GetAbsolutePath(DEV_RUNTIME_FP), DEV_RUNTIME_URL
	default:
		return "", ""
	}
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
func GetRuntimeBlob(testRuntimeFilePath, testRuntimeURL string) error {
	if utils.PathExists(testRuntimeFilePath) {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testRuntimeURL, nil)
	if err != nil {
		return err
	}

	const runtimeReqTimout = time.Second * 30

	httpcli := http.Client{Timeout: runtimeReqTimout}
	resp, err := httpcli.Do(req)
	if err != nil {
		return err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	return ioutil.WriteFile(testRuntimeFilePath, respBody, os.ModePerm)
}

// TestRuntimeNetwork ...
type TestRuntimeNetwork struct{}

// NetworkState ...
func (*TestRuntimeNetwork) NetworkState() common.NetworkState {
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
	for _, rt := range runtimes {
		testRuntimeFilePath, testRuntimeURL := GetRuntimeVars(rt)
		err := GetRuntimeBlob(testRuntimeFilePath, testRuntimeURL)
		if err != nil {
			return nil, err
		}

		wasmFilePaths = append(wasmFilePaths, testRuntimeFilePath)
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
