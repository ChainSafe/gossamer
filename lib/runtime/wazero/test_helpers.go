// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wazero_runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// NewTestInstance will create a new runtime instance using the given target runtime
func NewTestInstance(t *testing.T, targetRuntime string) *Instance {
	t.Helper()
	return NewTestInstanceWithTrie(t, targetRuntime, nil)
}

func setupConfig(t *testing.T, ctrl *gomock.Controller, tt *trie.Trie, lvl log.Level, role common.NetworkRole) Config {
	t.Helper()

	s := storage.NewTrieState(tt)

	ns := runtime.NodeStorage{
		LocalStorage:      runtime.NewInMemoryDB(t),
		PersistentStorage: runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
		BaseDB:            runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
	}

	return Config{
		Storage:     s,
		Keystore:    keystore.NewGlobalKeystore(),
		LogLvl:      lvl,
		NodeStorage: ns,
		Network:     new(runtime.TestRuntimeNetwork),
		Transaction: mocks.NewMockTransactionState(ctrl),
		Role:        role,
	}
}

// DefaultTestLogLvl is the log level used for test runtime instances
var DefaultTestLogLvl = log.Info

// NewTestInstanceWithTrie returns an instance based on the target runtime string specified,
// which can be a file path or a constant from the constants defined in `lib/runtime/constants.go`.
// The instance uses the trie given as argument for its storage.
func NewTestInstanceWithTrie(t *testing.T, targetRuntime string, tt *trie.Trie) *Instance {
	t.Helper()

	ctrl := gomock.NewController(t)

	cfg := setupConfig(t, ctrl, tt, DefaultTestLogLvl, common.NoNetworkRole)
	targetRuntime, err := runtime.GetRuntime(context.Background(), targetRuntime)
	require.NoError(t, err)

	r, err := NewInstanceFromFile(targetRuntime, cfg)
	require.NoError(t, err)

	return r
}

func NewBenchInstanceWithTrie(b *testing.B, targetRuntime string, tt *trie.Trie) *Instance {
	b.Helper()

	ctrl := gomock.NewController(b)

	cfg := setupBenchConfig(b, ctrl, tt, DefaultTestLogLvl, common.NoNetworkRole)
	targetRuntime, err := runtime.GetRuntime(context.Background(), targetRuntime)
	require.NoError(b, err)

	r, err := NewInstanceFromFile(targetRuntime, cfg)
	require.NoError(b, err)

	return r
}

func setupBenchConfig(b *testing.B, ctrl *gomock.Controller, tt *trie.Trie, lvl log.Level, role common.NetworkRole) Config {
	b.Helper()

	s := storage.NewTrieState(tt)

	ns := runtime.NodeStorage{
		LocalStorage:      runtime.NewBenchInMemoryDB(b),
		PersistentStorage: runtime.NewBenchInMemoryDB(b),
		BaseDB:            runtime.NewBenchInMemoryDB(b),
	}

	return Config{
		Storage:     s,
		Keystore:    keystore.NewGlobalKeystore(),
		LogLvl:      lvl,
		NodeStorage: ns,
		Network:     new(runtime.TestRuntimeNetwork),
		Transaction: mocks.NewMockTransactionState(ctrl),
		Role:        role,
	}
}

// NewInstanceFromFile instantiates a runtime from a .wasm file
func NewInstanceFromFile(fp string, cfg Config) (*Instance, error) {
	// Reads the WebAssembly module as bytes.
	// Retrieve WASM binary
	bytes, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, fmt.Errorf("failed to read wasm file: %s", err)
	}

	return NewInstance(bytes, cfg)
}
