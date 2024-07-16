// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests results are based on
// https://github.com/dimartiro/substrate-trie-test/blob/master/src/substrate_trie_test.rs
func TestRecorder(t *testing.T) {
	inmemoryDB := NewMemoryDB(emptyNode)

	triedb := NewEmptyTrieDB(inmemoryDB)

	triedb.Put([]byte("pol"), []byte("polvalue"))
	triedb.Put([]byte("polka"), []byte("polkavalue"))
	triedb.Put([]byte("polkadot"), []byte("polkadotvalue"))
	triedb.Put([]byte("go"), []byte("govalue"))
	triedb.Put([]byte("gossamer"), []byte("gossamervalue"))

	// Commit and get root
	root := triedb.MustHash()
	require.NotNil(t, root)

	t.Run("Record_pol_access_should_record_2_node", func(t *testing.T) {
		recorder := NewRecorder()
		trie := NewTrieDB(root, inmemoryDB, WithRecorder(recorder))

		trie.Get([]byte("pol"))

		recordedNodes := recorder.Drain()
		encodedNodes := [][]byte{}
		for _, node := range recordedNodes {
			encodedNodes = append(encodedNodes, node.data)
		}

		expectedNodes := [][]byte{
			// Root node
			{
				128, 192, 0, 128, 124, 255, 5, 248, 100, 180, 218,
				180, 146, 187, 118, 79, 161, 92, 153, 38, 78, 48,
				120, 69, 157, 112, 164, 176, 129, 164, 167, 36, 76,
				131, 68, 6, 128, 42, 2, 217, 41, 157, 5, 134, 74, 180,
				2, 124, 111, 183, 89, 195, 14, 111, 92, 59, 236, 175,
				34, 115, 200, 121, 201, 142, 57, 123, 84, 26, 222,
			},
			// Node for "pol"
			{
				197, 0, 111, 108, 64, 0, 32, 112, 111, 108, 118, 97,
				108, 117, 101, 128, 176, 59, 74, 69, 116, 80, 243, 95,
				83, 201, 2, 181, 136, 129, 18, 72, 171, 217, 123, 106,
				252, 198, 126, 49, 210, 152, 238, 0, 84, 233, 94, 217,
			},
		}

		for i, node := range encodedNodes {
			require.Equal(t, node, expectedNodes[i])
		}
	})

	t.Run("Record_go_access_should_record_2_nodes", func(t *testing.T) {
		recorder := NewRecorder()
		trie := NewTrieDB(root, inmemoryDB, WithRecorder(recorder))

		trie.Get([]byte("go"))

		recordedNodes := recorder.Drain()
		encodedNodes := [][]byte{}
		for _, node := range recordedNodes {
			encodedNodes = append(encodedNodes, node.data)
		}

		expectedNodes := [][]byte{
			// Root node
			{
				128, 192, 0, 128, 124, 255, 5, 248, 100, 180, 218, 180, 146, 187,
				118, 79, 161, 92, 153, 38, 78, 48, 120, 69, 157, 112, 164, 176, 129,
				164, 167, 36, 76, 131, 68, 6, 128, 42, 2, 217, 41, 157, 5, 134, 74, 180,
				2, 124, 111, 183, 89, 195, 14, 111, 92, 59, 236, 175, 34, 115, 200, 121,
				201, 142, 57, 123, 84, 26, 222,
			},

			// Node for "go"
			{
				195, 7, 111, 128, 0, 28, 103, 111, 118, 97, 108, 117, 101, 84, 75,
				3, 115, 97, 109, 101, 114, 52, 103, 111, 115, 115, 97, 109, 101,
				114, 118, 97, 108, 117, 101,
			},
		}

		for i, node := range encodedNodes {
			require.Equal(t, node, expectedNodes[i])
		}
	})
}
