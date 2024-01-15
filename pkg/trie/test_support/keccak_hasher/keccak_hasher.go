// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keccak_hasher

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"golang.org/x/crypto/sha3"
)

const KeccakHasherLength = 32

type KeccakHash [KeccakHasherLength]byte

func (k KeccakHash) Bytes() []byte {
	return k[:]
}

func (k KeccakHash) ComparableKey() string {
	return fmt.Sprintf("%x", k)
}

var _ hashdb.HashOut = KeccakHash{}

type KeccakHasher struct{}

func (k KeccakHasher) Length() int {
	return KeccakHasherLength
}

func (k KeccakHasher) FromBytes(in []byte) KeccakHash {
	var buf = [KeccakHasherLength]byte{}
	copy(buf[:], in)
	return KeccakHash(buf)
}

func (k KeccakHasher) Hash(in []byte) KeccakHash {
	h := sha3.NewLegacyKeccak256()

	_, err := h.Write(in)
	if err != nil {
		panic("Unexpected error hashing bytes")
	}

	hash := h.Sum(nil)
	return k.FromBytes(hash)
}

func NewKeccakHasher() KeccakHasher {
	return KeccakHasher{}
}

var _ hashdb.Hasher[KeccakHash] = (*KeccakHasher)(nil)
