// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	crand "crypto/rand"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

func newTriesEmpty() *Tries {
	return &Tries{
		rootToTrie:    make(map[common.Hash]*trie.Trie),
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}
}

func generateKeyValues(tb testing.TB, size int) (kv map[string][]byte) {
	tb.Helper()

	kv = make(map[string][]byte, size)

	const maxKeySize, maxValueSize = 510, 128
	for i := 0; i < size; i++ {
		populateKeyValueMap(tb, kv, maxKeySize, maxValueSize)
	}

	return kv
}

func populateKeyValueMap(tb testing.TB, kv map[string][]byte, maxKeySize, maxValueSize int) {
	tb.Helper()

	for {
		const minKeySize = 2
		key := generateRandBytesMinMax(tb, minKeySize, maxKeySize)

		keyString := string(key)

		_, keyExists := kv[keyString]

		if keyExists && key[1] != byte(0) {
			continue
		}

		const minValueSize = 0
		value := generateRandBytesMinMax(tb, minValueSize, maxValueSize)

		kv[keyString] = value

		break
	}
}

func generateRandBytesMinMax(tb testing.TB, minSize, maxSize int) (b []byte) {
	tb.Helper()
	randN, err := crand.Int(crand.Reader, big.NewInt(int64(maxSize-minSize)))
	require.NoError(tb, err)
	size := minSize + int(randN.Int64())
	return generateRandBytes(tb, size)
}

func generateRandBytes(tb testing.TB, size int) (b []byte) {
	tb.Helper()
	b = make([]byte, size)
	_, err := crand.Read(b)
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

	genesisTrie, err = wasmer.NewTrieFromGenesis(gen)
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
