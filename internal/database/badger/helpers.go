// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package badger

func makePrefixedKey(prefix, key []byte) (prefixedKey []byte) {
	// WARNING: Do not use:
	// return append(prefix, key...)
	// since the prefix might have a capacity larger than its length,
	// and that would produce data corruption on prefixed keys pointing
	// to the prefix underlying memory array.
	prefixedKey = make([]byte, 0, len(prefix)+len(key))
	prefixedKey = append(prefixedKey, prefix...)
	prefixedKey = append(prefixedKey, key...)
	return prefixedKey
}
