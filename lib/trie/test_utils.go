package trie

import (
	"encoding/binary"
	"math/rand"
)

// Test represents a key-value pair for a test
type Test struct {
	key   []byte
	value []byte
	pk    []byte
	op    int
}

func (t *Test) Key() []byte {
	return t.key
}

func (t *Test) Value() []byte {
	return t.value
}

// GenerateRandomTests returns an array of random Tests
func GenerateRandomTests(size int) []Test {
	rt := make([]Test, size)
	kv := make(map[string][]byte)

	for i := range rt {
		test := generateRandomTest(kv)
		rt[i] = test
		kv[string(test.key)] = rt[i].value
	}

	return rt
}

func generateRandomTest(kv map[string][]byte) Test {
	r := *rand.New(rand.NewSource(rand.Int63()))
	test := Test{}

	for {
		n := 2 // arbitrary positive number
		size := r.Intn(510) + n
		buf := make([]byte, size)
		r.Read(buf)

		key := binary.LittleEndian.Uint16(buf[:2])

		if kv[string(buf)] == nil || key < 256 {
			test.key = buf

			buf = make([]byte, r.Intn(128)+n)
			r.Read(buf)
			test.value = buf

			return test
		}
	}
}
