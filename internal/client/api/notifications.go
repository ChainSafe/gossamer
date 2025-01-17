// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package api

import (
	"github.com/ChainSafe/gossamer/internal/client/utils/pubsub"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

// StorageNotification is the message type delivered to subscribers.
type StorageNotification[H runtime.Hash] struct {
	Block            H // The hash of the block
	StorageChangeSet   // The set of changes
}

// StorageChange is a helper struct that contains a [storage.StorageKey] and [storage.StorageData].
// A nil for StorageData represents that the key should be deleted.
type StorageChange struct {
	storage.StorageKey
	storage.StorageData // can be nil
}

// StorageChildChange is a helper struct that contains a [storage.StorageKey] that represents the child key,
// and a changeset which is a slice of [StorageChange].
type StorageChildChange struct {
	storage.StorageKey
	ChangeSet []StorageChange
}

// StorageChangeset is a type that represents a storage changeset.
type StorageChangeSet struct {
	// changes: Arc<[(StorageKey, Option<StorageData>)]>,
	Changes []StorageChange
	// child_changes: Arc<[(StorageKey, Vec<(StorageKey, Option<StorageData>)>)]>,
	ChildChanges []StorageChildChange
	// filter: Keys,
	Filter Keys
	// child_filters: ChildKeys,
	ChildFilters ChildKeys
}

type Keys map[string]any                 // can be nil
type ChildKeys map[string]map[string]any // can be nil

// StorageNotifications manages storage listeners.
type StorageNotifications[H runtime.Hash] struct {
	*pubsub.Hub[SubscribeOp, SubscriberMessage[H], StorageNotification[H], *registry[H]]
}

// NewStorageNotifications is constructor for [StorageNotifications].
func NewStorageNotifications[H runtime.Hash]() StorageNotifications[H] {
	registry := newRegistry[H]()
	hub := pubsub.NewHub("mpsc_storage_notification_items", registry)
	return StorageNotifications[H]{
		Hub: hub,
	}
}

// Trigger notification to all listeners.
// Note the changes are going to be filtered by listener's filter key.
// In fact no event might be sent if clients are not interested in the changes.
func (s StorageNotifications[H]) Trigger(hash H, changeset []StorageChange, childChangeSet []StorageChildChange) {
	s.Hub.Send(SubscriberMessage[H]{
		Hash:           hash,
		ChangeSet:      changeset,
		ChildChangeSet: childChangeSet,
	})
}

// Listen will start listening for particular storage keys.
func (s StorageNotifications[H]) Listen(
	filterKeys []storage.StorageKey,
	filterChildKeys []ChildFilterKeys,
) StorageEventStream[H] {
	receiver := s.Hub.Subscribe(SubscribeOp{
		FilterKeys:      filterKeys,
		FilterChildKeys: filterChildKeys,
	})
	return StorageEventStream[H]{
		Receiver: receiver,
	}
}

// StorageEventStream is the receiving side of storage change events.
type StorageEventStream[H runtime.Hash] struct {
	*pubsub.Receiver[StorageNotification[H], *registry[H]]
}
