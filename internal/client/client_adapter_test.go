// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package client

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
)

func TestBlockStateImplemented(t *testing.T) {
	var _ state.BlockState = &ClientAdapter[hash.H256,
		runtime.BlakeTwo256, uint64, runtime.OpaqueExtrinsic, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]{}
}
