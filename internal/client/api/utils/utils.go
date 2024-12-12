// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// Hash and parent hash of a block
type HashParent[H runtime.Hash] struct {
	Hash   H
	Parent H
}

// Returns a function for checking block ancestry, the returned function will
// return true if the given hash (second parameter) is a descendent of the
// base (first parameter). If current is defined, it should
// represent the current block hash and its parent hash. if current is given, the
// function that is returned will assume that current.Hash isn't part of the local DB
// yet, and all searches in the DB will instead reference the parent.
func IsDescendantOf[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]](
	client blockchain.Backend[H, N, Header], current *HashParent[H],
) func(a H, b H) (bool, error) {
	return func(base H, hash H) (bool, error) {
		if base == hash {
			return false, nil
		}

		if current != nil {
			currentHash := current.Hash
			currentParentHash := current.Parent
			if base == currentHash {
				return false, nil
			}
			if hash == currentHash {
				if base == currentParentHash {
					return true, nil
				} else {
					hash = currentParentHash
				}
			}
		}

		ancestor, err := blockchain.LowestCommonAncestor(client, hash, base)
		if err != nil {
			return false, err
		}

		return ancestor.Hash == base, nil
	}
}
