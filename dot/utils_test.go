// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"log"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// TestNewConfig tests the NewTestConfig method
func TestNewConfig(t *testing.T) {
	cfg := NewTestConfig(t)
	defer utils.RemoveTestDir(t)
	require.NotNil(t, cfg)
}

// TestNewConfigAndFile tests the NewTestConfigWithFile method
func TestNewConfigAndFile(t *testing.T) {
	testCfg, testCfgFile := NewTestConfigWithFile(t)
	defer utils.RemoveTestDir(t)
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)
}

// TestInitNode
func TestNewTestGenesis(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()
}

func TestNewTestGenesisFile(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genHRFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genHRFile)
	defer os.Remove(genHRFile.Name())

	genRawFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genRawFile)
	defer os.Remove(genRawFile.Name())

	genHR, err := genesis.NewGenesisFromJSON(genHRFile.Name(), 0)
	require.NoError(t, err)
	genRaw, err := genesis.NewGenesisFromJSONRaw(genRawFile.Name())
	require.NoError(t, err)

	// values from raw genesis file should equal values generated from human readable genesis file
	require.Equal(t, genRaw.Genesis.Raw["top"], genHR.Genesis.Raw["top"])
}

func TestDeepCopyVsSnapshot(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genRawFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genRawFile)

	defer os.Remove(genRawFile.Name())

	genRaw, err := genesis.NewGenesisFromJSONRaw(genRawFile.Name())
	require.NoError(t, err)

	tri := trie.NewEmptyTrie()
	var ttlLenght int
	for k, v := range genRaw.Genesis.Raw["top"] {
		val := []byte(v)
		ttlLenght += len(val)
		tri.Put([]byte(k), val)
	}

	testCases := []struct {
		name string
		fn   func(tri *trie.Trie) (*trie.Trie, error)
	}{
		{"DeepCopy", func(tri *trie.Trie) (*trie.Trie, error) {
			return tri.DeepCopy()
		}},
		{"Snapshot", func(tri *trie.Trie) (*trie.Trie, error) {
			return tri.Snapshot(), nil
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			trieMap := make(map[int]*trie.Trie)
			start := time.Now()
			var m runtime.MemStats
			for i := 0; i <= 200; i++ {
				newTrie, err := tc.fn(tri)
				require.NoError(t, err)

				runtime.ReadMemStats(&m)
				trieMap[i] = newTrie
			}

			log.Printf("\nAlloc = %v MB \nTotalAlloc = %v MB \nSys = %v MB \nNumGC = %v \n\n", m.Alloc/(1024*1024), m.TotalAlloc/(1024*1024), m.Sys/(1024*1024), m.NumGC)
			elapsed := time.Since(start)
			log.Printf("DeepCopy to trie took %s", elapsed)
			runtime.GC()
		})
	}
}

func TestTrieSnapshot(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genRawFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genRawFile)

	defer os.Remove(genRawFile.Name())

	genRaw, err := genesis.NewGenesisFromJSONRaw(genRawFile.Name())
	require.NoError(t, err)

	tri := trie.NewEmptyTrie()
	key := []byte("key")
	value := []byte("value")

	for k, v := range genRaw.Genesis.Raw["top"] {
		val := []byte(v)
		tri.Put([]byte(k), val)
	}

	// DeepCopy the trie.
	dcTrie, err := tri.DeepCopy()
	require.NoError(t, err)

	// Take Snapshot of the trie.
	newTrie := tri.Snapshot()

	// Get the Trie root hash for all the 3 tries.
	tHash, err := tri.Hash()
	require.NoError(t, err)

	dcTrieHash, err := dcTrie.Hash()
	require.NoError(t, err)

	newTrieHash, err := newTrie.Hash()
	require.NoError(t, err)

	// Root hash for the 3 tries should be equal.
	require.Equal(t, tHash, dcTrieHash)
	require.Equal(t, tHash, newTrieHash)

	// Modify the current trie.
	value[0] = 'w'
	newTrie.Put(key, value)

	// Get the updated root hash of all tries.
	tHash, err = tri.Hash()
	require.NoError(t, err)

	dcTrieHash, err = dcTrie.Hash()
	require.NoError(t, err)

	newTrieHash, err = newTrie.Hash()
	require.NoError(t, err)

	// Only the current trie should have a different root hash since it is updated.
	require.NotEqual(t, newTrieHash, dcTrieHash)
	require.NotEqual(t, newTrieHash, tHash)
	require.Equal(t, dcTrieHash, tHash)
}
