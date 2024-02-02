package api

import "golang.org/x/exp/constraints"

// / A type of a message delivered to the subscribers
// pub struct StorageNotification<Hash> {
type StorageNotification[H any] struct {
	/// The hash of the block
	// 	pub block: Hash,
	Block H

	// /// The set of changes
	// pub changes: StorageChangeSet,
	Changes StorageChangeSet
}

// / Storage change set
// pub struct StorageChangeSet {
type StorageChangeSet struct {
	// changes: Arc<[(StorageKey, Option<StorageData>)]>,
	Changes []struct {
		StorageKey  []byte
		StorageData *[]byte
	}
	// child_changes: Arc<[(StorageKey, Vec<(StorageKey, Option<StorageData>)>)]>,
	ChildChanges []struct {
		StorageKey []byte
		KeyData    []struct {
			StorageKey  []byte
			StorageData *[]byte
		}
	}
	// filter: Keys,
	filter Keys[string]
	// child_filters: ChildKeys,
	childFilters ChildKeys[string]
}

// / Manages storage listeners.
// pub struct StorageNotifications<Block: BlockT>(Hub<StorageNotification<Block::Hash>, Registry>);
type StorageNotifications[H any] chan StorageNotification[H]

// / Type that implements `futures::Stream` of storage change events.
// pub struct StorageEventStream<H>(Receiver<StorageNotification<H>, Registry>);
type StorageEventStream[H any] chan<- StorageNotification[H]

// type Keys = Option<HashSet<StorageKey>>;
type Keys[H constraints.Ordered] *map[H]bool

// type ChildKeys = Option<HashMap<StorageKey, Option<HashSet<StorageKey>>>>;
type ChildKeys[H constraints.Ordered] *map[H]*map[H]bool
