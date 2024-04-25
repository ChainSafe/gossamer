// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package hashing

import (
	"golang.org/x/crypto/blake2b"
)

// Do a Blake2 256-bit hash and return result.
func Blake2_256(data []byte) [32]byte {
	h, err := blake2b.New256(nil)
	if err != nil {
		panic(err)
	}
	_, err = h.Write(data)
	if err != nil {
		panic(err)
	}
	encoded := h.Sum(nil)
	var arr [32]byte
	copy(arr[:], encoded)
	return arr
}
