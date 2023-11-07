// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/internal/trie/tracking"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

type keyValues struct {
	key   []byte
	value []byte
	op    int
}

// newGenerator creates a new PRNG seeded with the
// unix nanoseconds value of the current time.
func newGenerator() (prng *rand.Rand) {
	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	return rand.New(source) //skipcq: GSC-G404
}

func generateKeyValues(tb testing.TB, generator *rand.Rand, size int) (kv map[string][]byte) {
	tb.Helper()

	kv = make(map[string][]byte, size)

	const maxKeySize, maxValueSize = 510, 128
	for i := 0; i < size; i++ {
		populateKeyValueMap(tb, kv, generator, maxKeySize, maxValueSize)
	}

	return kv
}

func populateKeyValueMap(tb testing.TB, kv map[string][]byte,
	generator *rand.Rand, maxKeySize, maxValueSize int) {
	tb.Helper()

	for {
		const minKeySize = 2
		key := generateRandBytesMinMax(tb, minKeySize, maxKeySize, generator)

		keyString := string(key)

		_, keyExists := kv[keyString]

		if keyExists && key[1] != byte(0) {
			continue
		}

		const minValueSize = 1 // not 0 otherwise it mixes empty and nil byte slices
		value := generateRandBytesMinMax(tb, minValueSize, maxValueSize, generator)

		kv[keyString] = value

		break
	}
}

func generateRandBytesMinMax(tb testing.TB, minSize, maxSize int,
	generator *rand.Rand) (b []byte) {
	tb.Helper()
	size := minSize +
		generator.Intn(maxSize-minSize)
	return generateRandBytes(tb, size, generator)
}

func generateRandBytes(tb testing.TB, size int,
	generator *rand.Rand) (b []byte) {
	tb.Helper()
	b = make([]byte, size)
	_, err := generator.Read(b)
	require.NoError(tb, err)
	return b
}

func makeSeededTrie(t *testing.T, size int) (
	trie *Trie, keyValues map[string][]byte) {
	generator := newGenerator()
	keyValues = generateKeyValues(t, generator, size)

	trie = NewEmptyTrie()

	for keyString, value := range keyValues {
		key := []byte(keyString)
		trie.Put(key, value)
	}

	return trie, keyValues
}

func pickKeys(keyValues map[string][]byte,
	generator *rand.Rand, n int) (keys [][]byte) {
	allKeys := maps.Keys(keyValues)
	keys = make([][]byte, n)
	for i := range keys {
		pickedIndex := generator.Intn(len(allKeys))
		pickedKeyString := allKeys[pickedIndex]
		keys[i] = []byte(pickedKeyString)
	}

	return keys
}

func padRightChildren(slice []*Node) (paddedSlice []*Node) {
	paddedSlice = make([]*Node, node.ChildrenCapacity)
	copy(paddedSlice, slice)
	return paddedSlice
}

func checkMerkleValuesAreSet(t *testing.T, n *Node) {
	t.Helper()

	if n == nil {
		return
	}

	require.NotEmpty(t, n.MerkleValue)
	if n.Kind() == node.Leaf {
		return
	}

	for _, child := range n.Children {
		checkMerkleValuesAreSet(t, child)
	}
}

func newDeltas(deletedNodeHashesHex ...string) (deltas *tracking.Deltas) {
	deltas = tracking.New()
	for _, deletedNodeHashHex := range deletedNodeHashesHex {
		nodeHash := common.MustHexToHash(deletedNodeHashHex)
		deltas.RecordDeleted(nodeHash)
	}
	return deltas
}
