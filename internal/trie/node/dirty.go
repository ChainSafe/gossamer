// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// SetDirty sets the dirty status to true for the node.
func (n *Node) SetDirty() {
	n.Dirty = true
	// A node is marked dirty if its partial key or storage value is modified.
	// This means its Merkle value field is no longer valid.
	n.MerkleValue = nil
}

// SetClean sets the dirty status to false for the node.
func (n *Node) SetClean() {
	n.Dirty = false
}
