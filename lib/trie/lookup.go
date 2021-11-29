// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"

	"github.com/ChainSafe/gossamer/lib/trie/branch"
	"github.com/ChainSafe/gossamer/lib/trie/node"
)

// findAndRecord search for a desired key recording all the nodes in the path including the desired node
func findAndRecord(t *Trie, key []byte, recorder *recorder) error {
	return find(t.root, key, recorder)
}

func find(parent node.Node, key []byte, recorder *recorder) error {
	enc, hash, err := parent.EncodeAndHash()
	if err != nil {
		return err
	}

	recorder.record(hash, enc)

	b, ok := parent.(*branch.Branch)
	if !ok {
		return nil
	}

	length := lenCommonPrefix(b.Key, key)

	// found the value at this node
	if bytes.Equal(b.Key, key) || len(key) == 0 {
		return nil
	}

	// did not find value
	if bytes.Equal(b.Key[:length], key) && len(key) < len(b.Key) {
		return nil
	}

	return find(b.Children[key[length]], key[length+1:], recorder)
}
