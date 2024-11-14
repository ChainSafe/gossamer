// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

// Storage key.
type StorageKey []byte

// Storage value. Value can be nil
type StorageValue []byte

// Storage key and value.
type StorageKeyValue struct {
	StorageKey
	StorageValue
}

// In memory array of storage values.
type StorageCollection []StorageKeyValue
