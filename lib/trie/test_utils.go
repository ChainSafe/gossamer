// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test represents a key-value pair for a test
type Test struct {
	key   []byte
	value []byte
	pk    []byte
	op    int
}

// Key returns the test key
func (t *Test) Key() []byte {
	return t.key
}

// Value returns the test value
func (t *Test) Value() []byte {
	return t.value
}

// GenerateRandomTests returns an array of random Tests
func GenerateRandomTests(t testing.TB, size int) []Test {
	rt := make([]Test, size)
	kv := make(map[string][]byte)

	for i := range rt {
		test := generateRandomTest(t, kv)
		rt[i] = test
		kv[string(test.key)] = rt[i].value
	}

	return rt
}

func generateRandomTest(t testing.TB, kv map[string][]byte) Test {
	test := Test{}

	for {
		n := 2 // arbitrary positive number
		size, err := rand.Int(rand.Reader, big.NewInt(510))
		require.NoError(t, err)

		buf := make([]byte, size.Int64()+int64(n))
		_, err = rand.Read(buf)
		require.NoError(t, err)

		key := binary.LittleEndian.Uint16(buf[:2])

		if kv[string(buf)] == nil || key < 256 {
			test.key = buf

			size, err := rand.Int(rand.Reader, big.NewInt(128))
			require.NoError(t, err)

			buf = make([]byte, size.Int64()+int64(n))
			_, err = rand.Read(buf)
			require.NoError(t, err)

			test.value = buf

			return test
		}
	}
}

func rand32Bytes() []byte {
	r := make([]byte, 32)
	rand.Read(r) //nolint
	return r
}
