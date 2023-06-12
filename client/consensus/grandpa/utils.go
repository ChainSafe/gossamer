// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// SearchKey TODO for reviewer, this can be done by slices.BinarySearch however since it is a tuple being compared,
// it's unclear to me what value to sort on
func SearchKey(k key, changes []PendingChange) int {
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

func SearchSetChanges(number uint, changes AuthoritySetChanges) int {
	for i, change := range changes {
		if change.blockNumber == number || number < change.blockNumber {
			return i
		}
	}

	return len(changes)
}

func SearchSetChangesForIter(number uint, changes AuthoritySetChanges) (int, bool) {
	for i, change := range changes {
		if change.blockNumber == number {
			return i, true
		} else if number < change.blockNumber {
			return i, false
		}
	}

	return len(changes), false
}
