package api

import (
	"github.com/ChainSafe/gossamer/internal/client/utils/pubsub"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

// / A type of a message delivered to the subscribers
type StorageNotification[H runtime.Hash] struct {
	/// The hash of the block
	// pub block: Hash,
	Block H

	/// The set of changes
	// pub changes: StorageChangeSet,
	StorageChangeSet
}

type Change struct {
	Key   storage.StorageKey
	Value storage.StorageData // can be nil
}

type ChildChange struct {
	StorageKey storage.StorageKey
	ChangeSet  []Change
}

// / Storage change set
// pub struct StorageChangeSet {
type StorageChangeSet struct {
	// changes: Arc<[(StorageKey, Option<StorageData>)]>,
	Changes []Change
	// child_changes: Arc<[(StorageKey, Vec<(StorageKey, Option<StorageData>)>)]>,
	ChildChanges []ChildChange
	// filter: Keys,
	Filter Keys
	// child_filters: ChildKeys,
	ChildFilters ChildKeys
}

type Keys map[string]any                 // can be nil
type ChildKeys map[string]map[string]any // can be nil

// / Manages storage listeners.
// pub struct StorageNotifications<Block: BlockT>(Hub<StorageNotification<Block::Hash>, Registry>);
type StorageNotifications[H runtime.Hash] struct {
	*pubsub.Hub[SubscribeOp, Message[H], StorageNotification[H], *registry[H]]
}

// / Initialize a new StorageNotifications
func NewStorageNotifications[H runtime.Hash]() StorageNotifications[H] {
	registry := newRegistry[H]()
	hub := pubsub.NewHub("mpsc_storage_notification_items", registry)
	return StorageNotifications[H]{
		Hub: hub,
	}
}

// / Trigger notification to all listeners.
// /
// / Note the changes are going to be filtered by listener's filter key.
// / In fact no event might be sent if clients are not interested in the changes.
func (s StorageNotifications[H]) Trigger(hash H, changeset []Change, childChangeSet []ChildChange) {
	s.Hub.Send(Message[H]{
		Hash:           hash,
		ChangeSet:      changeset,
		ChildChangeSet: childChangeSet,
	})
}

// / Start listening for particular storage keys.
func (s StorageNotifications[H]) Listen(filterKeys []storage.StorageKey, filterChildKeys []ChildFilterKeys) StorageEventStream[H] {
	receiver := s.Hub.Subscribe(SubscribeOp{
		FilterKeys:      filterKeys,
		FilterChildKeys: filterChildKeys,
	})
	return StorageEventStream[H]{
		Receiver: receiver,
	}
}

// / Type that implements `futures::Stream` of storage change events.
// pub struct StorageEventStream<H>(Receiver<StorageNotification<H>, Registry>);
type StorageEventStream[H runtime.Hash] struct {
	*pubsub.Receiver[StorageNotification[H], *registry[H]]
}
