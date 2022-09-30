// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
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
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
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

var (
	ErrRuntimeUnknown  = errors.New("runtime is not known")
	ErrHTTPStatusNotOK = errors.New("HTTP status code received is not OK")
	ErrOpenRuntimeFile = errors.New("cannot open the runtime target file")
)

// GetRuntime returns the runtime file path located in the
// /tmp/gossamer/runtimes directory (depending on OS and environment).
// If the file did not exist, the runtime WASM blob is downloaded to that file.
func GetRuntime(ctx context.Context, runtime string) (
	runtimePath string, err error) {
	basePath := filepath.Join(os.TempDir(), "/gossamer/runtimes/")
	const perm = os.FileMode(0777)
	err = os.MkdirAll(basePath, perm)
	if err != nil {
		return "", fmt.Errorf("cannot create directory for runtimes: %w", err)
	}

	var runtimeFilename, url string
	switch runtime {
	case NODE_RUNTIME:
		runtimeFilename = NODE_RUNTIME_FP
		url = NODE_RUNTIME_URL
	case NODE_RUNTIME_v098:
		runtimeFilename = NODE_RUNTIME_FP_v098
		url = NODE_RUNTIME_URL_v098
	case POLKADOT_RUNTIME_v0925:
		runtimeFilename = POLKADOT_RUNTIME_FP_v0925
		url = POLKADOT_RUNTIME_URL_v0925
	case POLKADOT_RUNTIME_v0917:
		runtimeFilename = POLKADOT_RUNTIME_FP_v0917
		url = POLKADOT_RUNTIME_URL_v0917
	case POLKADOT_RUNTIME_v0910:
		runtimeFilename = POLKADOT_RUNTIME_FP_v0910
		url = POLKADOT_RUNTIME_URL_v0910
	case POLKADOT_RUNTIME:
		runtimeFilename = POLKADOT_RUNTIME_FP
		url = POLKADOT_RUNTIME_URL
	case HOST_API_TEST_RUNTIME:
		runtimeFilename = HOST_API_TEST_RUNTIME_FP
		url = HOST_API_TEST_RUNTIME_URL
	case DEV_RUNTIME:
		runtimeFilename = DEV_RUNTIME_FP
		url = DEV_RUNTIME_URL
	default:
		return "", fmt.Errorf("%w: %s", ErrRuntimeUnknown, runtime)
	}

	runtimePath = filepath.Join(basePath, runtimeFilename)
	runtimePath, err = filepath.Abs(runtimePath)
	if err != nil {
		return "", fmt.Errorf("malformed relative path: %w", err)
	}

	if utils.PathExists(runtimePath) {
		return runtimePath, nil
	}

	const requestTimeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("cannot make HTTP request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("cannot get: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return "", fmt.Errorf("%w: %d %s", ErrHTTPStatusNotOK,
			response.StatusCode, response.Status)
	}

	const flag = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	file, err := os.OpenFile(runtimePath, flag, perm) //nolint:gosec
	if err != nil {
		_ = response.Body.Close()
		return "", fmt.Errorf("cannot open target destination file: %w", err)
	}

	_, err = io.Copy(file, response.Body)
	if err != nil {
		_ = response.Body.Close()
		return "", fmt.Errorf("cannot copy response body to %s: %w",
			runtimePath, err)
	}

	err = file.Close()
	if err != nil {
		return "", fmt.Errorf("cannot close file: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return "", fmt.Errorf("cannot close HTTP response body: %w", err)
	}

	return runtimePath, nil
}

// GetAbsolutePath returns the completePath for a given targetDir
func GetAbsolutePath(targetDir string) string {
	dir, err := os.Getwd()
	if err != nil {
		panic("failed to get current working directory")
	}
	return path.Join(dir, targetDir)
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

// NewTestExtrinsic builds a new extrinsic using centrifuge pkg
func NewTestExtrinsic(t *testing.T, rt Instance, genHash, blockHash common.Hash,
	nonce uint64, call string, args ...interface{}) string {
	t.Helper()

	rawMeta, err := rt.Metadata()
	require.NoError(t, err)

	var decoded []byte
	err = scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = codec.Decode(decoded, meta)
	require.NoError(t, err)

	rv := rt.Version()
	require.NoError(t, err)

	c, err := ctypes.NewCall(meta, call, args...)
	require.NoError(t, err)

	ext := ctypes.NewExtrinsic(c)
	o := ctypes.SignatureOptions{
		BlockHash:          ctypes.Hash(blockHash),
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: false},
		GenesisHash:        ctypes.Hash(genHash),
		Nonce:              ctypes.NewUCompactFromUInt(nonce),
		SpecVersion:        ctypes.U32(rv.SpecVersion),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(rv.TransactionVersion),
	}

	// Sign the transaction using Alice's key
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	extEnc, err := codec.EncodeToHex(ext)
	require.NoError(t, err)

	return extEnc
}

// InitializeRuntimeToTest sets a new block using the runtime functions to set initial data into the host
func InitializeRuntimeToTest(t *testing.T, instance Instance, parentHash common.Hash) *types.Block {
	t.Helper()

	header := &types.Header{
		ParentHash: parentHash,
		Number:     1,
		Digest:     types.NewDigest(),
	}

	err := instance.InitializeBlock(header)
	require.NoError(t, err)

	idata := types.NewInherentData()
	err = idata.SetInherent(types.Timstap0, uint64(1))
	require.NoError(t, err)

	err = idata.SetInherent(types.Babeslot, uint64(1))
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
	preDigest := *types.NewBABEPreRuntimeDigest(data)

	digest := types.NewDigest()
	err = digest.Add(preDigest)
	require.NoError(t, err)
	res.Digest = digest

	expected := &types.Header{
		ParentHash: header.ParentHash,
		Number:     1,
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
