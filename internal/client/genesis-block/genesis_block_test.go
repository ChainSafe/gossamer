// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesisblock

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/executor"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	primitives_storage "github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
	version "github.com/ChainSafe/gossamer/internal/primitives/version"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
	gomock "go.uber.org/mock/gomock"
)

type Hasher = runtime.BlakeTwo256
type Hash = hash.H256
type Executor = executor.RuntimeVersionOf

func TestResolveStateVersionFromWasmOk(t *testing.T) {
	ctrl := gomock.NewController(t)

	var storagetTopData btree.Map[string, []byte]
	storagetTopData.Set(string(keys.Code), []byte{0x01, 0x02, 0x03})

	storage := primitives_storage.Storage{
		Top: storagetTopData,
	}

	executorMock := NewMockRuntimeVersionOf(ctrl)
	executorMock.EXPECT().RuntimeVersion(gomock.Any(), gomock.Any()).Return(version.RuntimeVersion{
		SystemVersion: 1,
	}, nil)

	stateVersion, err := ResolveStateVersionFromWasm[Hasher, Hash, Executor](storage, executorMock)

	require.NoError(t, err)
	require.Equal(t, primitives_storage.StateVersionV1, stateVersion)
}

func TestResolveStateVersionFromWasmMissingCode(t *testing.T) {
	var storagetTopData btree.Map[string, []byte]

	storage := primitives_storage.Storage{
		Top: storagetTopData,
	}

	stateVersion, err := ResolveStateVersionFromWasm[Hasher, Hash, Executor](storage, nil)

	require.ErrorIs(t, err, blockchain.ErrVersionInvalid)
	require.Equal(t, primitives_storage.NoStateVersion, stateVersion)
}

func TestResolveStateVersionFromWasmInvalidRuntimeVersion(t *testing.T) {
	ctrl := gomock.NewController(t)

	var storagetTopData btree.Map[string, []byte]
	storagetTopData.Set(string(keys.Code), []byte{0x01, 0x02, 0x03})

	storage := primitives_storage.Storage{
		Top: storagetTopData,
	}

	executorMock := NewMockRuntimeVersionOf(ctrl)
	executorMock.EXPECT().RuntimeVersion(gomock.Any(), gomock.Any()).Return(version.RuntimeVersion{
		SystemVersion: 1,
	}, blockchain.ErrVersionInvalid)

	stateVersion, err := ResolveStateVersionFromWasm[Hasher, Hash, Executor](storage, executorMock)

	require.ErrorIs(t, err, blockchain.ErrVersionInvalid)
	require.Equal(t, primitives_storage.NoStateVersion, stateVersion)
}
