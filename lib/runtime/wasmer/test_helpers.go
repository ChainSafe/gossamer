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
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// DefaultTestLogLvl is the log level used for test runtime instances
var DefaultTestLogLvl = log.Info

func mustHexTo64BArray(t *testing.T, inputHex string) (outputArray [64]byte) {
	t.Helper()
	copy(outputArray[:], common.MustHexToBytes(inputHex))
	return outputArray
}

func mustHexTo32BArray(t *testing.T, inputHex string) (outputArray [32]byte) {
	t.Helper()
	copy(outputArray[:], common.MustHexToBytes(inputHex))
	return outputArray
}

// NewTestInstance will create a new runtime instance using the given target runtime
func NewTestInstance(t *testing.T, targetRuntime string) *Instance {
	t.Helper()
	return NewTestInstanceWithTrie(t, targetRuntime, nil)
}

// NewTestInstanceWithTrie returns an instance based on the target runtime string specified,
// which can be a file path or a constant from the constants defined in `lib/runtime/constants.go`.
// The instance uses the trie given as argument for its storage.
func NewTestInstanceWithTrie(t *testing.T, targetRuntime string, tt *trie.Trie) *Instance {
	t.Helper()

	ctrl := gomock.NewController(t)

	cfg := setupConfig(t, ctrl, tt, DefaultTestLogLvl, common.NoNetworkRole, targetRuntime)
	targetRuntime, err := runtime.GetRuntime(context.Background(), targetRuntime)
	require.NoError(t, err)

	r, err := NewInstanceFromFile(targetRuntime, cfg)
	require.NoError(t, err)

	return r
}

func setupConfig(t *testing.T, ctrl *gomock.Controller, tt *trie.Trie, lvl log.Level,
	role common.NetworkRole, targetRuntime string) Config {
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
		Transaction: mocks.NewMockTransactionState(ctrl),
		Role:        role,
		testVersion: version,
	}
}
