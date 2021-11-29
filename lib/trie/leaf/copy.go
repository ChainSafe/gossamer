// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

import "github.com/ChainSafe/gossamer/lib/trie/node"

// Copy deep copies the leaf.
func (l *Leaf) Copy() node.Node {
	l.RLock()
	defer l.RUnlock()

	l.encodingMu.RLock()
	defer l.encodingMu.RUnlock()

	cpy := &Leaf{
		Key:        make([]byte, len(l.Key)),
		Value:      make([]byte, len(l.Value)),
		Dirty:      l.Dirty,
		Hash:       make([]byte, len(l.Hash)),
		Encoding:   make([]byte, len(l.Encoding)),
		Generation: l.Generation,
	}
	copy(cpy.Key, l.Key)
	copy(cpy.Value, l.Value)
	copy(cpy.Hash, l.Hash)
	copy(cpy.Encoding, l.Encoding)
	return cpy
}
