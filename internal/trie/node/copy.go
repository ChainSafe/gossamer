// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

var (
	// DefaultCopySettings contains the following copy settings:
	// - children are NOT deep copied recursively
	// - the Merkle value field is left empty on the copy
	// - the partial key field is deep copied
	// - the storage value field is deep copied
	DefaultCopySettings = CopySettings{
		CopyPartialKey:   true,
		CopyStorageValue: true,
	}

	// DeepCopySettings returns the following copy settings:
	// - children are deep copied recursively
	// - the Merkle value field is deep copied
	// - the partial key field is deep copied
	// - the storage value field is deep copied
	DeepCopySettings = CopySettings{
		CopyChildren:     true,
		CopyMerkleValue:  true,
		CopyPartialKey:   true,
		CopyStorageValue: true,
	}
)

// CopySettings contains settings to configure the deep copy
// of a node.
type CopySettings struct {
	// CopyChildren can be set to true to recursively deep copy the eventual
	// children of the node. This is false by default and should only be used
	// in tests since it is quite inefficient.
	CopyChildren bool
	// CopyMerkleValue can be set to true to deep copy the Merkle value
	// field on the current node copied.
	// This is false by default because in production, a node is copied
	// when it is about to be mutated, hence making its cached Merkle value
	// field no longer valid.
	CopyMerkleValue bool
	// CopyPartialKey can be set to true to deep copy the partial key field of
	// the node. This is useful when false if the partial key is about to
	// be assigned after the Copy operation, to save a memory operation.
	CopyPartialKey bool
	// CopyStorageValue can be set to true to deep copy the storage value field of
	// the node. This is useful when false if the storage value is about to
	// be assigned after the Copy operation, to save a memory operation.
	CopyStorageValue bool
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

	if n.Kind() == Branch {
		if settings.CopyChildren {
			// Copy all fields of children if we deep copy children
			childSettings := settings
			childSettings.CopyPartialKey = true
			childSettings.CopyStorageValue = true
			childSettings.CopyMerkleValue = true
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

	if settings.CopyPartialKey && n.PartialKey != nil {
		cpy.PartialKey = make([]byte, len(n.PartialKey))
		copy(cpy.PartialKey, n.PartialKey)
	}

	// nil and []byte{} storage values for branches result in a different node encoding,
	// so we ensure to keep the `nil` storage value.
	if settings.CopyStorageValue && n.StorageValue != nil {
		cpy.StorageValue = make([]byte, len(n.StorageValue))
		copy(cpy.StorageValue, n.StorageValue)
		cpy.HashedValue = n.HashedValue
	}

	if settings.CopyMerkleValue {
		if n.MerkleValue != nil {
			cpy.MerkleValue = make([]byte, len(n.MerkleValue))
			copy(cpy.MerkleValue, n.MerkleValue)
		}
	}

	return cpy
}
