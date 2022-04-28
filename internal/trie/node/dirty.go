// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// SetDirty sets the dirty status to the node.
func (n *Node) SetDirty(dirty bool) {
	n.Dirty = dirty
	if dirty {
		// A node is marked dirty if its key or value is modified.
		// This means its cached encoding and hash fields are no longer
		// valid. To improve memory usage, we clear these fields.
		n.Encoding = nil
		n.HashDigest = nil
	}
}
