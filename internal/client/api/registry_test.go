// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package api

import (
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/stretchr/testify/require"
)

func Test_StorageNotifications(t *testing.T) {
	t.Run("triggering_change_should_notify_wildcard_listeners", func(t *testing.T) {
		notifications := NewStorageNotifications[hash.H256]()
		childFilter := ChildFilterKeys{
			Key:        []byte{4},
			FilterKeys: nil,
		}
		recv := notifications.Listen(nil, []ChildFilterKeys{childFilter})

		changeset := []StorageChange{
			{StorageKey: []byte{2}, StorageData: []byte{3}},
			{StorageKey: []byte{3}, StorageData: nil},
		}
		cChangeset1 := []StorageChange{
			{StorageKey: []byte{5}, StorageData: []byte{4}},
			{StorageKey: []byte{6}, StorageData: nil},
		}
		cChangeset := []StorageChildChange{
			{StorageKey: []byte{4}, ChangeSet: cChangeset1},
		}

		done := make(chan any)
		go func() {
			defer close(done)
			storageNotification := <-recv.Chan()
			require.Equal(t, hash.NewH256FromLowUint64BigEndian(1), storageNotification.Block)
			require.Equal(t, changeset, storageNotification.StorageChangeSet.Changes)
			require.Equal(t, cChangeset, storageNotification.StorageChangeSet.ChildChanges)

		}()

		notifications.Trigger(
			hash.NewH256FromLowUint64BigEndian(1),
			changeset,
			cChangeset,
		)

		<-done
		recv.Drop()
		notifications.Shutdown()
	})

	t.Run("should_only_notify_interested_listeners", func(t *testing.T) {
		notifications := NewStorageNotifications[hash.H256]()
		childFilter := ChildFilterKeys{
			Key:        []byte{4},
			FilterKeys: []storage.StorageKey{[]byte{5}},
		}
		recv1 := notifications.Listen([]storage.StorageKey{{1}}, nil)
		recv2 := notifications.Listen([]storage.StorageKey{{2}}, nil)
		recv3 := notifications.Listen([]storage.StorageKey{}, []ChildFilterKeys{childFilter})

		changeset := []StorageChange{
			{StorageKey: []byte{2}, StorageData: []byte{3}},
			{StorageKey: []byte{1}, StorageData: nil},
		}
		cChangeset1 := []StorageChange{
			{StorageKey: []byte{5}, StorageData: []byte{4}},
			{StorageKey: []byte{6}, StorageData: nil},
		}
		cChangeset := []StorageChildChange{
			{StorageKey: []byte{4}, ChangeSet: cChangeset1},
		}

		var (
			wg     sync.WaitGroup
			notif1 StorageNotification[hash.H256]
			notif2 StorageNotification[hash.H256]
			notif3 StorageNotification[hash.H256]
		)
		wg.Add(3)
		go func() {
			defer wg.Done()
			notif1 = <-recv1.Chan()
		}()
		go func() {
			defer wg.Done()
			notif2 = <-recv2.Chan()
		}()
		go func() {
			defer wg.Done()
			notif3 = <-recv3.Chan()
		}()

		notifications.Trigger(hash.NewH256FromLowUint64BigEndian(1), changeset, cChangeset)

		wg.Wait()

		require.Equal(t, hash.NewH256FromLowUint64BigEndian(1), notif1.Block)
		require.Equal(t, []StorageChange{{StorageKey: []byte{1}, StorageData: nil}}, notif1.StorageChangeSet.Changes)
		require.Equal(t, []StorageChildChange(nil), notif1.StorageChangeSet.ChildChanges)

		require.Equal(t, hash.NewH256FromLowUint64BigEndian(1), notif2.Block)
		require.Equal(t, []StorageChange{{StorageKey: []byte{2}, StorageData: []byte{3}}}, notif2.StorageChangeSet.Changes)
		require.Equal(t, []StorageChildChange(nil), notif2.StorageChangeSet.ChildChanges)

		require.Equal(t, hash.NewH256FromLowUint64BigEndian(1), notif3.Block)
		require.Equal(t, []StorageChange(nil), notif3.StorageChangeSet.Changes)
		require.Equal(t, []StorageChildChange{{StorageKey: []byte{4}, ChangeSet: []StorageChange{
			{StorageKey: []byte{5}, StorageData: []byte{4}},
		}}}, notif3.StorageChangeSet.ChildChanges)

		recv1.Drop()
		recv2.Drop()
		recv3.Drop()
		notifications.Shutdown()
	})

	t.Run("should_cleanup_subscribers_if_dropped", func(t *testing.T) {
		notifications := NewStorageNotifications[hash.H256]()
		{
			childFilter := ChildFilterKeys{
				Key:        []byte{4},
				FilterKeys: []storage.StorageKey{[]byte{5}},
			}
			recv1 := notifications.Listen([]storage.StorageKey{{1}}, nil)
			recv2 := notifications.Listen([]storage.StorageKey{{2}}, nil)
			recv3 := notifications.Listen(nil, nil)
			recv4 := notifications.Listen(nil, []ChildFilterKeys{childFilter})

			require.Equal(t, 2, len(notifications.Registry().listeners))
			require.Equal(t, 2, len(notifications.Registry().wildcardListeners))
			require.Equal(t, 1, len(notifications.Registry().childListeners))

			recv1.Drop()
			recv2.Drop()
			recv3.Drop()
			recv4.Drop()
		}

		changeset := []StorageChange{
			{StorageKey: []byte{2}, StorageData: []byte{3}},
			{StorageKey: []byte{1}, StorageData: nil},
		}
		cChangeset := []StorageChildChange{}
		notifications.Trigger(hash.NewH256FromLowUint64BigEndian(1), changeset, cChangeset)

		require.Equal(t, 0, len(notifications.Registry().listeners))
		require.Equal(t, 0, len(notifications.Registry().wildcardListeners))
		require.Equal(t, 0, len(notifications.Registry().childListeners))

		notifications.Shutdown()
	})

	t.Run("should_cleanup_subscriber_if_stream_is_dropped", func(t *testing.T) {
		notifications := NewStorageNotifications[hash.H256]()
		stream := notifications.Listen(nil, nil)
		require.Equal(t, 1, len(notifications.Registry().sinks))
		stream.Drop()
		require.Equal(t, 0, len(notifications.Registry().sinks))
	})

	t.Run("should_not_send_empty_subscriber", func(t *testing.T) {
		notifications := NewStorageNotifications[hash.H256]()
		recv := notifications.Listen(nil, nil)

		changeset := []StorageChange{}
		cChangeset := []StorageChildChange{}

		var notifCount int
		done := make(chan any)
		go func() {
			defer close(done)
			for range recv.Chan() {
				notifCount++
			}
		}()
		notifications.Trigger(hash.NewH256FromLowUint64BigEndian(1), changeset, cChangeset)
		recv.Drop()
		<-done
		require.Zero(t, notifCount)

		notifications.Shutdown()
	})
}
