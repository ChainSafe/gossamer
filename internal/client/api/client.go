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
	unpinHandle any
}

func NewBlockImportNotificationFromSummary[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
](summary ImportSummary[H, N, Header]) *BlockImportNotification[H, N, Header] {
	return &BlockImportNotification[H, N, Header]{
		Hash:      summary.Hash,
		Origin:    summary.Origin,
		Header:    summary.Header,
		IsNewBest: summary.IsNewBest,
		TreeRoute: summary.TreeRoute,
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
	unpinHandle any
}

func NewFinalityNotificationFromSummary[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
](summary FinalizeSummary[H, N, Header]) *FinalityNotification[H, N, Header] {
	var hash H
	if len(summary.Finalized) > 0 {
		hash = summary.Finalized[len(summary.Finalized)-1]
	}
	return &FinalityNotification[H, N, Header]{
		Hash:       hash,
		Header:     summary.Header,
		TreeRoute:  summary.Finalized,
		StaleHeads: summary.StateHeads,
	}
}
