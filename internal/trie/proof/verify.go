// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/trie"
)

// Pair holds the key and value to check while verifying the proof
type Pair struct{ Key, Value []byte }

// Verify ensure a given key is inside a proof by creating a proof trie based on the proof slice
// this function ignores the order of proofs
func Verify(proof [][]byte, root []byte, items []Pair) (bool, error) {
	set := make(map[string]struct{}, len(items))

	// check for duplicate keys
	for _, item := range items {
		hexKey := hex.EncodeToString(item.Key)
		if _, ok := set[hexKey]; ok {
			return false, ErrDuplicateKeys
		}
		set[hexKey] = struct{}{}
	}

	proofTrie := trie.NewEmptyTrie()
	if err := proofTrie.LoadFromProof(proof, root); err != nil {
		return false, fmt.Errorf("%w: %s", ErrLoadFromProof, err)
	}

	for _, item := range items {
		recValue := proofTrie.Get(item.Key)
		if recValue == nil {
			return false, ErrKeyNotFound
		}
		// here we need to compare value only if the caller pass the value
		if len(item.Value) > 0 && !bytes.Equal(item.Value, recValue) {
			return false, ErrValueNotFound
		}
	}

	return true, nil
}
