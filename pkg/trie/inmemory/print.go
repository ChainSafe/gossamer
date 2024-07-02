// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

// String returns the trie stringified through pre-order traversal
func (t *InMemoryTrie) String() string {
	if t.root == nil {
		return "empty"
	}

	return t.root.String()
}
