// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

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

		const minValueSize = 0
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

func newWestendDevGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie trie.Trie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetWestendDevRawGenesisPath(t)
	genesisPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genesisPtr

	genesisTrie, err = runtime.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	parentHash := common.NewHash([]byte{0})
	stateRoot := genesisTrie.MustHash()
	extrinsicRoot := trie.EmptyHash
	const number = 0
	digest := types.NewDigest()
	genesisHeader = *types.NewHeader(parentHash,
		stateRoot, extrinsicRoot, number, digest)

	return gen, genesisTrie, genesisHeader
}
