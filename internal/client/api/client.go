// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package api

import (
	"github.com/ChainSafe/gossamer/internal/client/consensus"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// List of operations to be performed on storage aux data.
// Key is the encoded data key.
// Value is the encoded optional data to write.
// If Value is nil, the key and the associated data are deleted from storage.
type AuxDataOperation struct {
	Key  []byte
	Data []byte
}
type AuxDataOperations []AuxDataOperation

// / Sends a message to the pinning-worker once dropped to unpin a block in the backend.
type unpinHandleInner[H runtime.Hash] struct {
	/// Hash of the block pinned by this handle
	// hash: Block::Hash,
	hash H
	// unpin_worker_sender: TracingUnboundedSender<UnpinWorkerMessage<Block>>,
	// unpinChan chan<- UnpinWorkerMessage[H]
	unpin func(message Unpin[H]) error
}

func (uhi unpinHandleInner[H]) Drop() {
	err := uhi.unpin(Unpin[H]{uhi.hash})
	if err != nil {
		logger.Debugf("Unable to unpin block with hash: %s, error: %v", uhi.hash, err)
	}
}

// / Message that signals notification-based pinning actions to the pinning-worker.
// /
// / When the notification is dropped, an `Unpin` message should be sent to the worker.
type UnpinWorkerMessage[H runtime.Hash] interface {
	isUnpinWorkerMessage()
}

// / Should be sent when a import or finality notification is created.
type AnnouncePin[H runtime.Hash] struct {
	Hash H
}

// / Should be sent when a import or finality notification is dropped.
type Unpin[H runtime.Hash] struct {
	Hash H
}

func (AnnouncePin[H]) isUnpinWorkerMessage() {}
func (Unpin[H]) isUnpinWorkerMessage()       {}

// / Keeps a specific block pinned while the handle is alive.
// / Once the last handle instance for a given block is dropped, the
// / block is unpinned in the [`Backend`](crate::backend::Backend::unpin_block).
type UnpinHandle[H runtime.Hash] struct {
	unpinHandleInner[H]
}

func NewUnpinHandle[H runtime.Hash](hash H, unpin func(message Unpin[H]) error) UnpinHandle[H] {
	return UnpinHandle[H]{
		unpinHandleInner[H]{
			hash:  hash,
			unpin: unpin,
		},
	}
}

func (up UnpinHandle[H]) Hash() H {
	return up.hash
}

// / Summary of an imported block
type BlockImportNotification[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] struct {
	// /// Imported block header hash.
	// pub hash: Block::Hash,
	Hash H
	// /// Imported block origin.
	// pub origin: BlockOrigin,
	Origin consensus.BlockOrigin
	// /// Imported block header.
	// pub header: Block::Header,
	Header Header
	// /// Is this the new best block.
	// pub is_new_best: bool,
	IsNewBest bool
	/// Tree route from old best to new best.
	///
	/// If `None`, there was no re-org while importing.
	TreeRoute *blockchain.TreeRoute[H, N]
	/// Handle to unpin the block this notification is for
	// unpin_handle: UnpinHandle<Block>,
	unpinHandle UnpinHandle[H]
}

func (bin BlockImportNotification[H, N, Header]) Drop() {
	bin.unpinHandle.Drop()
}

func NewBlockImportNotificationFromSummary[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
](summary ImportSummary[H, N, Header], unpin func(message Unpin[H]) error) *BlockImportNotification[H, N, Header] {
	return &BlockImportNotification[H, N, Header]{
		Hash:        summary.Hash,
		Origin:      summary.Origin,
		Header:      summary.Header,
		IsNewBest:   summary.IsNewBest,
		TreeRoute:   summary.TreeRoute,
		unpinHandle: NewUnpinHandle[H](summary.Hash, unpin),
	}
}

// / Summary of a finalized block.
// pub struct FinalityNotification<Block: BlockT> {
type FinalityNotification[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] struct {
	// /// Finalized block header hash.
	// pub hash: Block::Hash,
	Hash H
	// /// Finalized block header.
	// pub header: Block::Header,
	Header Header
	// /// Path from the old finalized to new finalized parent (implicitly finalized blocks).
	// ///
	// /// This maps to the range `(old_finalized, new_finalized)`.
	// pub tree_route: Arc<[Block::Hash]>,
	TreeRoute []H
	// /// Stale branches heads.
	// pub stale_heads: Arc<[Block::Hash]>,
	StaleHeads []H
	// /// Handle to unpin the block this notification is for
	// unpin_handle: UnpinHandle<Block>,
	// Note: maybe move the unpin logic to the client
	unpinHandle UnpinHandle[H]
}

func (fn FinalityNotification[H, N, Header]) Drop() {
	fn.unpinHandle.Drop()
}

func NewFinalityNotificationFromSummary[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
](summary FinalizeSummary[H, N, Header], unpin func(message Unpin[H]) error) *FinalityNotification[H, N, Header] {
	var hash H
	if len(summary.Finalized) > 0 {
		hash = summary.Finalized[len(summary.Finalized)-1]
	}
	return &FinalityNotification[H, N, Header]{
		Hash:        hash,
		Header:      summary.Header,
		TreeRoute:   summary.Finalized,
		StaleHeads:  summary.StateHeads,
		unpinHandle: NewUnpinHandle[H](hash, unpin),
	}
}
