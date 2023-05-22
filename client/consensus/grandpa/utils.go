// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// SearchKey TODO for reviewer, this can be done by slices.BinarySearch however since it is a tuple being compared,
// it's unclear to me what value to sort on
func SearchKey(key Key, changes []PendingChange) int {
	for i, change := range changes {
		changeKey := Key{
			effectiveNumber:   change.EffectiveNumber(),
			signalBlockNumber: change.canonHeight,
		}
		if key.Equals(changeKey) {
			return i
		}
	}
	//return 0, fmt.Errorf("key not found")

	// DOnt return error, return idex where key could be inserted to retain sorted order
	// TODO ask Tim or eclesio about this logic on where to insert, for now insert at end
	return len(changes)
}
