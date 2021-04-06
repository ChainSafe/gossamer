// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package dot

import (
	"log"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// TestNewConfig tests the NewTestConfig method
func TestNewConfig(t *testing.T) {
	cfg := NewTestConfig(t)

	defer utils.RemoveTestDir(t)

	// TODO: improve dot tests #687
	require.NotNil(t, cfg)
}

// TestNewConfigAndFile tests the NewTestConfigWithFile method
func TestNewConfigAndFile(t *testing.T) {
	testCfg, testCfgFile := NewTestConfigWithFile(t)

	defer utils.RemoveTestDir(t)

	// TODO: improve dot tests #687
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

func TestNewRuntimeFromGenesis(t *testing.T) {
	gen := NewTestGenesis(t)
	_, err := wasmer.NewRuntimeFromGenesis(gen, &wasmer.Config{})
	require.NoError(t, err)
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
	ssTrie := tri.Snapshot()

	// Get the Trie root hash for all the 3 tries.
	tHash, err := tri.Hash()
	require.NoError(t, err)

	dcTrieHash, err := dcTrie.Hash()
	require.NoError(t, err)

	ssTrieHash, err := ssTrie.Hash()
	require.NoError(t, err)

	// Root hash for all the 3 tries should be equal.
	require.Equal(t, tHash, dcTrieHash)
	require.Equal(t, dcTrieHash, ssTrieHash)

	// Modify the current trie.
	value[0] = 'w'
	tri.Put(key, value)

	// Get the updated root hash of all tries.
	tHash, err = tri.Hash()
	require.NoError(t, err)

	dcTrieHash, err = dcTrie.Hash()
	require.NoError(t, err)

	ssTrieHash, err = ssTrie.Hash()
	require.NoError(t, err)

	// Only the current trie should have a different root hash since it is updated.
	require.NotEqual(t, tHash, dcTrieHash)
	require.NotEqual(t, tHash, ssTrieHash)
	require.Equal(t, dcTrieHash, ssTrieHash)
}
