// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	memorydb "github.com/ChainSafe/gossamer/internal/memory-db"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

func NewMemoryDB() *memorydb.MemoryDB[
	hash.H256, runtime.BlakeTwo256, hash.H256, memorydb.HashKey[hash.H256]] {
	db := memorydb.NewMemoryDB[
		hash.H256, runtime.BlakeTwo256, hash.H256, memorydb.HashKey[hash.H256],
	]([]byte{0})
	return &db
}
