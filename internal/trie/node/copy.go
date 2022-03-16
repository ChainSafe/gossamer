// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// CopySettings contains settings to configure the deep copy
// of a node. By default, it:
// - does not deep copy children recrursively
// - does not copy cached fields HashDigest and Encoding
// - deep copies the key field
// - deep copies the value field
type CopySettings struct {
	// CopyChildren can be set to true to recursively deep copy the eventual
	// children of the node. This is false by default and should only be used
	// in tests since it is quite inefficient.
	CopyChildren bool
	// CopyCached can be set to true to deep copy the cached digest
	// and encoding fields on the current node copied.
	// This is false by default because in production, a node is copied
	// when it is about to be mutated, hence making its cached fields
	// no longer valid.
	CopyCached bool
	// LeaveKeyEmpty can be set to true to not deep copy the key field of
	// the node. This is useful if the key is about to be assigned after the
	// Copy operation, to save a memory operation.
	LeaveKeyEmpty bool
	// LeaveValueEmpty can be set to true to not deep copy the value field of
	// the node. This is useful if the value is about to be assigned after the
	// Copy operation, to save a memory operation.
	LeaveValueEmpty bool
}

// Copy deep copies the branch.
// Setting copyChildren to true will deep copy
// children as well.
func (b *Branch) Copy(settings CopySettings) Node {
	cpy := &Branch{
		Dirty:      b.Dirty,
		Generation: b.Generation,
	}

	if settings.CopyChildren {
		for i, child := range b.Children {
			if child == nil {
				continue
			}
			cpy.Children[i] = child.Copy(settings)
		}
	} else {
		cpy.Children = b.Children // copy interface pointers only
	}

	if !settings.LeaveKeyEmpty && b.Key != nil {
		cpy.Key = make([]byte, len(b.Key))
		copy(cpy.Key, b.Key)
	}

	// nil and []byte{} are encoded differently, watch out!
	if !settings.LeaveValueEmpty && b.Value != nil {
		cpy.Value = make([]byte, len(b.Value))
		copy(cpy.Value, b.Value)
	}

	if settings.CopyCached {
		if b.HashDigest != nil {
			cpy.HashDigest = make([]byte, len(b.HashDigest))
			copy(cpy.HashDigest, b.HashDigest)
		}

		if b.Encoding != nil {
			cpy.Encoding = make([]byte, len(b.Encoding))
			copy(cpy.Encoding, b.Encoding)
		}
	}

	return cpy
}

// Copy deep copies the leaf.
func (l *Leaf) Copy(settings CopySettings) Node {
	cpy := &Leaf{
		Dirty:      l.Dirty,
		Generation: l.Generation,
	}

	if !settings.LeaveKeyEmpty && l.Key != nil {
		cpy.Key = make([]byte, len(l.Key))
		copy(cpy.Key, l.Key)
	}

	// nil and []byte{} are encoded differently, watch out!
	if !settings.LeaveValueEmpty && l.Value != nil {
		cpy.Value = make([]byte, len(l.Value))
		copy(cpy.Value, l.Value)
	}

	if settings.CopyCached {
		if l.HashDigest != nil {
			cpy.HashDigest = make([]byte, len(l.HashDigest))
			copy(cpy.HashDigest, l.HashDigest)
		}

		if l.Encoding != nil {
			cpy.Encoding = make([]byte, len(l.Encoding))
			copy(cpy.Encoding, l.Encoding)
		}
	}

	return cpy
}
