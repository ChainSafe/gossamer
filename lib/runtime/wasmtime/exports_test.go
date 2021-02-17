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

package wasmtime

import (
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/trie"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	if err != nil {
		log.Error("failed to generate runtime wasm file", err)
		os.Exit(1)
	}

	// Start all tests
	code := m.Run()

	runtime.RemoveFiles(wasmFilePaths)
	os.Exit(code)
}

func TestConcurrentRuntimeCalls(t *testing.T) {
	instance := NewTestInstance(t, runtime.LEGACY_NODE_RUNTIME)

	// execute 2 concurrent calls to the runtime
	go func() {
		_, _ = instance.exec(runtime.CoreVersion, []byte{})
	}()
	go func() {
		_, _ = instance.exec(runtime.CoreVersion, []byte{})
	}()
}

func TestInstance_Version_LegacyNodeRuntime(t *testing.T) {
	// https://github.com/paritytech/substrate/blob/7b1d822446982013fa5b7ad5caff35ca84f8b7d0/core/test-runtime/src/lib.rs#L73
	expected := runtime.NewLegacyVersionData(
		[]byte("node"),
		[]byte("substrate-node"),
		10,
		193,
		193,
		nil,
	)

	instance := NewTestInstance(t, runtime.LEGACY_NODE_RUNTIME)

	version, err := instance.inst.Version()
	require.Nil(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())

	require.Equal(t, 11, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
}

func TestInstance_BabeConfiguration_LegacyNodeRuntime(t *testing.T) {
	expected := &types.BabeConfiguration{
		SlotDuration:       3000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: nil,
		SecondarySlots:     1,
	}

	instance := NewTestInstance(t, runtime.LEGACY_NODE_RUNTIME)
	babeCfg, err := instance.BabeConfiguration()
	require.NoError(t, err)
	require.Equal(t, expected, babeCfg)
}

func TestInstance_GrandpaAuthorities_LegacyNodeRuntime(t *testing.T) {
	expected := []*types.Authority{}

	instance := NewTestInstance(t, runtime.LEGACY_NODE_RUNTIME)
	res, err := instance.GrandpaAuthorities()
	require.NoError(t, err)
	require.Equal(t, expected, res)
}

func TestInstance_InitializeBlock_LegacyNodeRuntime(t *testing.T) {
	header := &types.Header{
		ParentHash: trie.EmptyHash,
		Number:     big.NewInt(1),
		Digest:     types.Digest{},
	}

	instance := NewTestInstance(t, runtime.LEGACY_NODE_RUNTIME)
	err := instance.InitializeBlock(header)
	require.NoError(t, err)
}

func TestInstance_InherentExtrinsics_LegacyNodeRuntime(t *testing.T) {
	header := &types.Header{
		ParentHash: trie.EmptyHash,
		Number:     big.NewInt(1),
		Digest:     types.Digest{},
	}

	instance := NewTestInstance(t, runtime.LEGACY_NODE_RUNTIME)
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
		in, err := scale.Encode(ext)
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(in)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})
	}
}

func TestInstance_FinalizeBlock_LegacyNodeRuntime(t *testing.T) {
	instance := NewTestInstance(t, runtime.LEGACY_NODE_RUNTIME)

	header := &types.Header{
		ParentHash: trie.EmptyHash,
		Number:     big.NewInt(77),
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

		ret, err := instance.ApplyExtrinsic(in)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})
	}

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)

	res.Number = header.Number

	expected := &types.Header{
		ParentHash: header.ParentHash,
		Number:     big.NewInt(77),
		Digest:     types.Digest{},
	}

	require.Equal(t, expected.ParentHash, res.ParentHash)
	require.Equal(t, expected.Number, res.Number)
	require.Equal(t, expected.Digest, res.Digest)
	require.NotEqual(t, common.Hash{}, res.StateRoot)
	require.NotEqual(t, common.Hash{}, res.ExtrinsicsRoot)
	require.NotEqual(t, trie.EmptyHash, res.StateRoot)
	require.NotEqual(t, trie.EmptyHash, res.ExtrinsicsRoot)
}

func TestInstance_Version_NodeRuntime(t *testing.T) {
	t.Skip() // TODO: currently fails, returns all 0

	expected := runtime.NewVersionData(
		[]byte("node"),
		[]byte("substrate-node"),
		10,
		260,
		0,
		nil,
		1,
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

	require.Equal(t, 12, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}
