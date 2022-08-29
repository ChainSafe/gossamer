// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// SetDirty sets the dirty status to true for the node.
func (n *Node) SetDirty() {
	n.Dirty = true
	// A node is marked dirty if its key or value is modified.
	// This means its cached encoding and hash fields are no longer
	// valid. To improve memory usage, we clear these fields.
	n.Encoding = nil
	n.MerkleValue = nil
}

// SetClean sets the dirty status to false for the node.
func (n *Node) SetClean() {
	n.Dirty = false
}
