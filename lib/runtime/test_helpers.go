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

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

// NewInMemoryDB creates a new in-memory database
func NewInMemoryDB(t *testing.T) database.Database {
	testDatadirPath := t.TempDir()

	db, err := database.NewPebble(testDatadirPath, true)
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
// If the runtime argument is not defined in the constants.go and is a valid
// file path, the runtime argument is returned.
func GetRuntime(ctx context.Context, runtime string) (
	runtimePath string, err error) {
	if utils.PathExists(runtime) {
		return runtime, nil
	}

	basePath := filepath.Join(os.TempDir(), "/gossamer/runtimes/")
	const perm = os.FileMode(0777)
	err = os.MkdirAll(basePath, perm)
	if err != nil {
		return "", fmt.Errorf("cannot create directory for runtimes: %w", err)
	}

	var runtimeFilename, url string
	switch runtime {
	case HOST_API_TEST_RUNTIME:
		runtimeFilename = HOST_API_TEST_RUNTIME_FP
		url = HOST_API_TEST_RUNTIME_URL
	case POLKADOT_RUNTIME_v0929:
		runtimeFilename = POLKADOT_RUNTIME_V0929_FP
		url = POLKADOT_RUNTIME_V0929_URL
	case WESTEND_RUNTIME_v0929:
		runtimeFilename = WESTEND_RUNTIME_V0929_FP
		url = WESTEND_RUNTIME_V0929_URL
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

// MetadataVersioner is an interface for getting metadata
// and version from a runtime.
type MetadataVersioner interface {
	Metadataer
	Versioner
}

// NewTestExtrinsic builds a new extrinsic using centrifuge pkg
func NewTestExtrinsic(t *testing.T, rt MetadataVersioner, genHash, blockHash common.Hash,
	nonce uint64, keyRingPair signature.KeyringPair, call string, args ...interface{}) string {
	t.Helper()

	rawMeta, err := rt.Metadata()
	require.NoError(t, err)

	var decoded []byte
	err = scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = codec.Decode(decoded, meta)
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
		SpecVersion:        ctypes.U32(rv.SpecVersion),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(rv.TransactionVersion),
	}

	// Sign the transaction using Alice's key
	err = ext.Sign(keyRingPair, o)
	require.NoError(t, err)

	extEnc, err := codec.EncodeToHex(ext)
	require.NoError(t, err)

	return extEnc
}

// Versioner returns the version from the runtime.
// This should return the cached version and be cheap to execute.
type Versioner interface {
	Version() (Version, error)
}

// Metadataer returns the metadata from the runtime.
type Metadataer interface {
	Metadata() (metadata []byte, err error)
}

// InitializeRuntimeToTest sets a new block using the runtime functions to set initial data into the host
func InitializeRuntimeToTest(t *testing.T, instance Instance, parentHeader *types.Header) *types.Block {
	t.Helper()

	babeConfig, err := instance.BabeConfiguration()
	require.NoError(t, err)

	slotDuration := babeConfig.SlotDuration
	timestamp := uint64(time.Now().UnixMilli())
	currentSlot := timestamp / slotDuration

	babeDigest := types.NewBabeDigest()
	err = babeDigest.Set(*types.NewBabePrimaryPreDigest(0, currentSlot, [32]byte{}, [64]byte{}))
	require.NoError(t, err)

	encodedBabeDigest, err := scale.Marshal(babeDigest)
	require.NoError(t, err)
	preDigest := *types.NewBABEPreRuntimeDigest(encodedBabeDigest)

	digest := types.NewDigest()
	require.NoError(t, err)
	err = digest.Add(preDigest)
	require.NoError(t, err)

	header := &types.Header{
		ParentHash: parentHeader.Hash(),
		Number:     parentHeader.Number + 1,
		Digest:     digest,
	}

	err = instance.InitializeBlock(header)
	require.NoError(t, err)

	inherentData := types.NewInherentData()
	err = inherentData.SetInherent(types.Timstap0, timestamp)
	require.NoError(t, err)

	err = inherentData.SetInherent(types.Babeslot, currentSlot)
	require.NoError(t, err)

	parachainInherent := inherents.ParachainInherentData{
		ParentHeader: *parentHeader,
	}

	err = inherentData.SetInherent(types.Parachn0, parachainInherent)
	require.NoError(t, err)

	err = inherentData.SetInherent(types.Newheads, []byte{0})
	require.NoError(t, err)

	encodedInnherents, err := inherentData.Encode()
	require.NoError(t, err)

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as extrinsics
	inherentExts, err := instance.InherentExtrinsics(encodedInnherents)
	require.NoError(t, err)

	var extrinsics [][]byte
	err = scale.Unmarshal(inherentExts, &extrinsics)
	require.NoError(t, err)

	for _, ext := range extrinsics {
		encodedExtrinsic, err := scale.Marshal(ext)
		require.NoError(t, err)

		wasmResult, err := instance.ApplyExtrinsic(encodedExtrinsic)
		require.NoError(t, err, encodedExtrinsic)
		require.Equal(t, wasmResult, []byte{0, 0})
	}

	finalizedBlockHeader, err := instance.FinalizeBlock()
	require.NoError(t, err)

	return &types.Block{
		Header: *finalizedBlockHeader,
		Body:   *types.NewBody(types.BytesArrayToExtrinsics(extrinsics)),
	}
}
