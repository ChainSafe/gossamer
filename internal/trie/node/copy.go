// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// Copy deep copies the branch.
// Setting copyChildren to true will deep copy
// children as well.
func (b *Branch) Copy(copyChildren bool) Node {
	b.RLock()
	defer b.RUnlock()

	cpy := &Branch{
		Dirty:      b.Dirty,
		Generation: b.Generation,
	}

	if copyChildren {
		for i, child := range b.Children {
			if child == nil {
				continue
			}
			cpy.Children[i] = child.Copy(copyChildren)
		}
	} else {
		cpy.Children = b.Children // copy interface pointers only
	}

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

	if b.Encoding != nil {
		cpy.Encoding = make([]byte, len(b.Encoding))
		copy(cpy.Encoding, b.Encoding)
	}

	return cpy
}

// Copy deep copies the leaf.
func (l *Leaf) Copy(_ bool) Node {
	l.RLock()
	defer l.RUnlock()

	l.encodingMu.RLock()
	defer l.encodingMu.RUnlock()

	cpy := &Leaf{
		Dirty:      l.Dirty,
		Generation: l.Generation,
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

	if l.Encoding != nil {
		cpy.Encoding = make([]byte, len(l.Encoding))
		copy(cpy.Encoding, l.Encoding)
	}

	return cpy
}
