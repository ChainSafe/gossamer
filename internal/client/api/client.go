// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package api

import (
	"github.com/ChainSafe/gossamer/internal/client/consensus"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

// ImportNotifications is a channel of block import events.
type ImportNotifications[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] chan<- BlockImportNotification[H, N, Header]

// FinalityNotifications is a channel of block finality notifications.
type FinalityNotifications[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] chan<- FinalityNotification[H, N, Header]

// BlockchainEvents is the source of blockchain events.
type BlockchainEvents[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] interface {
	// RegisterImportNotificationStream retrieves a channel of block import events.
	//
	// Not guaranteed to be fired for every imported block. Use
	// [RegisterEveryImportNotificationStream] if you want a notification of every imported block
	// regardless.
	//
	// The events for this notification stream are emitted:
	// - During initial sync process: if there is a re-org while importing blocks.
	// - After initial sync process: on every imported block, regardless of whether it is
	// the new best block or not, or if it causes a re-org or not.
	RegisterImportNotificationStream() ImportNotifications[H, N, Header]
	// UnregisterImportNotificationStream will unregister a registered channel.
	UnregisterImportNotificationStream(ImportNotifications[H, N, Header])

	// RegisterEveryImportNotificationStream retrieves a channel of block import events for every imported block.
	RegisterEveryImportNotificationStream() ImportNotifications[H, N, Header]
	// UnregisterEveryImportNotificationStream will unregister a registered channel.
	UnregisterEveryImportNotificationStream(ImportNotifications[H, N, Header])

	// RegisterFinalityNotificationStream will get a channel of finality notifications. Not guaranteed to be fired for
	//every finalized block.
	RegisterFinalityNotificationStream() FinalityNotifications[H, N, Header]
	// UnregisterFinalityNotificationStream will unregister a registered channel.
	UnregisterFinalityNotificationStream(FinalityNotifications[H, N, Header])

	// StorageChangesNotificationStream retrieves a storage changes event stream.
	// Passing nil for filterKeys subscribes to all storage changes.
	StorageChangesNotificationStream(
		filterKeys []storage.StorageKey,
		childFilterKeys []ChildFilterKeys,
	) StorageEventStream[H]
}

// AuxDataOperation is an operation to be performed on storage aux data.
// Key is the encoded data key.
// Value is the encoded optional data to write.
// If Value is nil, the key and the associated data are deleted from storage.
type AuxDataOperation struct {
	Key  []byte
	Data []byte
}

// AuxDataOperations is a slice of [AuxDataOperation] to be performed on storage aux data.
type AuxDataOperations []AuxDataOperation

// OnImportAction is a callback invoked before committing the operations created during block import.
// This gives the opportunity to perform auxiliary pre-commit actions and optionally
// enqueue further storage write operations to be atomically performed on commit.
type OnImportAction[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] func(BlockImportNotification[H, N, Header]) AuxDataOperations

// OnFinalityAction is a callback invoked before committing the operations created during block finalization.
// This gives the opportunity to perform auxiliary pre-commit actions and optionally
// enqueue further storage write operations to be atomically performed on commit.
type OnFinalityAction[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] func(FinalityNotification[H, N, Header]) AuxDataOperations

// PreCommitActions is the interface to perform auxiliary actions before committing a block import or
// finality operation.
type PreCommitActions[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] interface {
	RegisterImportAction(op OnImportAction[H, N, Header])     // Actions to be performed on block import.
	RegisterFinalityAction(op OnFinalityAction[H, N, Header]) // Actions to be performed on block finalization.
}

// Sends a message to the pinning-worker once dropped to unpin a block in the backend.
type unpinHandleInner[H runtime.Hash] struct {
	hash  H // Hash of the block pinned by this handle
	unpin func(message Unpin[H]) error
}

func (uhi unpinHandleInner[H]) Drop() {
	err := uhi.unpin(Unpin[H]{uhi.hash})
	if err != nil {
		logger.Debugf("Unable to unpin block with hash: %s, error: %v", uhi.hash, err)
	}
}

// UnpinWorkerMessage is the message that signals notification-based pinning actions to the pinning-worker.
// When the notification is dropped, an [Unpin] message should be sent to the worker.
type UnpinWorkerMessage[H runtime.Hash] interface {
	isUnpinWorkerMessage()
}

// AnnouncePin should be sent when a import or finality notification is created.
type AnnouncePin[H runtime.Hash] struct {
	Hash H
}

// Unpin should be sent when a import or finality notification is dropped.
type Unpin[H runtime.Hash] struct {
	Hash H
}

func (AnnouncePin[H]) isUnpinWorkerMessage() {}
func (Unpin[H]) isUnpinWorkerMessage()       {}

// UnpinHandle keeps a specific block pinned while the handle is alive.
// Once the last handle instance for a given block is dropped, the
// block is unpinned in the [Backend].
type UnpinHandle[H runtime.Hash] struct {
	unpinHandleInner[H]
}

// NewUnpinHandle is constructor for [UnpinHandle].
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

// BlockImportNotification is the summary of an imported block.
type BlockImportNotification[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] struct {
	Hash      H                     // Imported block header hash.
	Origin    consensus.BlockOrigin // Imported block origin.
	Header    Header                // Imported block header.
	IsNewBest bool                  // Is this the new best block.
	// TreeRoute from old best to new best. If nil, there was no re-org while importing.
	TreeRoute   *blockchain.TreeRoute[H, N]
	unpinHandle UnpinHandle[H] // Handle to unpin the block this notification is associated with.
}

// Drop will unpin the block from the backend.
func (bin BlockImportNotification[H, N, Header]) Drop() {
	bin.unpinHandle.Drop()
}

// NewBlockImportNotificationFromSummary is constructor of [BlockImportNotification] given an [ImportSummary].
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

// FinalityNotification is the summary of a finalized block.
type FinalityNotification[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] struct {
	Hash   H      // Finalized block header hash.
	Header Header // Finalized block header.
	// TreeRoute path from the old finalized to new finalized parent (implicitly finalized blocks).
	// This maps to the range [oldFinalized, ...newFinalized].
	TreeRoute   []H
	StaleHeads  []H            // Stale branches heads.
	unpinHandle UnpinHandle[H] // Handle to unpin the block this notification associated with.
}

// Drop will unpin the block from the backend.
func (fn FinalityNotification[H, N, Header]) Drop() {
	fn.unpinHandle.Drop()
}

// NewFinalityNotificationFromSummary is constructor of [FinalityNotification] given an [FinalizeSummary].
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
