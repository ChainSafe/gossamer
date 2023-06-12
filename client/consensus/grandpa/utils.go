// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// searchKey TODO for reviewer is this ok or do we want a better search algorithm?
func searchKey(k key, changes []PendingChange) int {
	for i, change := range changes {
		changeKey := key{
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
func searchSetChanges(number uint, changes AuthoritySetChanges) (int, bool) {
	for i, change := range changes {
		if change.blockNumber == number {
			return i, true
		} else if number < change.blockNumber {
			return i, false
		}
	}

	return len(changes), false
}
