// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import "bytes"

// SubValueEqual returns true if the node subvalue is equal to the
// subvalue given as argument. In particular, it returns false
// if one subvalue is nil and the other subvalue is the empty slice.
func (n Node) SubValueEqual(subValue []byte) (equal bool) {
	if len(subValue) == 0 && len(n.SubValue) == 0 {
		return (subValue == nil && n.SubValue == nil) ||
			(subValue != nil && n.SubValue != nil)
	}
	return bytes.Equal(n.SubValue, subValue)
}
