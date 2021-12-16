// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// Copy deep copies the branch.
func (b *Branch) Copy() Node {
	b.RLock()
	defer b.RUnlock()

	cpy := &Branch{
		Children:   b.Children, // copy interface pointers
		dirty:      b.dirty,
		generation: b.generation,
	}
	copy(cpy.Key, b.Key)

	if b.Key != nil {
		cpy.Key = make([]byte, len(b.Key))
		copy(cpy.Key, b.Key)
	}

	// nil and []byte{} are encoded differently, watch out!
	if b.Value != nil {
		cpy.Value = make([]byte, len(b.Value))
		copy(cpy.Value, b.Value)
	}

	if b.hashDigest != nil {
		cpy.hashDigest = make([]byte, len(b.hashDigest))
		copy(cpy.hashDigest, b.hashDigest)
	}

	if b.encoding != nil {
		cpy.encoding = make([]byte, len(b.encoding))
		copy(cpy.encoding, b.encoding)
	}

	return cpy
}

// Copy deep copies the leaf.
func (l *Leaf) Copy() Node {
	l.RLock()
	defer l.RUnlock()

	l.encodingMu.RLock()
	defer l.encodingMu.RUnlock()

	cpy := &Leaf{
		dirty:      l.dirty,
		generation: l.generation,
	}

	if l.Key != nil {
		cpy.Key = make([]byte, len(l.Key))
		copy(cpy.Key, l.Key)
	}

	// nil and []byte{} are encoded differently, watch out!
	if l.Value != nil {
		cpy.Value = make([]byte, len(l.Value))
		copy(cpy.Value, l.Value)
	}

	if l.hashDigest != nil {
		cpy.hashDigest = make([]byte, len(l.hashDigest))
		copy(cpy.hashDigest, l.hashDigest)
	}

	if l.encoding != nil {
		cpy.encoding = make([]byte, len(l.encoding))
		copy(cpy.encoding, l.encoding)
	}

	return cpy
}
