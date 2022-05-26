// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/trie"
)

var (
	ErrKeyNotFoundInProofTrie = errors.New("key not found in proof trie")
	ErrValueMismatchProofTrie = errors.New("value found in proof trie does not match")
)

// Verify verifies a given key and value belongs to the trie by creating
// a proof trie based on the encoded proof nodes given. The order of proofs is ignored.
func Verify(encodedProofNodes [][]byte, rootHash, key, value []byte) (ok bool, err error) {
	proofTrie := trie.NewEmptyTrie()
	err = proofTrie.LoadFromProof(encodedProofNodes, rootHash)
	if err != nil {
		return false, fmt.Errorf("cannot build trie from proof encoded nodes: %w", err)
	}

	proofTrieValue := proofTrie.Get(key)
	if proofTrieValue == nil {
		return false, fmt.Errorf("%w: %s", ErrKeyNotFoundInProofTrie, bytesToString(key))
	}

	// compare the value only if the caller pass a non empty value
	if len(value) > 0 && !bytes.Equal(value, proofTrieValue) {
		return false, fmt.Errorf("%w: expected value %s but got value %s from proof trie",
			ErrValueMismatchProofTrie, bytesToString(value), bytesToString(proofTrieValue))
	}

	return true, nil
}

func bytesToString(b []byte) (s string) {
	switch {
	case b == nil:
		return "nil"
	case len(b) <= 20:
		return fmt.Sprintf("0x%x", b)
	default:
		return fmt.Sprintf("0x%x...%x", b[:8], b[len(b)-8:])
	}
}
