// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"crypto/rand"
	"encoding/binary"
	prand "math/rand"
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

	const seed = 912378
	generator := prand.New(prand.NewSource(seed)) //nolint:gosec

	for i := range rt {
		test := generateRandomTest(t, kv, generator)
		rt[i] = test
		kv[string(test.key)] = rt[i].value
	}

	return rt
}

func generateRandomTest(t testing.TB, kv map[string][]byte, generator *prand.Rand) Test {
	test := Test{}

	for {
		var n int64 = 2 // arbitrary positive number
		size := int64(generator.Intn(510))

		buf := make([]byte, size+n)
		_, err := generator.Read(buf)
		require.NoError(t, err)

		key := binary.LittleEndian.Uint16(buf[:2])

		if kv[string(buf)] == nil || key < 256 {
			test.key = buf

			size := int64(generator.Intn(128))

			buf = make([]byte, size+n)
			_, err = generator.Read(buf)
			require.NoError(t, err)

			test.value = buf

			return test
		}
	}
}

func rand32Bytes() []byte {
	r := make([]byte, 32)
	_, err := rand.Read(r)
	if err != nil {
		panic(err)
	}
	return r
}
