// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

import (
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	state_machine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
)

type ImportResult interface {
	isImportResult()
}

// Auxiliary data associated with an imported block result.
type importedAux struct {
	// Only the header has been imported. Block body verification was skipped.
	HeaderOnly bool
	// Clear all pending justification requests.
	ClearJustificationRequests bool
	// Request a justification for the given block.
	NeedsJustification bool
	// Received a bad justification.
	BadJustification bool
	// Whether the block that was imported is the new best block.
	IsNewBest bool
}

type (
	// Block imported.
	Imported struct{ importedAux }
	// Already in the blockchain.
	AlreadyInChain struct{}
	// Block or parent is known to be bad.
	KnownBad struct{}
	// Block parent is not in the chain.
	UnknownParent struct{}
	// Parent state is missing.
	MissingState struct{}
)

func (Imported) isImportResult()       {}
func (AlreadyInChain) isImportResult() {}
func (KnownBad) isImportResult()       {}
func (UnknownParent) isImportResult()  {}
func (MissingState) isImportResult()   {}

type BlockImport[H runtime.Hash, N runtime.Number] interface {
	/// Check block preconditions.
	CheckBlock(block BlockCheckParams[H, N]) (ImportResult, error)
	/// Import a block.
	ImportBlock(block BlockImportParams[H, N]) (ImportResult, error)
}

// Data required to check validity of a Block.
type BlockCheckParams[H runtime.Hash, N runtime.Number] struct {
	// Hash of the block that we verify.
	Hash H
	// Block number of the block that we verify.
	Number N
	// Parent hash of the block that we verify.
	ParentHash H
	// Allow importing the block skipping state verification if parent state is missing.
	AllowMissingState bool
	// Allow importing the block if parent block is missing.
	AllowMissingParent bool
	// Re-validate existing block.
	ImportExisting bool
}

// Imported state data. A vector of key-value pairs that should form a trie.
type ImportedState[H runtime.Hash] struct {
	// Target block hash.
	Block H
	// State keys and values.
	State state_machine.KeyValueStates
}

// Precomputed storage.
type StorageChanges interface {
	isStorageChanges()
}

type (
	// Changes coming from block execution.
	Changes[H runtime.Hash, Hasher runtime.Hasher[H]] overlayedchanges.StorageChanges[H, Hasher]
	// Whole new state.
	Import[H runtime.Hash] ImportedState[H]
)

func (Changes[H, Hasher]) isStorageChanges() {}
func (Import[H]) isStorageChanges()          {}

// Defines how a new state is computed for a given imported block.
type StateAction interface {
	isStateAction()
}

type (
	// Apply precomputed changes coming from block execution or state sync.
	ApplyChanges struct{}
)

// Fork choice strategy.
type ForkChoiceStrategy interface {
	isForkChoiceStrategy()
}

type (
	// Longest chain fork choice.
	LongestChain struct{}
	// Custom fork choice rule, where true indicates the new block should be the best block.
	Custom bool
)

// Data required to import a Block.
type BlockImportParams[H runtime.Hash, N runtime.Number] struct {
	/// Origin of the Block
	Origin common.BlockOrigin
	/// The header, without consensus post-digests applied. This should be in the same
	/// state as it comes out of the runtime.
	///
	/// Consensus engines which alter the header (by adding post-runtime digests)
	/// should strip those off in the initial verification process and pass them
	/// via the `post_digests` field. During block authorship, they should
	/// not be pushed to the header directly.
	///
	/// The reason for this distinction is so the header can be directly
	/// re-executed in a runtime that checks digest equivalence -- the
	/// post-runtime digests are pushed back on after.
	Header runtime.Header[N, H]
	/// Justification(s) provided for this block from the outside.
	Justifications *runtime.Justifications
	/// Digest items that have been added after the runtime for external
	/// work, like a consensus signature.
	PostDigests []runtime.DigestItem
	/// The body of the block.
	Body *[]runtime.Extrinsic
	/// Indexed transaction body of the block.
	IndexedBody *[][]byte
	/// Specify how the new state is computed.
	StateAction StateAction
	/// Is this block finalized already?
	Finalized bool
	/// Intermediate values that are interpreted by block importers. Each block importer,
	/// upon handling a value, removes it from the intermediate list. The final block importer
	/// rejects block import if there are still intermediate values that remain unhandled.
	Intermediates map[string]any
	/// Auxiliary consensus data produced by the block.
	/// Contains a list of key-value pairs. If values are `nil`, the keys will be deleted. These
	/// changes will be applied to `AuxStore` database all as one batch, which is more efficient
	/// than updating `AuxStore` directly.
	Auxiliary overlayedchanges.StorageCollection
	/// Fork choice strategy of this import. This should only be set by a
	/// synchronous import, otherwise it may race against other imports.
	/// `nil` indicates that the current verifier or importer cannot yet
	/// determine the fork choice value, and it expects subsequent importer
	/// to modify it. If `nil` is passed all the way down to bottom block
	/// importer, the import fails with an `IncompletePipeline` error.
	ForkChoice *ForkChoiceStrategy
	/// Re-validate existing block.
	ImportExisting bool
	/// Whether to create "block gap" in case this block doesn't have parent.
	CreateGap bool
	/// Cached full header hash (with post-digests applied).
	PostHash *H
}
