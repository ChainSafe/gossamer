package wasmer

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/trie"

	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func TestInstance_Version_PolkadotRuntime(t *testing.T) {
	expected := &runtime.Version{
		Spec_name:         []byte("polkadot"),
		Impl_name:         []byte("parity-polkadot"),
		Authoring_version: 0,
		Spec_version:      25,
		Impl_version:      0,
	}

	instance := NewTestInstance(t, runtime.POLKADOT_RUNTIME)

	ret, err := instance.exec(runtime.CoreVersion, []byte{})
	require.Nil(t, err)

	version := &runtime.VersionAPI{
		RuntimeVersion: &runtime.Version{},
		API:            nil,
	}
	version.Decode(ret)
	require.Nil(t, err)

	t.Logf("Spec_name: %s\n", version.RuntimeVersion.Spec_name)
	t.Logf("Impl_name: %s\n", version.RuntimeVersion.Impl_name)
	t.Logf("Authoring_version: %d\n", version.RuntimeVersion.Authoring_version)
	t.Logf("Spec_version: %d\n", version.RuntimeVersion.Spec_version)
	t.Logf("Impl_version: %d\n", version.RuntimeVersion.Impl_version)

	require.Equal(t, expected, version.RuntimeVersion)
}

func TestInstance_Version_NodeRuntime(t *testing.T) {
	expected := &runtime.Version{
		Spec_name:         []byte("node"),
		Impl_name:         []byte("substrate-node"),
		Authoring_version: 10,
		Spec_version:      260,
		Impl_version:      0,
	}

	instance := NewTestInstance(t, runtime.NODE_RUNTIME)

	ret, err := instance.exec(runtime.CoreVersion, []byte{})
	require.Nil(t, err)

	version := &runtime.VersionAPI{
		RuntimeVersion: &runtime.Version{},
		API:            nil,
	}
	version.Decode(ret)
	require.Nil(t, err)

	t.Logf("Spec_name: %s\n", version.RuntimeVersion.Spec_name)
	t.Logf("Impl_name: %s\n", version.RuntimeVersion.Impl_name)
	t.Logf("Authoring_version: %d\n", version.RuntimeVersion.Authoring_version)
	t.Logf("Spec_version: %d\n", version.RuntimeVersion.Spec_version)
	t.Logf("Impl_version: %d\n", version.RuntimeVersion.Impl_version)

	require.Equal(t, expected, version.RuntimeVersion)
}

func TestInstance_GrandpaAuthorities_NodeRuntime(t *testing.T) {
	tt := trie.NewEmptyTrie()

	value, err := common.HexToBytes("0x0108eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640100000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000")
	require.NoError(t, err)

	err = tt.Put(runtime.GrandpaAuthoritiesKey, value)
	require.NoError(t, err)

	rt := NewTestInstanceWithTrie(t, runtime.NODE_RUNTIME, tt, log.LvlTrace)

	auths, err := rt.GrandpaAuthorities()
	require.NoError(t, err)

	authABytes, _ := common.HexToBytes("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authBBytes, _ := common.HexToBytes("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	authA, _ := ed25519.NewPublicKey(authABytes)
	authB, _ := ed25519.NewPublicKey(authBBytes)

	expected := []*types.Authority{
		{Key: authA, Weight: 1},
		{Key: authB, Weight: 1},
	}

	require.Equal(t, expected, auths)
}

func TestInstance_GrandpaAuthorities_PolkadotRuntime(t *testing.T) {
	tt := trie.NewEmptyTrie()

	value, err := common.HexToBytes("0x0108eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640100000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000")
	require.NoError(t, err)

	err = tt.Put(runtime.GrandpaAuthoritiesKey, value)
	require.NoError(t, err)

	rt := NewTestInstanceWithTrie(t, runtime.POLKADOT_RUNTIME, tt, log.LvlTrace)

	auths, err := rt.GrandpaAuthorities()
	require.NoError(t, err)

	authABytes, _ := common.HexToBytes("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authBBytes, _ := common.HexToBytes("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	authA, _ := ed25519.NewPublicKey(authABytes)
	authB, _ := ed25519.NewPublicKey(authBBytes)

	expected := []*types.Authority{
		{Key: authA, Weight: 1},
		{Key: authB, Weight: 1},
	}

	require.Equal(t, expected, auths)
}

func TestInstance_BabeConfiguration_NodeRuntime_NoAuthorities(t *testing.T) {
	rt := NewTestInstance(t, runtime.NODE_RUNTIME)
	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	expected := &types.BabeConfiguration{
		SlotDuration:       3000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: nil,
		Randomness:         [32]byte{},
		SecondarySlots:     true,
	}

	require.Equal(t, expected, cfg)
}

func TestInstance_BabeConfiguration_NodeRuntime_WithAuthorities(t *testing.T) {
	tt := trie.NewEmptyTrie()

	rvalue, err := common.HexToHash("0x01")
	require.NoError(t, err)
	err = tt.Put(runtime.BABERandomnessKey(), rvalue[:])
	require.NoError(t, err)

	avalue, err := common.HexToBytes("0x08eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640100000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000")
	require.NoError(t, err)

	err = tt.Put(runtime.BABEAuthoritiesKey(), avalue)
	require.NoError(t, err)

	rt := NewTestInstanceWithTrie(t, runtime.NODE_RUNTIME, tt, log.LvlTrace)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	authA, _ := common.HexToHash("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authB, _ := common.HexToHash("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	expectedAuthData := []*types.AuthorityRaw{
		{Key: authA, Weight: 1},
		{Key: authB, Weight: 1},
	}

	expected := &types.BabeConfiguration{
		SlotDuration:       3000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: expectedAuthData,
		Randomness:         [32]byte{1},
		SecondarySlots:     true,
	}

	require.Equal(t, expected, cfg)
}

func TestInstance_InitializeBlock_NodeRuntime(t *testing.T) {
	rt := NewTestInstance(t, runtime.NODE_RUNTIME)

	header := &types.Header{
		Number: big.NewInt(1),
		Digest: [][]byte{},
	}

	err := rt.InitializeBlock(header)
	require.NoError(t, err)
}

func TestInstance_InitializeBlock_PolkadotRuntime(t *testing.T) {
	rt := NewTestInstance(t, runtime.POLKADOT_RUNTIME)

	header := &types.Header{
		Number: big.NewInt(1),
		Digest: [][]byte{},
	}

	err := rt.InitializeBlock(header)
	require.NoError(t, err)
}

func buildBlock(t *testing.T, instance runtime.Instance) *types.Block {
	header := &types.Header{
		ParentHash: trie.EmptyHash,
		Number:     big.NewInt(77),
		Digest:     [][]byte{},
	}

	err := instance.InitializeBlock(header)
	require.NoError(t, err)

	idata := types.NewInherentsData()
	err = idata.SetInt64Inherent(types.Timstap0, uint64(time.Now().Unix()))
	require.NoError(t, err)

	err = idata.SetInt64Inherent(types.Babeslot, 1)
	require.NoError(t, err)

	err = idata.SetBigIntInherent(types.Finalnum, big.NewInt(0))
	require.NoError(t, err)

	ienc, err := idata.Encode()
	require.NoError(t, err)

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as extrinsics
	inherentExts, err := instance.InherentExtrinsics(ienc)
	require.NoError(t, err)

	// decode inherent extrinsics
	exts, err := scale.Decode(inherentExts, [][]byte{})
	require.NoError(t, err)

	// apply each inherent extrinsic
	for _, ext := range exts.([][]byte) {
		in, err := scale.Encode(ext) //nolint
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(append([]byte{1}, in...))
		require.NoError(t, err, in)
		require.Equal(t, ret, []byte{0, 0})
	}

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)

	res.Number = header.Number

	expected := &types.Header{
		ParentHash: header.ParentHash,
		Number:     big.NewInt(77),
		Digest:     [][]byte{},
	}

	require.Equal(t, expected.ParentHash, res.ParentHash)
	require.Equal(t, expected.Number, res.Number)
	require.Equal(t, expected.Digest, res.Digest)
	require.NotEqual(t, common.Hash{}, res.StateRoot)
	require.NotEqual(t, common.Hash{}, res.ExtrinsicsRoot)
	require.NotEqual(t, trie.EmptyHash, res.StateRoot)

	return &types.Block{
		Header: res,
		Body:   types.NewBody(inherentExts),
	}
}

func TestInstance_FinalizeBlock_NodeRuntime(t *testing.T) {
	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	buildBlock(t, instance)
}

func TestInstance_ExecuteBlock_NodeRuntime(t *testing.T) {
	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	block := buildBlock(t, instance)

	// reset state back to parent state before executing
	parentState := storage.NewTestTrieState(t, nil)
	instance.SetContext(parentState)

	_, err := instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_PolkadotRuntime(t *testing.T) {
	DefaultTestLogLvl = 0

	instance := NewTestInstance(t, runtime.POLKADOT_RUNTIME)
	block := buildBlock(t, instance)

	// reset state back to parent state before executing
	parentState := storage.NewTestTrieState(t, nil)
	instance.SetContext(parentState)

	_, err := instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_PolkadotRuntime_PolkadotBlock1(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/polkadot/genesis-raw.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	expectedGenesisRoot := common.MustHexToHash("0x29d0d972cd27cbc511e9589fcb7a4506d5eb6a9e8df205f00472e5ab354a4e17")
	require.Equal(t, expectedGenesisRoot, genTrie.MustHash())

	// set state to genesis state
	genState := storage.NewTestTrieState(t, genTrie)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 5

	instance, err := NewRuntimeFromGenesis(gen, cfg)
	require.NoError(t, err)

	// block data is received from querying a polkadot node
	body := []byte{8, 40, 4, 3, 0, 11, 80, 149, 160, 81, 114, 1, 16, 4, 20, 0, 0}
	exts, err := scale.Decode(body, [][]byte{})
	require.NoError(t, err)
	require.Equal(t, 2, len(exts.([][]byte)))

	// polkadot block 1, from polkadot.js
	block := &types.Block{
		Header: &types.Header{
			ParentHash:     common.MustHexToHash("0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3"),
			Number:         big.NewInt(1),
			StateRoot:      common.MustHexToHash("0xc56fcd6e7a757926ace3e1ecff9b4010fc78b90d459202a339266a7f6360002f"),
			ExtrinsicsRoot: common.MustHexToHash("0x9a87f6af64ef97aff2d31bebfdd59f8fe2ef6019278b634b2515a38f1c4c2420"),
			Digest:         [][]byte{},
		},
		Body: types.NewBody(body),
	}

	_, _ = instance.ExecuteBlock(block) // TODO: complete this
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock1(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/ksmcc/genesis-raw.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	expectedGenesisRoot := common.MustHexToHash("0xb0006203c3a6e6bd2c6a17b1d4ae8ca49a31da0f4579da950b127774b44aef6b")
	require.Equal(t, expectedGenesisRoot, genTrie.MustHash())

	// set state to genesis state
	genState := storage.NewTestTrieState(t, genTrie)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 5

	instance, err := NewRuntimeFromGenesis(gen, cfg)
	require.NoError(t, err)

	// block data is received from querying a polkadot node
	body := []byte{8, 40, 4, 2, 0, 11, 144, 17, 14, 179, 110, 1, 16, 4, 20, 0, 0}
	exts, err := scale.Decode(body, [][]byte{})
	require.NoError(t, err)
	require.Equal(t, 2, len(exts.([][]byte)))

	// kusama block 1, from polkadot.js
	block := &types.Block{
		Header: &types.Header{
			ParentHash:     common.MustHexToHash("0xb0a8d493285c2df73290dfb7e61f870f17b41801197a149ca93654499ea3dafe"),
			Number:         big.NewInt(1),
			StateRoot:      common.MustHexToHash("0xfabb0c6e92d29e8bb2167f3c6fb0ddeb956a4278a3cf853661af74a076fc9cb7"),
			ExtrinsicsRoot: common.MustHexToHash("0xa35fb7f7616f5c979d48222b3d2fa7cb2331ef73954726714d91ca945cc34fd8"),
			Digest:         [][]byte{},
		},
		Body: types.NewBody(body),
	}

	_, _ = instance.ExecuteBlock(block) // TODO: complete this'
}
