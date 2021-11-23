// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
)

// findAndRecord search for a desired key recording all the nodes in the path including the desired node
func findAndRecord(t *Trie, key []byte, recorder *recorder) error {
	return find(t.root, key, recorder)
}

func find(parent Node, key []byte, recorder *recorder) error {
	enc, hash, err := parent.EncodeAndHash()
	if err != nil {
		return err
	}

	recorder.record(hash, enc)

	b, ok := parent.(*Branch)
	if !ok {
		return nil
	}

	length := lenCommonPrefix(b.key, key)

	// found the value at this node
	if bytes.Equal(b.key, key) || len(key) == 0 {
		return nil
	}

	// did not find value
	if bytes.Equal(b.key[:length], key) && len(key) < len(b.key) {
		return nil
	}

	return find(b.children[key[length]], key[length+1:], recorder)
}
