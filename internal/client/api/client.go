// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package api

// List of operations to be performed on storage aux data.
// Key is the encoded data key.
// Value is the encoded optional data to write.
// If Value is nil, the key and the associated data are deleted from storage.
type AuxDataOperation struct {
	Key  []byte
	Data []byte
}
type AuxDataOperations []AuxDataOperation
