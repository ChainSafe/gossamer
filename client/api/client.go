package api

import (
	"github.com/ChainSafe/gossamer/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/primitives/state-machine"
)

// / Type that implements `futures::Stream` of block import events.
// pub type ImportNotifications<Block> = TracingUnboundedReceiver<BlockImportNotification<Block>>;
type ImportNofications[H statemachine.HasherOut, N runtime.Number] chan<- BlockImportOperation[N, H]

// / A stream of block finality notifications.
// pub type FinalityNotifications<Block> = TracingUnboundedReceiver<FinalityNotification<Block>>;
type FinalityNotifiactions[H statemachine.HasherOut, N runtime.Number] chan<- FinalityNotification[H, N]

// / A source of blockchain events.
// pub trait BlockchainEvents<Block: BlockT> {
type BlockchainEvents[H statemachine.HasherOut, N runtime.Number] interface {
	/// Get block import event stream.
	///
	/// Not guaranteed to be fired for every imported block, only fired when the node
	/// has synced to the tip or there is a re-org. Use `every_import_notification_stream()`
	/// if you want a notification of every imported block regardless.
	// fn import_notification_stream(&self) -> ImportNotifications<Block>;
	ImportNotifications() ImportNofications[H, N]

	/// Get a stream of every imported block.
	// fn every_import_notification_stream(&self) -> ImportNotifications<Block>;
	EveryImportNotificationStream() ImportNofications[H, N]

	/// Get a stream of finality notifications. Not guaranteed to be fired for every
	/// finalized block.
	// fn finality_notification_stream(&self) -> FinalityNotifications<Block>;
	FinalityNotificationStream()

	/// Get storage changes event stream.
	///
	/// Passing `None` as `filter_keys` subscribes to all storage changes.
	// fn storage_changes_notification_stream(
	// 	&self,
	// 	filter_keys: Option<&[StorageKey]>,
	// 	child_filter_keys: Option<&[(StorageKey, Option<Vec<StorageKey>>)]>,
	// ) -> sp_blockchain::Result<StorageEventStream<Block::Hash>>;
	StorageChangesNotificationStream(filterKeys *[][]byte, childFilterKeys *[]struct {
		Key  []byte
		Keys *[][]byte
	}) (StorageEventStream[H], error)
}

// / Summary of a finalized block.
// pub struct FinalityNotification<Block: BlockT> {
type FinalityNotification[H statemachine.HasherOut, N runtime.Number] struct {
	/// Finalized block header hash.
	// pub hash: Block::Hash,
	Hash H

	/// Finalized block header.
	// pub header: Block::Header,
	Header runtime.Header[N, H]

	/// Path from the old finalized to new finalized parent (implicitly finalized blocks).
	///
	/// This maps to the range `(old_finalized, new_finalized)`.
	// pub tree_route: Arc<[Block::Hash]>,
	TreeRoute []H

	/// Stale branches heads.
	// pub stale_heads: Arc<[Block::Hash]>,
	StaleHeads []H

	/// Handle to unpin the block this notification is for
	// unpin_handle: UnpinHandle<Block>,
}
