// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package client

import (
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/db"
	statedb "github.com/ChainSafe/gossamer/internal/client/state-db"
	memorykvdb "github.com/ChainSafe/gossamer/internal/kvdb/memory-kvdb"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	rt_testing "github.com/ChainSafe/gossamer/internal/primitives/runtime/testing"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/stretchr/testify/require"
)

type noopExtrinsic struct{}

func (noopExtrinsic) IsSigned() *bool {
	return nil
}

var _ runtime.Extrinsic = noopExtrinsic{}

type TestClient struct {
	Client[
		hash.H256, runtime.BlakeTwo256, uint64, noopExtrinsic, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	]
}

var (
	_ api.BlockchainEvents[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]] = &TestClient{}
	_ api.PreCommitActions[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]] = &TestClient{}
)

func NewTestBackend(t *testing.T,
	blocksPruning db.BlocksPruning, canonicalizationDelay uint64,
) api.Backend[
	hash.H256,
	uint64,
	runtime.BlakeTwo256,
	*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	rt_testing.ExtrinsicsWrapper[uint64],
] {
	t.Helper()

	kvdb := memorykvdb.New(13)
	var statePruning statedb.PruningMode
	switch blocksPruning := blocksPruning.(type) {
	case db.BlocksPruningKeepAll:
		statePruning = statedb.PruningModeArchiveAll{}
	case db.BlocksPruningKeepFinalized:
		statePruning = statedb.PruningModeArchiveCanonical{}
	case db.BlocksPruningSome:
		statePruning = statedb.NewPruningModeConstrained(uint32(blocksPruning))
	default:
		t.Fatalf("unreachable")
	}
	trieCacheMaxSize := uint(16 * 1024 * 1024)
	dbSetting := db.DatabaseConfig{
		TrieCacheMaximumSize: &trieCacheMaxSize,
		StatePruning:         statePruning,
		Source:               db.DatabaseSource{DB: database.NewDBAdapter[hash.H256](kvdb), RequireCreateFlag: true},
		BlocksPruning:        blocksPruning,
	}

	backend, err := db.NewBackend[
		hash.H256,
		uint64,
		rt_testing.ExtrinsicsWrapper[uint64],
		runtime.BlakeTwo256,
		*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	](dbSetting, canonicalizationDelay)
	if err != nil {
		panic(err)
	}
	return backend
}

func TestNew(t *testing.T) {
	c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0))
	require.NotNil(t, c)
}

func TestBlockchainEvents(t *testing.T) {
	t.Run("register_unregister", func(t *testing.T) {
		c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0))
		blockImport := c.RegisterImportNotificationStream()
		require.NotNil(t, blockImport)
		_, ok := c.importNotificationChans[blockImport]
		require.True(t, ok)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for range blockImport {
			}
			wg.Done()
		}()

		everyImport := c.RegisterEveryImportNotificationStream()
		require.NotNil(t, everyImport)
		_, ok = c.everyImportNotificationChans[everyImport]
		require.True(t, ok)

		wg.Add(1)
		go func() {
			for range everyImport {
			}
			wg.Done()
		}()

		finality := c.RegisterFinalityNotificationStream()
		require.NotNil(t, finality)
		_, ok = c.finalityNotificationChans[finality]
		require.True(t, ok)

		wg.Add(1)
		go func() {
			for range finality {
			}
			wg.Done()
		}()

		c.UnregisterImportNotificationStream(blockImport)
		_, ok = c.importNotificationChans[blockImport]
		require.False(t, ok)

		c.UnregisterEveryImportNotificationStream(everyImport)
		_, ok = c.everyImportNotificationChans[everyImport]
		require.False(t, ok)

		c.UnregisterFinalityNotificationStream(finality)
		_, ok = c.finalityNotificationChans[finality]
		require.False(t, ok)

		wg.Wait()
	})

	t.Run("register_receive_block_import_unregister", func(t *testing.T) {
		c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0))
		blockImport := c.RegisterImportNotificationStream()
		require.NotNil(t, blockImport)
		_, ok := c.importNotificationChans[blockImport]
		require.True(t, ok)

		var blockImportNotifications []api.BlockImportNotification[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for notif := range blockImport {
				blockImportNotifications = append(blockImportNotifications, notif)
			}
			wg.Done()
		}()

		var everyImportNotifications []api.BlockImportNotification[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]
		everyImport := c.RegisterEveryImportNotificationStream()
		require.NotNil(t, everyImport)
		_, ok = c.everyImportNotificationChans[everyImport]
		require.True(t, ok)

		wg.Add(1)
		go func() {
			for notif := range everyImport {
				everyImportNotifications = append(everyImportNotifications, notif)
			}
			wg.Done()
		}()

		// sends to both
		c.notifyImported(&api.BlockImportNotification[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]{}, api.BothBlockImportNotificationAction, nil)
		// sends to import
		c.notifyImported(&api.BlockImportNotification[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]{}, api.RecentBlockImportNotificationAction, nil)
		// sends to every
		c.notifyImported(&api.BlockImportNotification[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]{}, api.EveryBlockImportNotificationAction, nil)

		c.UnregisterImportNotificationStream(blockImport)
		_, ok = c.importNotificationChans[blockImport]
		require.False(t, ok)

		c.UnregisterEveryImportNotificationStream(everyImport)
		_, ok = c.everyImportNotificationChans[everyImport]
		require.False(t, ok)

		wg.Wait()

		require.Len(t, blockImportNotifications, 2)
		require.Len(t, everyImportNotifications, 2)
	})

	t.Run("register_receive_block_import_storage_changes_unregister", func(t *testing.T) {
		c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0))
		wg := sync.WaitGroup{}

		topStorage := c.StorageChangesNotificationStream(nil, []api.ChildFilterKeys{})
		wg.Add(1)
		go func() {
			msg := <-topStorage.Chan()
			require.Len(t, msg.Changes, 1)
			require.Len(t, msg.ChildChanges, 0)
			wg.Done()
		}()

		childStorage := c.StorageChangesNotificationStream([]storage.StorageKey{}, []api.ChildFilterKeys{
			{
				Key:        storage.StorageKey("child0"),
				FilterKeys: []storage.StorageKey{storage.StorageKey("child0")},
			},
		})
		wg.Add(1)
		go func() {
			msg := <-childStorage.Chan()
			require.Len(t, msg.Changes, 0)
			require.Len(t, msg.ChildChanges, 1)
			wg.Done()
		}()

		wildCard := c.StorageChangesNotificationStream(nil, []api.ChildFilterKeys{
			{
				Key:        storage.StorageKey("child0"),
				FilterKeys: []storage.StorageKey{storage.StorageKey("child0")},
			},
		})
		wg.Add(1)
		go func() {
			msg := <-wildCard.Chan()
			require.Len(t, msg.Changes, 1)
			require.Len(t, msg.ChildChanges, 1)
			wg.Done()
		}()

		// sends to both
		c.notifyImported(
			&api.BlockImportNotification[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]{},
			api.BothBlockImportNotificationAction,
			&api.StorageChanges{
				StorageCollection: statemachine.StorageCollection{
					{statemachine.StorageKey("top0"), statemachine.StorageValue("top0")},
				},
				ChildStorageCollection: []struct {
					statemachine.StorageKey
					statemachine.StorageCollection
				}{
					{
						StorageKey: statemachine.StorageKey("child0"),
						StorageCollection: statemachine.StorageCollection{
							{statemachine.StorageKey("child0"), statemachine.StorageValue("child0")},
						},
					},
				},
			},
		)

		wg.Wait()
		topStorage.Drop()
		childStorage.Drop()

	})

	t.Run("register_receive_finality_unregister", func(t *testing.T) {
		c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0))
		finality := c.RegisterFinalityNotificationStream()
		require.NotNil(t, finality)
		_, ok := c.finalityNotificationChans[finality]
		require.True(t, ok)

		var finalityNotifications []api.FinalityNotification[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for notif := range finality {
				finalityNotifications = append(finalityNotifications, notif)
			}
			wg.Done()
		}()

		c.notifyFinalized(&api.FinalityNotification[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]{})

		c.UnregisterFinalityNotificationStream(finality)
		_, ok = c.finalityNotificationChans[finality]
		require.False(t, ok)

		wg.Wait()

		require.Len(t, finalityNotifications, 1)
	})
}
