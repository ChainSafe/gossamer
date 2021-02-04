package life

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

func newInstanceFromGenesis(t *testing.T) *Instance {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/gssmr/genesis-raw.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	// set state to genesis state
	genState := storage.NewTestTrieState(t, genTrie)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 4

	instance, err := NewRuntimeFromGenesis(gen, cfg)
	require.NoError(t, err)
	return instance
}

func TestInstance_Version_NodeRuntime(t *testing.T) {
	expected := &runtime.Version{
		Spec_name:         []byte("node"),
		Impl_name:         []byte("substrate-node"),
		Authoring_version: 10,
		Spec_version:      260,
		Impl_version:      0,
	}

	instance := newInstanceFromGenesis(t)

	version, err := instance.Version()
	require.NoError(t, err)

	t.Logf("Spec_name: %s\n", version.RuntimeVersion.Spec_name)
	t.Logf("Impl_name: %s\n", version.RuntimeVersion.Impl_name)
	t.Logf("Authoring_version: %d\n", version.RuntimeVersion.Authoring_version)
	t.Logf("Spec_version: %d\n", version.RuntimeVersion.Spec_version)
	t.Logf("Impl_version: %d\n", version.RuntimeVersion.Impl_version)

	require.Equal(t, expected, version.RuntimeVersion)
}

func TestInstance_BabeConfiguration_NodeRuntime_WithAuthorities(t *testing.T) {
	instance := newInstanceFromGenesis(t)
	cfg, err := instance.BabeConfiguration()
	require.NoError(t, err)

	kr, _ := keystore.NewSr25519Keyring()

	expectedAuthData := []*types.AuthorityRaw{}

	for _, kp := range kr.Keys {
		kb := [32]byte{}
		copy(kb[:], kp.Public().Encode())
		expectedAuthData = append(expectedAuthData, &types.AuthorityRaw{
			Key:    kb,
			Weight: 1,
		})
	}

	expected := &types.BabeConfiguration{
		SlotDuration:       3000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: expectedAuthData,
		Randomness:         [32]byte{},
		SecondarySlots:     true,
	}

	require.Equal(t, expected, cfg)
}

func buildBlock(t *testing.T, instance *Instance) *types.Block {
	header := &types.Header{
		ParentHash: trie.EmptyHash,
		Number:     big.NewInt(1),
		Digest:     types.Digest{},
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

	babeDigest := types.NewBabePrimaryPreDigest(0, 1, [32]byte{}, [64]byte{})
	data := babeDigest.Encode()
	preDigest := types.NewBABEPreRuntimeDigest(data)
	res.Digest = types.Digest{preDigest}

	expected := &types.Header{
		ParentHash: header.ParentHash,
		Number:     big.NewInt(1),
		Digest:     types.Digest{preDigest},
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
	instance := newInstanceFromGenesis(t)
	buildBlock(t, instance)
}
