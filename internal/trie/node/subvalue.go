// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import "bytes"

// StorageValueEqual returns true if the node storage value is equal to the
// storage value given as argument. In particular, it returns false
// if one storage value is nil and the other storage value is the empty slice.
func (n *Node) StorageValueEqual(stoageValue []byte) (equal bool) {
	if len(stoageValue) == 0 && len(n.StorageValue) == 0 {
		return (stoageValue == nil && n.StorageValue == nil) ||
			(stoageValue != nil && n.StorageValue != nil)
	}
	return bytes.Equal(n.StorageValue, stoageValue)
}
