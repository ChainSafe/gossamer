// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import "github.com/ChainSafe/gossamer/internal/saturating"

// NodeHeader without content
type nodeKind uint

const (
	nodeKindLeaf nodeKind = iota
	nodeKindBranchNoValue
	nodeKindBranchWithValue
	nodeKindHashedValueLeaf
	nodeKindHashedValueBranch
)

// Returns an iterator over encoded bytes for node header and size.
// Size encoding allows unlimited, length inefficient, representation, but is bounded to 16 bit maximum value to avoid
// possible DOS.
func sizeAndPrefixIterator(size uint, prefix uint8, prefixMask int) []byte {
	maxValue := uint8(255) >> prefixMask
	l1 := saturating.Sub(maxValue, 1)
	if size < uint(l1) {
		l1 = uint8(size)
	}

	var firstByte uint8
	var rem uint
	if size == uint(l1) {
		firstByte = prefix + l1
		rem = 0
	} else {
		firstByte = prefix + maxValue
		rem = size - uint(l1)
	}
	nextBytes := make([]byte, 0)
	for {
		if rem > 0 {
			if rem < 256 {
				result := rem - 1
				rem = 0
				nextBytes = append(nextBytes, uint8(result))
			} else {
				rem = saturating.Sub(rem, 255)
				nextBytes = append(nextBytes, 255)
			}
		} else {
			break
		}
	}

	return append([]byte{firstByte}, nextBytes...)
}
