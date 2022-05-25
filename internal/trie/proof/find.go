// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/trie"
)

// findAndRecord search for a desired key recording all the nodes in the path including the desired node
func findAndRecord(t *trie.Trie, key []byte, recorder *recorder) error {
	return find(t.RootNode(), key, recorder, true)
}

func find(parent *node.Node, key []byte, recorder *recorder, isCurrentRoot bool) error {
	enc, hash, err := parent.EncodeAndHash(isCurrentRoot)
	if err != nil {
		return err
	}

	recorder.record(hash, enc)

	if parent.Type() != node.Branch {
		return nil
	}

	branch := parent
	length := lenCommonPrefix(branch.Key, key)

	// found the value at this node
	if bytes.Equal(branch.Key, key) || len(key) == 0 {
		return nil
	}

	// did not find value
	if bytes.Equal(branch.Key[:length], key) && len(key) < len(branch.Key) {
		return nil
	}

	return find(branch.Children[key[length]], key[length+1:], recorder, false)
}

// lenCommonPrefix returns the length of the
// common prefix between two byte slices.
func lenCommonPrefix(a, b []byte) (length int) {
	min := len(a)
	if len(b) < min {
		min = len(b)
	}

	for length = 0; length < min; length++ {
		if a[length] != b[length] {
			break
		}
	}

	return length
}
