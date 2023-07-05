// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

// searchKey TODO for reviewer is this ok or do we want a better search algorithm?
func searchKey[H comparable, N constraints.Unsigned](k key[N], changes []PendingChange[H, N]) int {
	for i, change := range changes {
		changeKey := key[N]{
			effectiveNumber:   change.EffectiveNumber(),
			signalBlockNumber: change.canonHeight,
		}
		if (k.effectiveNumber == k.effectiveNumber &&
			k.signalBlockNumber == k.signalBlockNumber) || k.effectiveNumber < changeKey.effectiveNumber {
			return i
		}
	}

	return len(changes)
}

// returns an index representing either the found element or the index to insert the given element, and a bool
// indicating if the given element was found
func searchSetChanges[N constraints.Unsigned](number N, changes AuthoritySetChanges[N]) (int, bool) {
	return slices.BinarySearchFunc(
		changes,
		number,
		func(a authorityChange[N], b N) int {
			switch {
			case a.blockNumber == b:
				return 0
			case a.blockNumber > b:
				return 1
			case a.blockNumber < b:
				return -1
			default:
				panic("huh?")
			}
		},
	)
}

func bytesToHash(b []byte) Hash {
	var h Hash
	h.setBytes(b)
	return h
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) setBytes(b []byte) { //skipcq: GO-W1029
	if len(b) > len(h) {
		b = b[len(b)-32:]
	}

	copy(h[32-len(b):], b)
}

// String returns the hex string for the hash
func (h Hash) String() string { //skipcq: GO-W1029
	return fmt.Sprintf("0x%x", h[:])
}
