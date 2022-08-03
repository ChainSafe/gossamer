// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"context"
	"testing"

	"github.com/ChainSafe/gossamer/internal/log"
	proofmetrics "github.com/ChainSafe/gossamer/internal/runtime/metrics/proof"
	roothashmetrics "github.com/ChainSafe/gossamer/internal/runtime/metrics/roothash"
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

	cfg := setupConfig(t, tt, DefaultTestLogLvl, 0)
	runtimeFilepath, err := runtime.GetRuntime(context.Background(), targetRuntime)
	require.NoError(t, err)

	rootHashMetrics := roothashmetrics.NewNoop()
	proofMetrics := proofmetrics.NewNoop()
	r, err := NewInstanceFromFile(runtimeFilepath, cfg, rootHashMetrics, proofMetrics)
	require.NoError(t, err, "Got error when trying to create new VM", "targetRuntime", targetRuntime)
	require.NotNil(t, r, "Could not create new VM instance", "targetRuntime", targetRuntime)
	return r
}

func setupConfig(t *testing.T, tt *trie.Trie, lvl log.Level, role byte) runtime.InstanceConfig {
	t.Helper()

	s := storage.NewTrieState(tt)

	ns := runtime.NodeStorage{
		LocalStorage:      runtime.NewInMemoryDB(t),
		PersistentStorage: runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
		BaseDB:            runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
	}

	return runtime.InstanceConfig{
		Storage:     s,
		Keystore:    keystore.NewGlobalKeystore(),
		LogLvl:      lvl,
		NodeStorage: ns,
		Network:     new(runtime.TestRuntimeNetwork),
		Transaction: newTransactionStateMock(),
		Role:        role,
	}
}

// NewTransactionStateMock create and return an runtime Transaction State interface mock
func newTransactionStateMock() *mocks.TransactionState {
	m := new(mocks.TransactionState)
	m.On("AddToPool", mock.AnythingOfType("*transaction.ValidTransaction")).Return(common.BytesToHash([]byte("test")))
	return m
}
