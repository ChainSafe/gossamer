package api

import (
	"github.com/ChainSafe/gossamer/internal/client/consensus"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
)

// / Type that implements `futures::Stream` of block import events.
// pub type ImportNotifications<Block> = TracingUnboundedReceiver<BlockImportNotification<Block>>;
type ImportNofications[H runtime.Hash, N runtime.Number, T statemachine.Transaction] chan<- BlockImportOperation[N, H, T]

// / A stream of block finality notifications.
// pub type FinalityNotifications<Block> = TracingUnboundedReceiver<FinalityNotification<Block>>;
type FinalityNotifications[H runtime.Hash, N runtime.Number] chan<- FinalityNotification[H, N]

// / A source of blockchain events.
// pub trait BlockchainEvents<Block: BlockT> {
type BlockchainEvents[H runtime.Hash, N runtime.Number, T statemachine.Transaction] interface {
	/// Get block import event stream.
	///
	/// Not guaranteed to be fired for every imported block, only fired when the node
	/// has synced to the tip or there is a re-org. Use `every_import_notification_stream()`
	/// if you want a notification of every imported block regardless.
	// fn import_notification_stream(&self) -> ImportNotifications<Block>;
	ImportNotifications() ImportNofications[H, N, T]

	/// Get a stream of every imported block.
	// fn every_import_notification_stream(&self) -> ImportNotifications<Block>;
	EveryImportNotificationStream() ImportNofications[H, N, T]

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

// / List of operations to be performed on storage aux data.
// / First tuple element is the encoded data key.
// / Second tuple element is the encoded optional data to write.
// / If `None`, the key and the associated data are deleted from storage.
// pub type AuxDataOperations = Vec<(Vec<u8>, Option<Vec<u8>>)>;
type AuxDataOperation struct {
	Key  []byte
	Data []byte
}
type AuxDataOperations []AuxDataOperation

// / Callback invoked before committing the operations created during block import.
// / This gives the opportunity to perform auxiliary pre-commit actions and optionally
// / enqueue further storage write operations to be atomically performed on commit.
// pub type OnImportAction<Block> =
//
//	Box<dyn (Fn(&BlockImportNotification<Block>) -> AuxDataOperations) + Send>;
type OnImportAction[H runtime.Hash, N runtime.Number] func(BlockImportNotification[H, N]) AuxDataOperations

// / Callback invoked before committing the operations created during block finalization.
// / This gives the opportunity to perform auxiliary pre-commit actions and optionally
// / enqueue further storage write operations to be atomically performed on commit.
// pub type OnFinalityAction<Block> =
//
//	Box<dyn (Fn(&FinalityNotification<Block>) -> AuxDataOperations) + Send>;
type OnFinalityAction[H runtime.Hash, N runtime.Number] func(FinalityNotification[H, N]) AuxDataOperations

// / Summary of an imported block
// #[derive(Clone, Debug)]
// pub struct BlockImportNotification<Block: BlockT> {
type BlockImportNotification[H runtime.Hash, N runtime.Number] struct {
	// /// Imported block header hash.
	// pub hash: Block::Hash,
	Hash H
	// /// Imported block origin.
	// pub origin: BlockOrigin,
	Origin consensus.BlockOrigin
	// /// Imported block header.
	// pub header: Block::Header,
	Header runtime.Header[N, H]
	// /// Is this the new best block.
	// pub is_new_best: bool,
	IsNewBest bool
	// /// Tree route from old best to new best parent.
	// ///
	// /// If `None`, there was no re-org while importing.
	// pub tree_route: Option<Arc<sp_blockchain::TreeRoute<Block>>>,
	TreeRoute *blockchain.TreeRoute[H, N]
	// /// Handle to unpin the block this notification is for
	// unpin_handle: UnpinHandle<Block>,
}

// / Summary of a finalized block.
// pub struct FinalityNotification<Block: BlockT> {
type FinalityNotification[H runtime.Hash, N runtime.Number] struct {
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

// / Create finality notification from finality summary.
func NewFinalityNotificationFromSummary[H runtime.Hash, N runtime.Number](
	summary FinalizeSummary[N, H],
) FinalityNotification[H, N] {
	var hash H
	if len(summary.Finalized) > 0 {
		hash = summary.Finalized[len(summary.Finalized)-1]
	}
	return FinalityNotification[H, N]{
		Hash:       hash,
		Header:     summary.Header,
		TreeRoute:  summary.Finalized,
		StaleHeads: summary.StaleHeads,
	}
}

// / Create finality notification from finality summary.
func NewBlockImportNotificationFromSummary[H runtime.Hash, N runtime.Number](
	summary ImportSummary[N, H],
) BlockImportNotification[H, N] {
	hash := summary.Hash
	return BlockImportNotification[H, N]{
		Hash:      hash,
		Origin:    summary.Origin,
		Header:    summary.Header,
		IsNewBest: summary.IsNewBest,
		TreeRoute: &summary.TreeRoute,
	}
}
