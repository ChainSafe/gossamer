// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package chainspec

import (
	"errors"

	"github.com/ChainSafe/gossamer/internal/client/executor"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/basic"
	primitives_storage "github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

func ResolveStateVersionFromWasm[Hasher runtime.Hasher[H], H runtime.Hash, E executor.RuntimeVersionOf](
	storage primitives_storage.Storage,
	executor E,
) (primitives_storage.StateVersion, error) {
	wasm, has := storage.Top.Get(string(keys.Code))
	if !has {
		return primitives_storage.NoStateVersion,
			errors.New("runtime missing from initial storage, could not read state version")
	}

	ext := basic.NewBasicExternalities() // Just to read runtime version
	codeFetcher := core.NewWrappedRuntimeCode(wasm)

	hasher := *new(Hasher)
	runtimeCode := core.RuntimeCode{
		CodeFetcher: codeFetcher,
		HeapPages:   nil,
		Hash:        scale.MustMarshal(hasher.Hash(wasm)),
	}

	runtimeVersion, err := executor.RuntimeVersion(ext, runtimeCode)
	if err != nil {
		return primitives_storage.NoStateVersion, blockchain.ErrVersionInvalid
	}

	return runtimeVersion.StateVersion(), nil
}
