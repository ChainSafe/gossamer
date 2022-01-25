// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"context"
	"io"
	"math/big"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v3/types"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

// NewInMemoryDB creates a new in-memory database
func NewInMemoryDB(t *testing.T) chaindb.Database {
	testDatadirPath := t.TempDir()

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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	return os.WriteFile(testRuntimeFilePath, respBody, os.ModePerm)
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

func generateEd25519Signatures(t *testing.T, n int) []*crypto.SignatureInfo {
	t.Helper()
	signs := make([]*crypto.SignatureInfo, n)
	for i := 0; i < n; i++ {
		msg := []byte("Hello")
		key, err := ed25519.GenerateKeypair()
		require.NoError(t, err)

		sign, err := key.Private().Sign(msg)
		require.NoError(t, err)

		signs[i] = &crypto.SignatureInfo{
			PubKey:     key.Public().Encode(),
			Sign:       sign,
			Msg:        msg,
			VerifyFunc: ed25519.VerifySignature,
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

// NewTestExtrinsic builds a new extrinsic using centrifuge pkg
func NewTestExtrinsic(t *testing.T, rt Instance, genHash, blockHash common.Hash, nonce uint64, call string, args ...interface{}) string { //nolint
	t.Helper()

	rawMeta, err := rt.Metadata()
	require.NoError(t, err)

	var decoded []byte
	err = scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = ctypes.DecodeFromBytes(decoded, meta)
	require.NoError(t, err)

	rv, err := rt.Version()
	require.NoError(t, err)

	c, err := ctypes.NewCall(meta, call, args...)
	require.NoError(t, err)

	ext := ctypes.NewExtrinsic(c)
	o := ctypes.SignatureOptions{
		BlockHash:          ctypes.Hash(blockHash),
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: false},
		GenesisHash:        ctypes.Hash(genHash),
		Nonce:              ctypes.NewUCompactFromUInt(nonce),
		SpecVersion:        ctypes.U32(rv.SpecVersion()),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(rv.TransactionVersion()),
	}

	// Sign the transaction using Alice's key
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	extEnc, err := ctypes.EncodeToHexString(ext)
	require.NoError(t, err)

	return extEnc
}

// InitializeRuntimeToTest sets a new block using the runtime functions to set initial data into the host
func InitializeRuntimeToTest(t *testing.T, instance Instance, parentHash common.Hash) *types.Block {
	t.Helper()

	header := &types.Header{
		ParentHash: parentHash,
		Number:     big.NewInt(1),
		Digest:     types.NewDigest(),
	}

	err := instance.InitializeBlock(header)
	require.NoError(t, err)

	idata := types.NewInherentsData()
	err = idata.SetInt64Inherent(types.Timstap0, 1)
	require.NoError(t, err)

	err = idata.SetInt64Inherent(types.Babeslot, 1)
	require.NoError(t, err)

	ienc, err := idata.Encode()
	require.NoError(t, err)

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as extrinsics
	inherentExts, err := instance.InherentExtrinsics(ienc)
	require.NoError(t, err)

	// decode inherent extrinsics
	var exts [][]byte
	err = scale.Unmarshal(inherentExts, &exts)
	require.NoError(t, err)

	// apply each inherent extrinsic
	for _, ext := range exts {
		in, err := scale.Marshal(ext)
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(append([]byte{1}, in...))
		require.NoError(t, err, in)
		require.Equal(t, ret, []byte{0, 0})
	}

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)

	res.Number = header.Number

	babeDigest := types.NewBabeDigest()
	err = babeDigest.Set(*types.NewBabePrimaryPreDigest(0, 1, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	data, err := scale.Marshal(babeDigest)
	require.NoError(t, err)
	preDigest := types.NewBABEPreRuntimeDigest(data)

	digest := types.NewDigest()
	err = digest.Add(preDigest)
	require.NoError(t, err)
	res.Digest = digest

	expected := &types.Header{
		ParentHash: header.ParentHash,
		Number:     big.NewInt(1),
		Digest:     digest,
	}

	require.Equal(t, expected.ParentHash, res.ParentHash)
	require.Equal(t, expected.Number, res.Number)
	require.Equal(t, expected.Digest, res.Digest)
	require.False(t, res.StateRoot.IsEmpty())
	require.False(t, res.ExtrinsicsRoot.IsEmpty())
	require.NotEqual(t, trie.EmptyHash, res.StateRoot)

	return &types.Block{
		Header: *res,
		Body:   *types.NewBody(types.BytesArrayToExtrinsics(exts)),
	}
}
