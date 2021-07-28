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

package wasmer

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// DefaultTestLogLvl is the log level used for test runtime instances
var DefaultTestLogLvl = log.LvlInfo

// NewTestInstance will create a new runtime instance using the given target runtime
func NewTestInstance(t *testing.T, targetRuntime string) *Instance {
	return NewTestInstanceWithTrie(t, targetRuntime, nil, DefaultTestLogLvl)
}

// NewTestInstanceWithTrie will create a new runtime (polkadot/test) with the supplied trie as the storage
func NewTestInstanceWithTrie(t *testing.T, targetRuntime string, tt *trie.Trie, lvl log.Lvl) *Instance {
	fp, cfg := setupConfig(t, targetRuntime, tt, DefaultTestLogLvl, 0)
	r, err := NewInstanceFromFile(fp, cfg)
	require.NoError(t, err, "Got error when trying to create new VM", "targetRuntime", targetRuntime)
	require.NotNil(t, r, "Could not create new VM instance", "targetRuntime", targetRuntime)
	return r
}

// NewTestInstanceWithRole returns a test runtime with given role value
func NewTestInstanceWithRole(t *testing.T, targetRuntime string, role byte) *Instance {
	fp, cfg := setupConfig(t, targetRuntime, nil, DefaultTestLogLvl, role)
	r, err := NewInstanceFromFile(fp, cfg)
	require.NoError(t, err, "Got error when trying to create new VM", "targetRuntime", targetRuntime)
	require.NotNil(t, r, "Could not create new VM instance", "targetRuntime", targetRuntime)
	return r
}

func setupConfig(t *testing.T, targetRuntime string, tt *trie.Trie, lvl log.Lvl, role byte) (string, *Config) {
	testRuntimeFilePath, testRuntimeURL := runtime.GetRuntimeVars(targetRuntime)

	_, err := runtime.GetRuntimeBlob(testRuntimeFilePath, testRuntimeURL)
	require.Nil(t, err, "Fail: could not get runtime", "targetRuntime", targetRuntime)

	s, err := storage.NewTrieState(tt)
	require.NoError(t, err)

	fp, err := filepath.Abs(testRuntimeFilePath)
	require.Nil(t, err, "could not create testRuntimeFilePath", "targetRuntime", targetRuntime)

	ns := runtime.NodeStorage{
		LocalStorage:      runtime.NewInMemoryDB(t),
		PersistentStorage: runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
	}
	cfg := &Config{
		Imports: ImportsNodeRuntime,
	}
	cfg.Storage = s
	cfg.Keystore = keystore.NewGlobalKeystore()
	cfg.LogLvl = lvl
	cfg.NodeStorage = ns
	cfg.Network = new(runtime.TestRuntimeNetwork)
	cfg.Transaction = NewTransactionStateMock()
	cfg.Role = role
	return fp, cfg
}

// NewTransactionStateMock create and return an runtime Transaction State interface mock
func NewTransactionStateMock() *runtime.MockTransactionState {
	m := new(runtime.MockTransactionState)
	m.On("AddToPool", mock.AnythingOfType("*transaction.ValidTransaction")).Return(common.BytesToHash([]byte("test")))
	return m
}
