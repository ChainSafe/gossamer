// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
)

var (
	_ hashdb.HashDB[hash.H256] = &KeyspacedDB[hash.H256]{}
)
