// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package mmr

import (
	"encoding/binary"
	"hash"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/blake2b"
)

func newInMemMMR(hasher hash.Hash) *MMR {
	return NewMMR(0, NewMemStorage(), hasher)
}

func hashNumber(number uint32) MMRElement {
	hasher, _ := blake2b.New256(nil)
	var numBytes [4]byte
	binary.LittleEndian.PutUint32(numBytes[:], number)
	hasher.Write(numBytes[:])

	var hash [32]byte
	hasher.Sum(hash[:0])
	return hash[:]
}

func TestPushOneElement_RootShouldBeSameLeaf(t *testing.T) {
	hasher, err := blake2b.New256(nil)
	assert.NoError(t, err)

	inMemMMR := newInMemMMR(hasher)

	leaf := hashNumber(0)
	_, err = inMemMMR.Push(leaf)
	assert.NoError(t, err)

	root, err := inMemMMR.Root()
	assert.NoError(t, err)

	assert.Equal(t, root, leaf)
}

// Compared with the same MMR using substrate's implementation
func TestPushManyElementsGetRootOk(t *testing.T) {
	hasher, err := blake2b.New256(nil)
	assert.NoError(t, err)

	inMemMMR := newInMemMMR(hasher)

	for i := uint32(0); i < 100; i++ {
		leaf := hashNumber(i)
		_, err := inMemMMR.Push(leaf)
		assert.NoError(t, err)
	}

	root, err := inMemMMR.Root()
	assert.NoError(t, err)

	assert.Equal(t, []byte{
		0x5, 0x0, 0xd0, 0xeb, 0xdb, 0xca, 0xd3, 0x6a, 0x79, 0xd3, 0x32, 0x5d,
		0xbd, 0x2a, 0x4b, 0x2b, 0x97, 0x30, 0x1d, 0x8e, 0x48, 0x2a, 0x9b, 0xe2,
		0x2, 0x1, 0x6e, 0x9f, 0x1c, 0xaa, 0xe1, 0x3f,
	}, []byte(root))
}
