// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package hash

// Hash type
type Hash interface {
	comparable
	// Bytes returns a byte slice representation of Hash
	Bytes() []byte
	// Length return the byte length of the hash
	Length() int
}

// Hasher is an interface around hashing
type Hasher[H Hash] interface {
	// Produce the hash of some byte slice.
	Hash(s []byte) H
}
