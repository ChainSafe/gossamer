// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"context"
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
	t.Helper()
	return NewTestInstanceWithTrie(t, targetRuntime, nil)
}

// NewTestInstanceWithTrie will create a new runtime (polkadot/test) with the supplied trie as the storage
func NewTestInstanceWithTrie(t *testing.T, targetRuntime string, tt *trie.Trie) *Instance {
	t.Helper()

	cfg := setupConfig(t, tt, DefaultTestLogLvl, common.NoNetworkRole, targetRuntime)
	runtimeFilepath, err := runtime.GetRuntime(context.Background(), targetRuntime)
	require.NoError(t, err)

	r, err := NewInstanceFromFile(runtimeFilepath, cfg)
	require.NoError(t, err)
	return r
}

func setupConfig(t *testing.T, tt *trie.Trie, lvl log.Level,
	role common.Roles, targetRuntime string) Config {
	t.Helper()

	s := storage.NewTrieState(tt)

	ns := runtime.NodeStorage{
		LocalStorage:      runtime.NewInMemoryDB(t),
		PersistentStorage: runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
		BaseDB:            runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
	}

	version := (*runtime.Version)(nil)
	if targetRuntime == runtime.HOST_API_TEST_RUNTIME {
		// Force state version to 0 since the host api test runtime
		// does not implement the Core_version call so we cannot get the
		// state version from it.
		version = &runtime.Version{}
	}

	return Config{
		Storage:     s,
		Keystore:    keystore.NewGlobalKeystore(),
		LogLvl:      lvl,
		NodeStorage: ns,
		Network:     new(runtime.TestRuntimeNetwork),
		Transaction: newTransactionStateMock(t),
		Role:        role,
		testVersion: version,
	}
}

// NewTransactionStateMock create and return an runtime Transaction State interface mock
func newTransactionStateMock(t *testing.T) *mocks.TransactionState {
	m := mocks.NewTransactionState(t)
	m.On("AddToPool", mock.AnythingOfType("*transaction.ValidTransaction")).Return(common.BytesToHash([]byte("test")))
	return m
}
