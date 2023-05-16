// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import "fmt"

// SearchKey TODO for reviewer, this can be done by slices.BinarySearch however since it is a tuple being compared,
// it's unclear to me what value to sort on
func SearchKey(key Key, changes []PendingChange) (int, error) {
	for i, change := range changes {
		changeKey := Key{
			effectiveNumber:   change.EffectiveNumber(),
			signalBlockNumber: change.canonHeight,
		}
		if key.Equals(changeKey) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("key not found")
}
