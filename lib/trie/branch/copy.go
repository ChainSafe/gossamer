// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import "github.com/ChainSafe/gossamer/lib/trie/node"

// Copy deep copies the branch.
func (b *Branch) Copy() node.Node {
	b.RLock()
	defer b.RUnlock()

	cpy := &Branch{
		Key:        make([]byte, len(b.Key)),
		Children:   b.Children, // copy interface pointers
		Value:      nil,
		Dirty:      b.Dirty,
		Hash:       make([]byte, len(b.Hash)),
		Encoding:   make([]byte, len(b.Encoding)),
		Generation: b.Generation,
	}
	copy(cpy.Key, b.Key)

	// nil and []byte{} are encoded differently, watch out!
	if b.Value != nil {
		cpy.Value = make([]byte, len(b.Value))
		copy(cpy.Value, b.Value)
	}

	copy(cpy.Hash, b.Hash)
	copy(cpy.Encoding, b.Encoding)
	return cpy
}
