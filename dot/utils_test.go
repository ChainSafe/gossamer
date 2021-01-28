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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/dgraph-io/badger/v2"
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

	cfg.Init.GenesisRaw = genFile.Name()
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

func TestBenchmarkWithTimeStamp(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_timestampBenchmark")
	require.NoError(t, err)

	t.Cleanup(func() {
		if err = os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})

	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	opts := badger.DefaultOptions(filepath.Join(dir, "badger"))
	opts.SyncWrites = false
	db, err := badger.OpenManaged(opts)

	genRawFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genRawFile)
	defer os.Remove(genRawFile.Name())

	genRaw, err := genesis.NewGenesisFromJSONRaw(genRawFile.Name())
	require.NoError(t, err)

	start := time.Now()
	for i := 1; i <= 1001; i++ {
		// Write after every 100th entry (since runtime key is hardly changed).
		if i%100 != 1 {
			continue
		}
		// Start writing at a particular timestamp(MVCC).
		wrtBatch := db.NewWriteBatchAt(uint64(i))
		// Write all keys to database.
		for k, v := range genRaw.Genesis.Raw["top"] {
			err = wrtBatch.Set([]byte(k), append([]byte(v), uint8(i)))
			require.NoError(t, err)
		}
		require.NoError(t, wrtBatch.Flush())
		// Write to disk or data might be lost.
		require.NoError(t, db.Sync())
	}

	elapsed := time.Since(start)
	log.Printf("Writing to DB took %s", elapsed)

	k := "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da923a05cabf6d3bde7ca3ef0d11596b5611cbd2d43530a44705ad088af313e18f80b53ef16b36177cd4b77b846f2a5f07c"
	val := "0x00000000000000000000000000000010"
	expectedVal := append([]byte(val), 1)

	for i := 1; i <= 1001; i++ {
		txn := db.NewTransactionAt(uint64(i), false)

		item, err := txn.Get([]byte(k))
		require.NoError(t, err)

		var data []byte
		data, err = item.ValueCopy(data)
		require.NoError(t, err)

		// After every 100 iteration value is changing
		if i%100 == 1 {
			expectedVal = append([]byte(val), uint8(i))
		}
		require.Equal(t, expectedVal, data)
	}
	require.NoError(t, db.Close())
}

func TestBenchmark1(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_benchmark")
	require.NoError(t, err)

	t.Cleanup(func() {
		if err = os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})

	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	opts := badger.DefaultOptions(filepath.Join(dir, "badger"))
	opts.SyncWrites = false
	db, err := badger.Open(opts)
	genRawFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genRawFile)
	defer os.Remove(genRawFile.Name())

	genRaw, err := genesis.NewGenesisFromJSONRaw(genRawFile.Name())
	require.NoError(t, err)

	start := time.Now()
	for i := 1; i <= 1001; i++ {
		txn := db.NewTransaction(true)
		for k, v := range genRaw.Genesis.Raw["top"] {
			err = txn.Set(append([]byte(k), uint8(i)), []byte(v))
			require.NoError(t, err)
		}
		require.NoError(t, txn.Commit())
		require.NoError(t, db.Sync())
	}
	elapsed := time.Since(start)
	log.Printf("Writing to DB took %s", elapsed)

	key := "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da923a05cabf6d3bde7ca3ef0d11596b5611cbd2d43530a44705ad088af313e18f80b53ef16b36177cd4b77b846f2a5f07c"
	expectedVal := "0x00000000000000000000000000000010"
	for i := 1; i <= 1001; i++ {
		txn := db.NewTransaction(true)
		item, err := txn.Get(append([]byte(key), uint8(i)))
		require.NoError(t, err)

		var data []byte
		data, err = item.ValueCopy(data)
		require.NoError(t, err)
		require.Equal(t, []byte(expectedVal), data)
	}
	require.NoError(t, db.Close())
}
