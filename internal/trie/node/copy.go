// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

var (
	// DefaultCopySettings contains the following copy settings:
	// - children are NOT deep copied recursively
	// - the HashDigest field is left empty on the copy
	// - the Encoding field is left empty on the copy
	// - the key field is deep copied
	// - the value field is deep copied
	DefaultCopySettings = CopySettings{
		CopyKey:   true,
		CopyValue: true,
	}

	// DeepCopySettings returns the following copy settings:
	// - children are deep copied recursively
	// - the HashDigest field is deep copied
	// - the Encoding field is deep copied
	// - the key field is deep copied
	// - the value field is deep copied
	DeepCopySettings = CopySettings{
		CopyChildren: true,
		CopyCached:   true,
		CopyKey:      true,
		CopyValue:    true,
	}
)

// CopySettings contains settings to configure the deep copy
// of a node.
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
	// CopyKey can be set to true to deep copy the key field of
	// the node. This is useful when false if the key is about to
	// be assigned after the Copy operation, to save a memory operation.
	CopyKey bool
	// CopyValue can be set to true to deep copy the value field of
	// the node. This is useful when false if the value is about to
	// be assigned after the Copy operation, to save a memory operation.
	CopyValue bool
}

// Copy deep copies the node.
// Setting copyChildren to true will deep copy
// children as well.
func (n *Node) Copy(settings CopySettings) *Node {
	cpy := &Node{
		Dirty:       n.Dirty,
		Generation:  n.Generation,
		Descendants: n.Descendants,
	}

	if n.Type() == Branch {
		if settings.CopyChildren {
			// Copy all fields of children if we deep copy children
			childSettings := settings
			childSettings.CopyKey = true
			childSettings.CopyValue = true
			childSettings.CopyCached = true
			cpy.Children = make([]*Node, ChildrenCapacity)
			for i, child := range n.Children {
				if child == nil {
					continue
				}
				cpy.Children[i] = child.Copy(childSettings)
			}
		} else {
			cpy.Children = make([]*Node, ChildrenCapacity)
			copy(cpy.Children, n.Children) // copy node pointers only
		}
	}

	if settings.CopyKey && n.Key != nil {
		cpy.Key = make([]byte, len(n.Key))
		copy(cpy.Key, n.Key)
	}

	// nil and []byte{} are encoded differently, watch out!
	if settings.CopyValue && n.Value != nil {
		cpy.Value = make([]byte, len(n.Value))
		copy(cpy.Value, n.Value)
	}

	if settings.CopyCached {
		if n.HashDigest != nil {
			cpy.HashDigest = make([]byte, len(n.HashDigest))
			copy(cpy.HashDigest, n.HashDigest)
		}

		if n.Encoding != nil {
			cpy.Encoding = make([]byte, len(n.Encoding))
			copy(cpy.Encoding, n.Encoding)
		}
	}

	return cpy
}
