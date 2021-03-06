// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/OneOfOne/xxhash"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

// Blake2b128 returns the 128-bit blake2b hash of the input data
func Blake2b128(in []byte) ([]byte, error) {
	h, err := blake2b.New(16, nil)
	if err != nil {
		return nil, err
	}

	_, err = h.Write(in)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// Blake2bHash returns the 256-bit blake2b hash of the input data
func Blake2bHash(in []byte) (Hash, error) {
	h, err := blake2b.New256(nil)
	if err != nil {
		return [32]byte{}, err
	}

	_, err = h.Write(in)
	if err != nil {
		return [32]byte{}, err
	}

	hash := h.Sum(nil)
	var buf = [32]byte{}
	copy(buf[:], hash)
	return buf, nil
}

// MustBlake2bHash returns the 256-bit blake2b hash of the input data. It panics if it fails to hash.
func MustBlake2bHash(in []byte) Hash {
	hash, err := Blake2bHash(in)
	if err != nil {
		panic(err)
	}

	return hash
}

// Keccak256 returns the keccak256 hash of the input data
func Keccak256(in []byte) (Hash, error) {
	h := sha3.NewLegacyKeccak256()

	_, err := h.Write(in)
	if err != nil {
		return [32]byte{}, err
	}

	hash := h.Sum(nil)
	var buf = [32]byte{}
	copy(buf[:], hash)
	return buf, nil
}

// Twox64 returns the xx64 hash of the input data
func Twox64(in []byte) ([]byte, error) {
	hasher := xxhash.NewS64(0)
	_, err := hasher.Write(in)
	if err != nil {
		return nil, err
	}

	res := hasher.Sum64()
	hash := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash, res)
	return hash, nil
}

// Twox128Hash computes xxHash64 twice with seeds 0 and 1 applied on given byte array
func Twox128Hash(msg []byte) ([]byte, error) {
	// compute xxHash64 twice with seeds 0 and 1 applied on given byte array
	h0 := xxhash.NewS64(0) // create xxHash with 0 seed
	_, err := h0.Write(msg)
	if err != nil {
		return nil, err
	}
	res0 := h0.Sum64()
	hash0 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash0, res0)

	h1 := xxhash.NewS64(1) // create xxHash with 1 seed
	_, err = h1.Write(msg)
	if err != nil {
		return nil, err
	}
	res1 := h1.Sum64()
	hash1 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash1, res1)

	//concatenated result
	both := append(hash0, hash1...)
	return both, nil
}

// Twox256 returns the twox256 hash of the input data
func Twox256(in []byte) (Hash, error) {
	h0 := xxhash.NewS64(0)
	_, err := h0.Write(in)
	if err != nil {
		return Hash{}, err
	}
	res0 := h0.Sum64()
	hash0 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash0, res0)

	h1 := xxhash.NewS64(1)
	_, err = h1.Write(in)
	if err != nil {
		return Hash{}, err
	}
	res1 := h1.Sum64()
	hash1 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash1, res1)

	h2 := xxhash.NewS64(2)
	_, err = h2.Write(in)
	if err != nil {
		return Hash{}, err
	}
	res2 := h2.Sum64()
	hash2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash2, res2)

	h3 := xxhash.NewS64(3)
	_, err = h3.Write(in)
	if err != nil {
		return Hash{}, err
	}
	res3 := h3.Sum64()
	hash3 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash3, res3)

	return NewHash(append(append(append(hash0, hash1...), hash2...), hash3...)), nil
}

// Sha256 returns the SHA2-256 hash of the input data
func Sha256(in []byte) Hash {
	hash := sha256.Sum256(in)
	return Hash(hash)
}
