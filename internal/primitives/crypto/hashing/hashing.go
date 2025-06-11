// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package hashing

import (
	"encoding/binary"

	"github.com/OneOfOne/xxhash"
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

// / Do a XX 128-bit hash and return result.
func Twox128(data []byte) [16]byte {
	// compute xxHash64 twice with seeds 0 and 1 applied on given byte array
	h0 := xxhash.NewS64(0) // create xxHash with 0 seed
	_, err := h0.Write(data)
	if err != nil {
		panic(err)
	}
	res0 := h0.Sum64()
	hash0 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash0, res0)

	h1 := xxhash.NewS64(1) // create xxHash with 1 seed
	_, err = h1.Write(data)
	if err != nil {
		panic(err)
	}
	res1 := h1.Sum64()
	hash1 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash1, res1)

	return [16]byte{
		hash0[0], hash0[1], hash0[2], hash0[3],
		hash0[4], hash0[5], hash0[6], hash0[7],
		hash1[0], hash1[1], hash1[2], hash1[3],
		hash1[4], hash1[5], hash1[6], hash1[7],
	}
}
