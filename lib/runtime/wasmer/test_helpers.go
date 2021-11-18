// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// DefaultTestLogLvl is the log level used for test runtime instances
var DefaultTestLogLvl = log.Info

// NewTestInstance will create a new runtime instance using the given target runtime
func NewTestInstance(t *testing.T, targetRuntime string) *Instance {
	return NewTestInstanceWithTrie(t, targetRuntime, nil, DefaultTestLogLvl)
}

// NewTestInstanceWithTrie will create a new runtime (polkadot/test) with the supplied trie as the storage
func NewTestInstanceWithTrie(t *testing.T, targetRuntime string, tt *trie.Trie, lvl log.Level) *Instance {
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

func setupConfig(t *testing.T, targetRuntime string, tt *trie.Trie, lvl log.Level, role byte) (string, *Config) {
	testRuntimeFilePath, testRuntimeURL := runtime.GetRuntimeVars(targetRuntime)

	err := runtime.GetRuntimeBlob(testRuntimeFilePath, testRuntimeURL)
	require.Nil(t, err, "Fail: could not get runtime", "targetRuntime", targetRuntime)

	s, err := storage.NewTrieState(tt)
	require.NoError(t, err)

	fp, err := filepath.Abs(testRuntimeFilePath)
	require.Nil(t, err, "could not create testRuntimeFilePath", "targetRuntime", targetRuntime)

	ns := runtime.NodeStorage{
		LocalStorage:      runtime.NewInMemoryDB(t),
		PersistentStorage: runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
		BaseDB:            runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
	}
	cfg := &Config{
		Imports: ImportsNodeRuntime,
	}
	cfg.Storage = s
	cfg.Keystore = keystore.NewGlobalKeystore()
	cfg.LogLvl = lvl
	cfg.NodeStorage = ns
	cfg.Network = new(runtime.TestRuntimeNetwork)
	cfg.Transaction = newTransactionStateMock()
	cfg.Role = role
	return fp, cfg
}

// NewTransactionStateMock create and return an runtime Transaction State interface mock
func newTransactionStateMock() *mocks.TransactionState {
	m := new(mocks.TransactionState)
	m.On("AddToPool", mock.AnythingOfType("*transaction.ValidTransaction")).Return(common.BytesToHash([]byte("test")))
	return m
}
