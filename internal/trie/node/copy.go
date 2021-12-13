// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// Copy deep copies the branch.
func (b *Branch) Copy() Node {
	b.RLock()
	defer b.RUnlock()

	cpy := &Branch{
		Key:        make([]byte, len(b.Key)),
		Children:   b.Children, // copy interface pointers
		Value:      nil,
		dirty:      b.dirty,
		hashDigest: make([]byte, len(b.hashDigest)),
		Encoding:   make([]byte, len(b.Encoding)),
		generation: b.generation,
	}
	copy(cpy.Key, b.Key)

	// nil and []byte{} are encoded differently, watch out!
	if b.Value != nil {
		cpy.Value = make([]byte, len(b.Value))
		copy(cpy.Value, b.Value)
	}

	copy(cpy.hashDigest, b.hashDigest)
	copy(cpy.Encoding, b.Encoding)
	return cpy
}

// Copy deep copies the leaf.
func (l *Leaf) Copy() Node {
	l.RLock()
	defer l.RUnlock()

	l.encodingMu.RLock()
	defer l.encodingMu.RUnlock()

	cpy := &Leaf{
		Key:        make([]byte, len(l.Key)),
		Value:      make([]byte, len(l.Value)),
		dirty:      l.dirty,
		hashDigest: make([]byte, len(l.hashDigest)),
		Encoding:   make([]byte, len(l.Encoding)),
		generation: l.generation,
	}
	copy(cpy.Key, l.Key)
	copy(cpy.Value, l.Value)
	copy(cpy.hashDigest, l.hashDigest)
	copy(cpy.Encoding, l.Encoding)
	return cpy
}
