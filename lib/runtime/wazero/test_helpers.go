// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wazero_runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/mocks"
	inmemory_storage "github.com/ChainSafe/gossamer/lib/runtime/storage/inmemory"
	storage "github.com/ChainSafe/gossamer/lib/runtime/storage/inmemory"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// DefaultTestLogLvl is the log level used for test runtime instances
var DefaultTestLogLvl = log.Info

type TestInstanceOption func(*Config)

func TestWithLogLevel(lvl log.Level) TestInstanceOption {
	return func(c *Config) {
		c.LogLvl = lvl
	}
}

func TestWithTrie(tt *trie.InMemoryTrie) TestInstanceOption {
	return func(c *Config) {
		c.Storage = inmemory_storage.NewTrieState(tt)
	}
}

func TestWithVersion(version *runtime.Version) TestInstanceOption {
	return func(c *Config) {
		c.DefaultVersion = version
	}
}

func NewTestInstance(t *testing.T, targetRuntime string, opts ...TestInstanceOption) *Instance {
	t.Helper()

	ctrl := gomock.NewController(t)
	cfg := &Config{
		Storage:  storage.NewTrieState(trie.NewEmptyInmemoryTrie()),
		Keystore: keystore.NewGlobalKeystore(),
		LogLvl:   DefaultTestLogLvl,
		NodeStorage: runtime.NodeStorage{
			LocalStorage:      runtime.NewInMemoryDB(t),
			PersistentStorage: runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
			BaseDB:            runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
		},
		Network:     new(runtime.TestRuntimeNetwork),
		Transaction: mocks.NewMockTransactionState(ctrl),
		Role:        common.NoNetworkRole,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	targetRuntime, err := runtime.GetRuntime(context.Background(), targetRuntime)
	require.NoError(t, err)

	// Reads the WebAssembly module as bytes.
	// Retrieve WASM binary
	bytes, err := os.ReadFile(filepath.Clean(targetRuntime))
	require.NoError(t, err)

	r, err := NewInstance(bytes, *cfg)
	require.NoError(t, err)
	return r
}
