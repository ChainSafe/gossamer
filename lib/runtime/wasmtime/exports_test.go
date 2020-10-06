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
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

func TestConcurrentRuntimeCalls(t *testing.T) {
	instance := NewTestInstance(t, runtime.NODE_RUNTIME)

	// execute 2 concurrent calls to the runtime
	go func() {
		_, _ = instance.exec(runtime.CoreVersion, []byte{})
	}()
	go func() {
		_, _ = instance.exec(runtime.CoreVersion, []byte{})
	}()
}

func TestInstance_Version_NodeRuntime(t *testing.T) {
	// https://github.com/paritytech/substrate/blob/7b1d822446982013fa5b7ad5caff35ca84f8b7d0/core/test-runtime/src/lib.rs#L73
	expected := &runtime.Version{
		Spec_name:         []byte("node"),
		Impl_name:         []byte("substrate-node"),
		Authoring_version: 10,
		Spec_version:      193,
		Impl_version:      193,
	}

	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	version, err := instance.Version()
	require.NoError(t, err)

	t.Logf("Spec_name: %s\n", version.RuntimeVersion.Spec_name)
	t.Logf("Impl_name: %s\n", version.RuntimeVersion.Impl_name)
	t.Logf("Authoring_version: %d\n", version.RuntimeVersion.Authoring_version)
	t.Logf("Spec_version: %d\n", version.RuntimeVersion.Spec_version)
	t.Logf("Impl_version: %d\n", version.RuntimeVersion.Impl_version)

	require.Equal(t, expected, version.RuntimeVersion)
}

func TestInstance_BabeConfiguration_NodeRuntime(t *testing.T) {
	expected := &types.BabeConfiguration{
		SlotDuration:       3000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: nil,
		SecondarySlots:     true,
	}

	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	babeCfg, err := instance.BabeConfiguration()
	require.NoError(t, err)
	require.Equal(t, expected, babeCfg)
}

func TestInstance_GrandpaAuthorities_NodeRuntime(t *testing.T) {
	expected := []*types.Authority{}

	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	res, err := instance.GrandpaAuthorities()
	require.NoError(t, err)
	require.Equal(t, expected, res)
}

func TestInstance_InitializeBlock_NodeRuntime(t *testing.T) {
	header := &types.Header{
		ParentHash: trie.EmptyHash,
		Number:     big.NewInt(1),
		Digest:     [][]byte{},
	}

	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	err := instance.InitializeBlock(header)
	require.NoError(t, err)
}

func TestInstance_FinalizeBlock_NodeRuntime(t *testing.T) {
	// TODO: need to add inherents before calling finalize_block (see babe/inherents_test.go)
	// need to move inherents to a different package for use with BABE and runtime

	instance := NewTestInstance(t, runtime.NODE_RUNTIME)

	header := &types.Header{
		ParentHash: trie.EmptyHash,
		Number:     big.NewInt(77),
		//StateRoot: trie.EmptyHash,
		//ExtrinsicsRoot: trie.EmptyHash,
		Digest: [][]byte{},
	}

	err := instance.InitializeBlock(header)
	require.NoError(t, err)

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)

	res.Number = header.Number

	expected := &types.Header{
		StateRoot:      trie.EmptyHash,
		ExtrinsicsRoot: trie.EmptyHash,
		Number:         big.NewInt(77),
		Digest:         [][]byte{},
	}

	res.Hash()
	expected.Hash()

	require.Equal(t, expected, res)
}
