// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package api

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	ptrie "github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/internal/primitives/trie/recorder"
	"github.com/ChainSafe/gossamer/internal/primitives/version"
)

// Something that provides a runtime api.
type ProvideRuntimeAPI[API any] interface {
	// Returns the runtime api.
	// The returned instance will keep track of modifications to the storage. Any successful call to an api function,
	// will commit its changes to an internal buffer. Otherwise, the modifications will be discarded. The modifications
	// will not be applied to the storage, even on a commit.
	RuntimeAPI() API
}

// Something that can be constructed to a runtime api.
type ConstructRuntimeApi[RuntimeAPI any] interface {
	// Construct an instance of the runtime api.
	ConstructRuntimeAPI() RuntimeAPI
}

// A type that records all accessed trie nodes and generates a proof out of it.
type ProofRecorder[H runtime.Hash] recorder.Recorder[H]

// Extends the runtime api implementation with some common functionality.
type ApiExt[
	N runtime.Number,
	E runtime.Extrinsic,
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	Backend statemachine.Backend[H, Hasher],
	Result any,
	Header runtime.Header[N, H],
] interface {
	// Execute the given closure inside a new transaction.
	// Depending on the outcome of the closure, the transaction is committed or rolled-back.
	// The internal result of the closure is returned afterwards.
	ExecuteInTransaction(
		call func(api ApiExt[N, E, H, Hasher, Backend, Result, Header]) runtime.TransactionOutcome[Result],
	) Result
	// Checks if the given api is implemented and versions match.
	HasAPI(atHash H) (bool, error)
	// Check if the given api is implemented and the version passes a predicate.
	HasAPIWith(atHash H, pred func(uint32) bool) (bool, error)
	// Returns the version of the given api.
	APIVersion(atHash H) (*uint32, error)
	// Start recording all accessed trie nodes for generating proofs.
	RecordProof()
	// Extract the recorded proof.
	// This stops the proof recording.
	// If RecordProof was not called before, this will return nil.
	ExtractProof() *ptrie.StorageProof
	// Returns the current active proof recorder.
	ProofRecorder() *ProofRecorder[H]
	// Convert the api object into the storage changes that were done while executing runtime
	// api functions.
	// After executing this function, all collected changes are reset.
	IntoStorageChanges(backend Backend, parentHash H) (overlayedchanges.StorageChanges[H, Hasher], error)
	// Set the call context for the current transaction.
	SetCallContext(callContext core.CallContext)
	// Register an [Extension] that will be accessible while executing a runtime api call.
	RegisterExtension(extension any)
	// Execute the given block
	ExecuteBlock(runtimeApiAtParam H, block runtime.Block[H, N, E, Header]) error
}

// pub trait Core {
type Core[H runtime.Hash] interface {
	/// Returns the version of the runtime.
	// fn version() -> RuntimeVersion;
	Version(hash H) (version.RuntimeVersion, error)
	// /// Execute the given block.
	// fn execute_block(block: Block);
	// /// Initialize a block with the given header.
	// #[changed_in(5)]
	// #[renamed("initialise_block", 2)]
	// fn initialize_block(header: &<Block as BlockT>::Header);
	// /// Initialize a block with the given header and return the runtime executive mode.
	// fn initialize_block(header: &<Block as BlockT>::Header) -> ExtrinsicInclusionMode;
}
