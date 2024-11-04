// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package hashing

import (
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

// BlakeTwo256 returns a Blake2 256-bit hash of the input data
func BlakeTwo256(data []byte) [32]byte {
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

// Keccak256 returns the keccak256 hash of the input data
func Keccak256(data []byte) [32]byte {
	h := sha3.NewLegacyKeccak256()
	_, err := h.Write(data)
	if err != nil {
		panic(err)
	}

	hash := h.Sum(nil)
	var buf = [32]byte{}
	copy(buf[:], hash)
	return buf
}
