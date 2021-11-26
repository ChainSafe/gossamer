// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer/testdata"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/stretchr/testify/require"
)

func createTestExtrinsic(t *testing.T, rt runtime.Instance, genHash common.Hash, nonce uint64) types.Extrinsic {
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

	c, err := ctypes.NewCall(meta, "System.remark", []byte{0xab, 0xcd})
	require.NoError(t, err)

	ext := ctypes.NewExtrinsic(c)
	o := ctypes.SignatureOptions{
		BlockHash:          ctypes.Hash(genHash),
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

	return types.Extrinsic(common.MustHexToBytes(extEnc))
}

func TestInstance_Version_NodeRuntime_v098(t *testing.T) {
	expected := runtime.NewVersionData(
		[]byte("node"),
		[]byte("substrate-node"),
		10,
		267,
		0,
		nil,
		2,
	)

	instance := NewTestInstance(t, runtime.NODE_RUNTIME_v098)

	version, err := instance.Version()
	require.Nil(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())
	t.Logf("TransactionVersion: %d\n", version.TransactionVersion())

	require.Equal(t, 13, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}

func TestInstance_Version_PolkadotRuntime_v0910(t *testing.T) {
	expected := runtime.NewVersionData(
		[]byte("polkadot"),
		[]byte("parity-polkadot"),
		0,
		9100,
		0,
		nil,
		8,
	)

	instance := NewTestInstance(t, runtime.POLKADOT_RUNTIME_v0910)
	version, err := instance.Version()
	require.NoError(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())
	t.Logf("TransactionVersion: %d\n", version.TransactionVersion())

	require.Equal(t, 14, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}

func TestInstance_Version_PolkadotRuntime(t *testing.T) {
	expected := runtime.NewVersionData(
		[]byte("polkadot"),
		[]byte("parity-polkadot"),
		0,
		25,
		0,
		nil,
		5,
	)

	instance := NewTestInstance(t, runtime.POLKADOT_RUNTIME)

	version, err := instance.Version()
	require.Nil(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())
	t.Logf("TransactionVersion: %d\n", version.TransactionVersion())

	require.Equal(t, 12, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}

func TestInstance_Version_KusamaRuntime(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/kusama/genesis.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	expectedGenesisRoot := common.MustHexToHash("0xb0006203c3a6e6bd2c6a17b1d4ae8ca49a31da0f4579da950b127774b44aef6b")
	require.Equal(t, expectedGenesisRoot, genTrie.MustHash())

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 4

	instance, err := NewRuntimeFromGenesis(cfg)
	require.NoError(t, err)

	expected := runtime.NewVersionData(
		[]byte("kusama"),
		[]byte("parity-kusama"),
		2,
		1020,
		0,
		nil,
		0,
	)

	version, err := instance.(*Instance).Version()
	require.NoError(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())
	t.Logf("TransactionVersion: %d\n", version.TransactionVersion())

	require.Equal(t, 12, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}

func TestInstance_Version_NodeRuntime(t *testing.T) {
	expected := runtime.NewVersionData(
		[]byte("node"),
		[]byte("substrate-node"),
		10,
		264,
		0,
		nil,
		2,
	)

	instance := NewTestInstance(t, runtime.NODE_RUNTIME)

	version, err := instance.Version()
	require.Nil(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())
	t.Logf("TransactionVersion: %d\n", version.TransactionVersion())

	require.Equal(t, 13, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}

func TestInstance_Version_DevRuntime(t *testing.T) {
	expected := runtime.NewVersionData(
		[]byte("node"),
		[]byte("gossamer-node"),
		10,
		260,
		0,
		nil,
		1,
	)

	instance := NewTestInstance(t, runtime.DEV_RUNTIME)

	version, err := instance.Version()
	require.Nil(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())
	t.Logf("TransactionVersion: %d\n", version.TransactionVersion())

	require.Equal(t, 12, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}

func balanceKey(t *testing.T, pub []byte) []byte {
	h0, err := common.Twox128Hash([]byte("System"))
	require.NoError(t, err)
	h1, err := common.Twox128Hash([]byte("Account"))
	require.NoError(t, err)
	h2, err := common.Blake2b128(pub)
	require.NoError(t, err)
	return append(append(append(h0, h1...), h2...), pub...)
}

func TestNodeRuntime_ValidateTransaction(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/gssmr/genesis.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 4
	nodeStorage := runtime.NodeStorage{}
	nodeStorage.BaseDB = runtime.NewInMemoryDB(t)
	cfg.NodeStorage = nodeStorage

	rt, err := NewRuntimeFromGenesis(cfg)
	require.NoError(t, err)

	alicePub := common.MustHexToBytes("0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d")
	aliceBalanceKey := balanceKey(t, alicePub)

	accInfo := types.AccountInfo{
		Nonce: 0,
		Data: struct {
			Free       *scale.Uint128
			Reserved   *scale.Uint128
			MiscFrozen *scale.Uint128
			FreeFrozen *scale.Uint128
		}{
			Free:       scale.MustNewUint128(big.NewInt(1152921504606846976)),
			Reserved:   scale.MustNewUint128(big.NewInt(0)),
			MiscFrozen: scale.MustNewUint128(big.NewInt(0)),
			FreeFrozen: scale.MustNewUint128(big.NewInt(0)),
		},
	}

	encBal, err := scale.Marshal(accInfo)
	require.NoError(t, err)

	rt.(*Instance).ctx.Storage.Set(aliceBalanceKey, encBal)
	// this key is System.UpgradedToDualRefCount -> set to true since all accounts have been upgraded to v0.9 format
	rt.(*Instance).ctx.Storage.Set(common.UpgradedToDualRefKey, []byte{1})

	genesisHeader := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: genTrie.MustHash(),
	}

	ext := createTestExtrinsic(t, rt, genesisHeader.Hash(), 0)
	ext = append([]byte{byte(types.TxnExternal)}, ext...)

	_ = buildBlockVdt(t, rt, genesisHeader.Hash())
	_, err = rt.ValidateTransaction(ext)
	require.NoError(t, err)
}

func TestInstance_GrandpaAuthorities_NodeRuntime(t *testing.T) {
	tt := trie.NewEmptyTrie()

	value, err := common.HexToBytes("0x0108eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640100000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000") //nolint:lll
	require.NoError(t, err)

	tt.Put(runtime.GrandpaAuthoritiesKey, value)

	rt := NewTestInstanceWithTrie(t, runtime.NODE_RUNTIME, tt, log.Info)

	auths, err := rt.GrandpaAuthorities()
	require.NoError(t, err)

	authABytes, _ := common.HexToBytes("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authBBytes, _ := common.HexToBytes("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	authA, _ := ed25519.NewPublicKey(authABytes)
	authB, _ := ed25519.NewPublicKey(authBBytes)

	expected := []types.Authority{
		{Key: authA, Weight: 1},
		{Key: authB, Weight: 1},
	}

	require.Equal(t, expected, auths)
}

func TestInstance_GrandpaAuthorities_PolkadotRuntime(t *testing.T) {
	tt := trie.NewEmptyTrie()

	value, err := common.HexToBytes("0x0108eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640100000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000") //nolint:lll
	require.NoError(t, err)

	tt.Put(runtime.GrandpaAuthoritiesKey, value)

	rt := NewTestInstanceWithTrie(t, runtime.POLKADOT_RUNTIME, tt, log.Info)

	auths, err := rt.GrandpaAuthorities()
	require.NoError(t, err)

	authABytes, _ := common.HexToBytes("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authBBytes, _ := common.HexToBytes("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	authA, _ := ed25519.NewPublicKey(authABytes)
	authB, _ := ed25519.NewPublicKey(authBBytes)

	expected := []types.Authority{
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
		C2:                 2,
		GenesisAuthorities: nil,
		Randomness:         [32]byte{},
		SecondarySlots:     1,
	}

	require.Equal(t, expected, cfg)
}

func TestInstance_BabeConfiguration_DevRuntime_NoAuthorities(t *testing.T) {
	rt := NewTestInstance(t, runtime.DEV_RUNTIME)
	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	expected := &types.BabeConfiguration{
		SlotDuration:       3000,
		EpochLength:        200,
		C1:                 1,
		C2:                 1,
		GenesisAuthorities: nil,
		Randomness:         [32]byte{},
		SecondarySlots:     1,
	}

	require.Equal(t, expected, cfg)
}

func TestInstance_BabeConfiguration_NodeRuntime_WithAuthorities(t *testing.T) {
	tt := trie.NewEmptyTrie()

	rvalue, err := common.HexToHash("0x01")
	require.NoError(t, err)
	tt.Put(runtime.BABERandomnessKey(), rvalue[:])

	avalue, err := common.HexToBytes("0x08eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640100000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000") //nolint:lll
	require.NoError(t, err)

	tt.Put(runtime.BABEAuthoritiesKey(), avalue)

	rt := NewTestInstanceWithTrie(t, runtime.NODE_RUNTIME, tt, log.Info)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	authA, _ := common.HexToHash("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authB, _ := common.HexToHash("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	expectedAuthData := []types.AuthorityRaw{
		{Key: authA, Weight: 1},
		{Key: authB, Weight: 1},
	}

	expected := &types.BabeConfiguration{
		SlotDuration:       3000,
		EpochLength:        200,
		C1:                 1,
		C2:                 2,
		GenesisAuthorities: expectedAuthData,
		Randomness:         [32]byte{1},
		SecondarySlots:     1,
	}

	require.Equal(t, expected, cfg)
}

func TestInstance_InitializeBlock_NodeRuntime(t *testing.T) {
	rt := NewTestInstance(t, runtime.NODE_RUNTIME)

	header := &types.Header{
		Number: big.NewInt(1),
		Digest: types.NewDigest(),
	}

	err := rt.InitializeBlock(header)
	require.NoError(t, err)
}

func TestInstance_InitializeBlock_PolkadotRuntime(t *testing.T) {
	rt := NewTestInstance(t, runtime.POLKADOT_RUNTIME)

	header := &types.Header{
		Number: big.NewInt(1),
		Digest: types.NewDigest(),
	}

	err := rt.InitializeBlock(header)
	require.NoError(t, err)
}

func buildBlockVdt(t *testing.T, instance runtime.Instance, parentHash common.Hash) *types.Block {
	header := &types.Header{
		ParentHash: parentHash,
		Number:     big.NewInt(1),
		Digest:     types.NewDigest(),
	}

	err := instance.InitializeBlock(header)
	require.NoError(t, err)

	idata := types.NewInherentsData()
	err = idata.SetInt64Inherent(types.Timstap0, uint64(time.Now().Unix()))
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

func TestInstance_FinalizeBlock_NodeRuntime(t *testing.T) {
	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	buildBlockVdt(t, instance, common.Hash{})
}

func TestInstance_ExecuteBlock_NodeRuntime(t *testing.T) {
	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	block := buildBlockVdt(t, instance, common.Hash{})

	// reset state back to parent state before executing
	parentState, err := storage.NewTrieState(nil)
	require.NoError(t, err)
	instance.SetContextStorage(parentState)

	block.Header.Digest = types.NewDigest()
	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_GossamerRuntime(t *testing.T) {
	t.Skip() // TODO: this fails with "syscall frame is no longer valid" (#1026)
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/gssmr/genesis.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 4

	instance, err := NewRuntimeFromGenesis(cfg)
	require.NoError(t, err)
	block := buildBlockVdt(t, instance, common.Hash{})

	// reset state back to parent state before executing
	parentState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)
	instance.SetContextStorage(parentState)

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ApplyExtrinsic_GossamerRuntime(t *testing.T) {
	t.Skip() // TODO: this fails with "syscall frame is no longer valid" (#1026)
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/gssmr/genesis.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 4

	instance, err := NewRuntimeFromGenesis(cfg)
	require.NoError(t, err)

	// reset state back to parent state before executing
	parentState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)
	instance.SetContextStorage(parentState)

	parentHash := common.Hash{}
	header, err := types.NewHeader(parentHash, common.Hash{}, common.Hash{}, big.NewInt(1), types.NewDigest())
	require.NoError(t, err)
	err = instance.InitializeBlock(header)
	require.NoError(t, err)

	ext := createTestExtrinsic(t, instance, parentHash, 0)
	enc, err := scale.Marshal(ext)
	require.NoError(t, err)

	res, err := instance.ApplyExtrinsic(enc)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0}, res)
}

func TestInstance_ExecuteBlock_PolkadotRuntime(t *testing.T) {
	DefaultTestLogLvl = 0

	instance := NewTestInstance(t, runtime.POLKADOT_RUNTIME)
	block := buildBlockVdt(t, instance, common.Hash{})

	// reset state back to parent state before executing
	parentState, err := storage.NewTrieState(nil)
	require.NoError(t, err)
	instance.SetContextStorage(parentState)

	block.Header.Digest = types.NewDigest()
	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_PolkadotRuntime_PolkadotBlock1(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/polkadot/genesis.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	expectedGenesisRoot := common.MustHexToHash("0x29d0d972cd27cbc511e9589fcb7a4506d5eb6a9e8df205f00472e5ab354a4e17")
	require.Equal(t, expectedGenesisRoot, genTrie.MustHash())

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 5

	instance, err := NewRuntimeFromGenesis(cfg)
	require.NoError(t, err)

	// block data is received from querying a polkadot node
	body := []byte{8, 40, 4, 3, 0, 11, 80, 149, 160, 81, 114, 1, 16, 4, 20, 0, 0}
	var exts [][]byte
	err = scale.Unmarshal(body, &exts)
	require.NoError(t, err)
	require.Equal(t, 2, len(exts))

	// digest data received from querying polkadot node
	digestBytes := common.MustHexToBytes("0x0c0642414245b501010000000093decc0f00000000362ed8d6055645487fe42e9c8640be651f70a3a2a03658046b2b43f021665704501af9b1ca6e974c257e3d26609b5f68b5b0a1da53f7f252bbe5d94948c39705c98ffa4b869dd44ac29528e3723d619cc7edf1d3f7b7a57a957f6a7e9bdb270a044241424549040118fa3437b10f6e7af8f31362df3a179b991a8c56313d1bcd6307a4d0c734c1ae310100000000000000d2419bc8835493ac89eb09d5985281f5dff4bc6c7a7ea988fd23af05f301580a0100000000000000ccb6bef60defc30724545d57440394ed1c71ea7ee6d880ed0e79871a05b5e40601000000000000005e67b64cf07d4d258a47df63835121423551712844f5b67de68e36bb9a21e12701000000000000006236877b05370265640c133fec07e64d7ca823db1dc56f2d3584b3d7c0f1615801000000000000006c52d02d95c30aa567fda284acf25025ca7470f0b0c516ddf94475a1807c4d250100000000000000000000000000000000000000000000000000000000000000000000000000000005424142450101d468680c844b19194d4dfbdc6697a35bf2b494bda2c5a6961d4d4eacfbf74574379ba0d97b5bb650c2e8670a63791a727943bcb699dc7a228bdb9e0a98c9d089") //nolint:lll

	digest := types.NewDigest()
	err = scale.Unmarshal(digestBytes, &digest)
	require.NoError(t, err)

	// polkadot block 1, from polkadot.js
	block := &types.Block{
		Header: types.Header{
			ParentHash:     common.MustHexToHash("0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3"),
			Number:         big.NewInt(1),
			StateRoot:      common.MustHexToHash("0xc56fcd6e7a757926ace3e1ecff9b4010fc78b90d459202a339266a7f6360002f"),
			ExtrinsicsRoot: common.MustHexToHash("0x9a87f6af64ef97aff2d31bebfdd59f8fe2ef6019278b634b2515a38f1c4c2420"),
			Digest:         digest,
		},
		Body: *types.NewBody(types.BytesArrayToExtrinsics(exts)),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock1(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/kusama/genesis.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	expectedGenesisRoot := common.MustHexToHash("0xb0006203c3a6e6bd2c6a17b1d4ae8ca49a31da0f4579da950b127774b44aef6b")
	require.Equal(t, expectedGenesisRoot, genTrie.MustHash())

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 4

	instance, err := NewRuntimeFromGenesis(cfg)
	require.NoError(t, err)

	// block data is received from querying a polkadot node
	body := []byte{8, 40, 4, 2, 0, 11, 144, 17, 14, 179, 110, 1, 16, 4, 20, 0, 0}
	var exts [][]byte
	err = scale.Unmarshal(body, &exts)
	require.NoError(t, err)
	require.Equal(t, 2, len(exts))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x0c0642414245340201000000ef55a50f00000000044241424549040118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000054241424501014625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d") //nolint:lll

	digest := types.NewDigest()
	err = scale.Unmarshal(digestBytes, &digest)
	require.NoError(t, err)

	// kusama block 1, from polkadot.js
	block := &types.Block{
		Header: types.Header{
			ParentHash:     common.MustHexToHash("0xb0a8d493285c2df73290dfb7e61f870f17b41801197a149ca93654499ea3dafe"),
			Number:         big.NewInt(1),
			StateRoot:      common.MustHexToHash("0xfabb0c6e92d29e8bb2167f3c6fb0ddeb956a4278a3cf853661af74a076fc9cb7"),
			ExtrinsicsRoot: common.MustHexToHash("0xa35fb7f7616f5c979d48222b3d2fa7cb2331ef73954726714d91ca945cc34fd8"),
			Digest:         digest,
		},
		Body: *types.NewBody(types.BytesArrayToExtrinsics(exts)),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock3784(t *testing.T) {
	gossTrie3783 := newTrieFromPairs(t, "../test_data/kusama/block3783.out")
	expectedRoot := common.MustHexToHash("0x948338bc0976aee78879d559a1f42385407e5a481b05a91d2a9386aa7507e7a0")
	require.Equal(t, expectedRoot, gossTrie3783.MustHash())

	// set state to genesis state
	state3783, err := storage.NewTrieState(gossTrie3783)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state3783
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(gossTrie3783, cfg)
	require.NoError(t, err)

	// block data is received from querying a polkadot node
	body := common.MustHexToBytes("0x10280402000bb00d69b46e0114040900193b10041400009101041300eaaec5728cd6ea9160ff92a49bb45972c532d2163241746134726aaa5b2f72129d8650715320f23765c6306503669f69bf684b188dea73b1e247dd1dd166513b1c13daa387c35f24ac918d2fa772b73cffd20204a8875e48a1b11bb3229deb7f00") //nolint:lll
	var exts [][]byte
	err = scale.Unmarshal(body, &exts)
	require.NoError(t, err)
	require.Equal(t, 4, len(exts))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x080642414245340203000000bd64a50f0000000005424142450101bc0d6850dba8d32ea1dbe26cb4ac56da6cca662c7cc642dc8eed32d2bddd65029f0721436eafeebdf9b4f17d1673c6bc6c3c51fe3dda3121a5fc60c657a5808b") //nolint:lll

	digest := types.NewDigest()
	err = scale.Unmarshal(digestBytes, &digest)
	require.NoError(t, err)

	// kusama block 3784, from polkadot.js
	block := &types.Block{
		Header: types.Header{
			ParentHash:     common.MustHexToHash("0x4843b4aa38cf2e3e2f6fae401b98dd705bed668a82dd3751dc38f1601c814ca8"),
			Number:         big.NewInt(3784),
			StateRoot:      common.MustHexToHash("0xac44cc18ec22f0f3fca39dfe8725c0383af1c982a833e081fbb2540e46eb09a5"),
			ExtrinsicsRoot: common.MustHexToHash("0x52b7d4852fc648cb8f908901e1e36269593c25050c31718454bca74b69115d12"),
			Digest:         digest,
		},
		Body: *types.NewBody(types.BytesArrayToExtrinsics(exts)),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock901442(t *testing.T) {
	ksmTrie901441 := newTrieFromPairs(t, "../test_data/kusama/block901441.out")
	expectedRoot := common.MustHexToHash("0x3a2ef7ee032f5810160bb8f3ffe3e3377bb6f2769ee9f79a5425973347acd504")
	require.Equal(t, expectedRoot, ksmTrie901441.MustHash())

	// set state to genesis state
	state901441, err := storage.NewTrieState(ksmTrie901441)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state901441
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(ksmTrie901441, cfg)
	require.NoError(t, err)

	body := common.MustHexToBytes("0x0c280402000b207eb80a70011c040900fa0437001004140000")
	var exts [][]byte
	err = scale.Unmarshal(body, &exts)
	require.NoError(t, err)
	require.Equal(t, 3, len(exts))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x080642414245340244000000aeffb30f00000000054241424501011cbef2a084a774c34d9990c7bfc6b4d2d5e9f5b59feca792cd2bb89a890c2a6f09668b5e8224879f007f49f299d25fbb3c0f30d94fb8055e07fa8a4ed10f8083") //nolint:lll

	digest := types.NewDigest()
	err = scale.Unmarshal(digestBytes, &digest)
	require.NoError(t, err)
	require.Equal(t, 2, len(digest.Types))

	// kusama block 901442, from polkadot.js
	block := &types.Block{
		Header: types.Header{
			ParentHash:     common.MustHexToHash("0x68d9c5f75225f09d7ce493eff8aabac7bae8b65cb81a2fd532a99fbb8c663931"),
			Number:         big.NewInt(901442),
			StateRoot:      common.MustHexToHash("0x6ea065f850894c5b58cb1a73ec887e56842851943641149c57cea357cae4f596"),
			ExtrinsicsRoot: common.MustHexToHash("0x13483a4c148fff5f072e86b5af52bf031556514e9c87ea19f9e31e7b13c0c414"),
			Digest:         digest,
		},
		Body: *types.NewBody(types.BytesArrayToExtrinsics(exts)),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock1377831(t *testing.T) {
	ksmTrie := newTrieFromPairs(t, "../test_data/kusama/block1377830.out")
	expectedRoot := common.MustHexToHash("0xe4de6fecda9e9e35f937d159665cf984bc1a68048b6c78912de0aeb6bd7f7e99")
	require.Equal(t, expectedRoot, ksmTrie.MustHash())

	// set state to genesis state
	state, err := storage.NewTrieState(ksmTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(ksmTrie, cfg)
	require.NoError(t, err)

	body := common.MustHexToBytes("0x08280402000b60c241c070011004140000")
	var exts [][]byte
	err = scale.Unmarshal(body, &exts)
	require.NoError(t, err)
	require.Equal(t, 2, len(exts))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x080642414245b50101020000008abebb0f00000000045553c32a949242580161bcc35d7c3e492e66defdcf4525d7a338039590012f42660acabf1952a2d5d01725601705404d6ac671507a6aa2cf09840afbdfbb006f48062dae16c56b8dc5c6ea6ffba854b7e8f46e153e98c238cbe7bbb1556f0b0542414245010136914c6832dd5ba811a975a3b654d76a1ec81684f4b03d115ce2e694feadc96411930438fde4beb008c5f8e26cfa2f5b554fa3814b5b73d31f348446fd4fd688") //nolint:lll

	digest := types.NewDigest()
	err = scale.Unmarshal(digestBytes, &digest)
	require.NoError(t, err)
	require.Equal(t, 2, len(digest.Types))

	// kusama block 1377831, from polkadot.js
	block := &types.Block{
		Header: types.Header{
			ParentHash:     common.MustHexToHash("0xca387b3cc045e8848277069d8794cbf077b08218c0b55f74d81dd750b14e768c"),
			Number:         big.NewInt(1377831),
			StateRoot:      common.MustHexToHash("0x7e5569e652c4b1a3cecfcf5e5e64a97fe55071d34bab51e25626ec20cae05a02"),
			ExtrinsicsRoot: common.MustHexToHash("0x7f3ea0ed63b4053d9b75e7ee3e5b3f6ce916e8f59b7b6c5e966b7a56ea0a563a"),
			Digest:         digest,
		},
		Body: *types.NewBody(types.BytesArrayToExtrinsics(exts)),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock1482003(t *testing.T) {
	ksmTrie := newTrieFromPairs(t, "../test_data/kusama/block1482002.out")
	expectedRoot := common.MustHexToHash("0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb")
	require.Equal(t, expectedRoot, ksmTrie.MustHash())

	// set state to genesis state
	state, err := storage.NewTrieState(ksmTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(ksmTrie, cfg)
	require.NoError(t, err)

	body := common.MustHexToBytes("0x0c280402000b10c3e3e570011c04090042745a001004140000")
	var exts [][]byte
	err = scale.Unmarshal(body, &exts)
	require.NoError(t, err)
	require.Equal(t, 3, len(exts))

	// digest from polkadot.js
	digestBytes := testdata.DigestKusama1482002(t)

	digest := types.NewDigest()
	err = scale.Unmarshal(digestBytes, &digest)
	require.NoError(t, err)

	require.Equal(t, 4, len(digest.Types))

	// kusama block 1482003, from polkadot.js
	block := &types.Block{
		Header: types.Header{
			ParentHash:     common.MustHexToHash("0x587f6da1bfa71a675f10dfa0f63edfcf168e8ece97eb5f526aaf0e8a8e82db3f"),
			Number:         big.NewInt(1482003),
			StateRoot:      common.MustHexToHash("0xd2de750002f33968437bdd54912dd4f55c3bddc5a391a8e0b8332568e1efea8d"),
			ExtrinsicsRoot: common.MustHexToHash("0xdf5da95780b77e83ad0bf820d5838f07a0d5131aa95a75f8dfbd01fbccb300bd"),
			Digest:         digest,
		},
		Body: *types.NewBody(types.BytesArrayToExtrinsics(exts)),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock4939774(t *testing.T) {
	t.Skip("skip for now as block4939773 is too large")
	ksmTrie := newTrieFromPairs(t, "../test_data/kusama/block4939773.out")
	expectedRoot := common.MustHexToHash("0xc45748e6e8632b44fc32b04cc4380098a9584cbd63ffbc59adce189574fc36fe")
	require.Equal(t, expectedRoot, ksmTrie.MustHash())

	// set state to genesis state
	state, err := storage.NewTrieState(ksmTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(ksmTrie, cfg)
	require.NoError(t, err)

	body := common.MustHexToBytes("0x08280402000b80eb3cd17501710984c2292bcf6f34fc2d25f7a1ebaec41c3239536f12f75417c73f7c5aca53308668016ec90c2318ee45af373755527436c4d7a257c481fdc3214634eb4b5c6711ae181827c378843da82c72191647667607ee97e0f0335f14d0876c63503b5f2b8986650304001f010200083e1f2bfd408d3b8d2266ce9b6f2d40acef27b773414537be72576ee3e6108b256eb45e26258d7ac737c3ad3af8cd1b2208d45c472ba19ebfc3e2fb834a6e904d01de574b00010000007506180228040052dac5497bbdd42583d07aa46102790d54aacdcbfac8877189e3b609117a29150b00a0724e180904001cf8853df87ca8588405e30c46a434d636c86561b955b09e2e9b27fc296bf4290b005039278c040400f49db9c8894863a7dd213be93b1c440b145cc19d4927b4c29fe5fa25e8a1667f0b005039278c040400e05f031d874257a24232076830a073a6af6851c07735de201edfc412ca8853180b005039278c0404009289e88ec986066d04f7d93d80f7a3c9794580b5e59d2a7af6b19745dd148f6f0b005039278c0404006c8aff52c496b64b476ca22e58fc54822b435abbbbcaf0c9dd7cf1ab573227790b005039278c04040044e31f7c4afa3b055696923ccb405da2ee2d9eefccf568aa3c6855dbff573e5f0b005039278c040400469ec0f872af2503a9251666fd089d0e84d3f6c8b761ee94b0e868788e0f60500b005039278c040400b41cc00e4ee2945ce9974dbb355265e39c9cf325c176147d7f6b1631af38ce590b005039278c040400d8e2f26a12d4bfc513fd32c1e5a7f14e930c3ef37997bf4e3de2fed51eed515a0b005039278c040048227b8300000000") //nolint:lll
	var exts [][]byte
	err = scale.Unmarshal(body, &exts)
	require.NoError(t, err)
	require.Equal(t, 2, len(exts))

	digestBytes := common.MustHexToBytes("0x080642414245b50101ef0100000815f30f000000004014ed1a99f017ea2c0d879d7317f51106938f879b296ff92c64319c0c70fe453d72035395da8d53e885def26e63cf90461ee549d0864f9691a4f401b31c1801730c014bc0641b307e8a30692e7d074b4656993b40d6f08698bc49dea40c11090542414245010192ed24972a8108b9bad1a8785b443efe72d4bc2069ab40eac65519fb01ff04250f44f6202d30ca88c30fee385bc8d7f51df15dddacf4e5d53788d260ce758c89") //nolint:lll
	digest := types.NewDigest()
	err = scale.Unmarshal(digestBytes, &digest)
	require.NoError(t, err)
	require.Equal(t, 2, len(digest.Types))

	block := &types.Block{
		Header: types.Header{
			ParentHash:     common.MustHexToHash("0xac08290f49cb9760a3a4c5a49351af76ba9432add29178e5cc27d4451f9126c9"),
			Number:         big.NewInt(4939774),
			StateRoot:      common.MustHexToHash("0x5d66f43cdbf1740b8ca41f0cd016602f1648fb08b74fe49f5f078845071d0a54"),
			ExtrinsicsRoot: common.MustHexToHash("0x5d887e118ee6320aca38e49cbd98adc25472c6efbf77a695ab0d6c476a4ec6e9"),
			Digest:         digest,
		},
		Body: *types.NewBody(types.BytesArrayToExtrinsics(exts)),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_PolkadotBlock1089328(t *testing.T) {
	dotTrie := newTrieFromPairs(t, "../test_data/polkadot/block1089327.json")
	expectedRoot := common.MustHexToHash("0x87ed9ebe7fb645d3b5b0255cc16e78ed022d9fbb52486105436e15a74557535b")
	require.Equal(t, expectedRoot, dotTrie.MustHash())

	// set state to genesis state
	state, err := storage.NewTrieState(dotTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(dotTrie, cfg)
	require.NoError(t, err)

	body := common.MustHexToBytes("0x0c280403000be02ab6d873011004140000b90384468e34dbdcc8da24e44b0f0d34d97ccad5ce0281e465db0cc1d8e1423d50d90a018a89185c693f77b050fa35d1f80b19608b72a6e626110e835caedf949668a12b0ad7b786accf2caac0ec874941ccea9825d50b6bb5870e1400f0e56bb4c18b87a5021501001d00862e432e0cf75693899c62691ac0f48967f815add97ae85659dcde8332708551001b000cf4da8aea0e5649a8bedbc1f08e8a8c0febe50cd5b1c9ce0da2164f19aef40f01014a87a7d3673e5c80aec79973682140828a0d1c3899f4f3cc953bd02673e11a022aaa4f269e3f1a90156db29df88f780b1540b610aeb5cd347ee703c5dff48485") //nolint:lll
	var exts [][]byte
	err = scale.Unmarshal(body, &exts)
	require.NoError(t, err)
	require.Equal(t, 3, len(exts))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x080642414245b501017b000000428edd0f00000000c4fd75c7535d8eec375d70d21cc62262247b599aa67d8a9cf2f7d1b8cb93cd1f9539f04902c33d4c0fe47f723dfed8505d31de1c04d0036a9df233ff902fce0d70060908faa4b3f481e54cbd6a52dfc20c3faac82f746d84dc03c2f824a89a0d0542414245010122041949669a56c8f11b3e3e7c803e477ad24a71ed887bc81c956b59ea8f2b30122e6042494aab60a75e0db8fdff45951e456e6053bd64eb5722600e4a13038b") //nolint:lll

	digest := types.NewDigest()
	err = scale.Unmarshal(digestBytes, &digest)
	require.NoError(t, err)
	require.Equal(t, 2, len(digest.Types))

	block := &types.Block{
		Header: types.Header{
			ParentHash:     common.MustHexToHash("0x21dc35454805411be396debf3e1d5aad8d6e9d0d7679cce0cc632ba8a647d07c"),
			Number:         big.NewInt(1089328),
			StateRoot:      common.MustHexToHash("0x257b1a7f6bc0287fcbf50676dd29817f2f7ae193cb65b31962e351917406fa23"),
			ExtrinsicsRoot: common.MustHexToHash("0x950173af1d9fdcd0be5428fc3eaf05d5f34376bd3882d9a61b348fa2dc641012"),
			Digest:         digest,
		},
		Body: *types.NewBody(types.BytesArrayToExtrinsics(exts)),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_DecodeSessionKeys(t *testing.T) {
	keys := "0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d34309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc3852042602634309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc3852042602634309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc38520426026" //nolint:lll
	pubkeys, err := common.HexToBytes(keys)
	require.NoError(t, err)

	pukeysBytes, err := scale.Marshal(pubkeys)
	require.NoError(t, err)

	instance := NewTestInstance(t, runtime.NODE_RUNTIME_v098)
	decoded, err := instance.DecodeSessionKeys(pukeysBytes)
	require.NoError(t, err)

	var decodedKeys *[]struct {
		Data []uint8
		Type [4]uint8
	}

	err = scale.Unmarshal(decoded, &decodedKeys)
	require.NoError(t, err)

	require.Len(t, *decodedKeys, 4)
}

func TestInstance_PaymentQueryInfo(t *testing.T) {
	tests := []struct {
		extB   []byte
		ext    string
		err    error
		expect *types.TransactionPaymentQueryInfo
	}{
		{
			// Was made with @polkadot/api on https://github.com/danforbes/polkadot-js-scripts/tree/create-signed-tx
			ext: "0xd1018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01bc2b6e35929aabd5b8bc4e5b0168c9bee59e2bb9d6098769f6683ecf73e44c776652d947a270d59f3d37eb9f9c8c17ec1b4cc473f2f9928ffdeef0f3abd43e85d502000000012844616e20466f72626573", //nolint:lll
			err: nil,
			expect: &types.TransactionPaymentQueryInfo{
				Weight: 1973000,
				Class:  0,
				PartialFee: &scale.Uint128{
					Upper: 0,
					Lower: uint64(1180126973000),
				},
			},
		},
		{
			// incomplete extrinsic
			ext: "0x4ccde39a5684e7a56da23b22d4d9fbadb023baa19c56495432884d0640000000000000000000000000000000",
			err: errors.New("Failed to call the `TransactionPaymentApi_query_info` exported function."), //nolint:revive
		},
		{
			// incomplete extrinsic
			extB: nil,
			err:  errors.New("Failed to call the `TransactionPaymentApi_query_info` exported function."), //nolint:revive
		},
	}

	for _, test := range tests {
		var err error
		var extBytes []byte

		if test.ext == "" {
			extBytes = test.extB
		} else {
			extBytes, err = common.HexToBytes(test.ext)
			require.NoError(t, err)
		}

		ins := NewTestInstance(t, runtime.NODE_RUNTIME)
		info, err := ins.PaymentQueryInfo(extBytes)

		if test.err != nil {
			require.Error(t, err)
			require.Equal(t, err.Error(), test.err.Error())
			continue
		}

		fmt.Println(info.PartialFee.String())
		fmt.Println(test.expect.PartialFee.String())

		require.NoError(t, err)
		require.NotNil(t, info)
		require.Equal(t, test.expect, info)
	}
}

func newTrieFromPairs(t *testing.T, filename string) *trie.Trie {
	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	rpcPairs := make(map[string]interface{})
	err = json.Unmarshal(data, &rpcPairs)
	require.NoError(t, err)
	pairs := rpcPairs["result"].([]interface{})

	entries := make(map[string]string)
	for _, pair := range pairs {
		pairArr := pair.([]interface{})
		entries[pairArr[0].(string)] = pairArr[1].(string)
	}

	tr := trie.NewEmptyTrie()
	err = tr.LoadFromMap(entries)
	require.NoError(t, err)
	return tr
}
