// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"golang.org/x/exp/constraints"
)

type DummyMockInterface[H constraints.Ordered] interface {
	Get(input H) H
}

// Telemetry TODO issue #3474
type Telemetry interface{}

/*
	Following is from api/backend
*/

type Key []byte

type KeyValue struct {
	Key   Key
	Value []byte
}

// AuxStore is part of the substrate backend.
// Provides access to an auxiliary database.
//
// This is a simple global database not aware of forks. Can be used for storing auxiliary
// information like total block weight/difficulty for fork resolution purposes as a common use
// case.
// TODO should this just be in Backend?
type AuxStore interface {
	// Insert auxiliary data into key-Value store.
	//
	// Deletions occur after insertions.
	Insert(insert []KeyValue, delete []Key) error
	// Get Query auxiliary data from key-Value store.
	Get(key Key) (*[]byte, error)
}

// Client backend.
//
// Manages the data layer.
//
// # State Pruning
//
// While an object from `state_at` is alive, the state
// should not be pruned. The backend should internally reference-count
// its state objects.
//
// The same applies for live `BlockImportOperation`s: while an import operation building on a
// parent `P` is alive, the state for `P` should not be pruned.
//
// # Block Pruning
//
// Users can pin blocks in memory by calling `pin_block`. When
// a block would be pruned, its value is kept in an in-memory cache
// until it is unpinned via `unpin_block`.
//
// While a block is pinned, its state is also preserved.
//
// The backend should internally reference count the number of pin / unpin calls.
type Backend[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N], B BlockchainBackend[Hash, N, H]] interface {
	AuxStore
	Blockchain() B
}

/*
	Following is from primatives/blockchain
*/

// Blockchain database header backend. Does not perform any validation.
// primatives/blockchains/src/backend
type HeaderBackend[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N]] interface {
	// Header Get block header. Returns None if block is not found.
	Header(hash Hash) (*H, error)
	// Get blockchain info.
	Info() Info[N]
	// This takes an enum blockID, but for now just using block Number N
	ExpectBlockHashFromID(id N) (Hash, error)
	ExpectHeader(hash Hash) (H, error)
}

type Info[N constraints.Unsigned] struct {
	FinalizedNumber N
}

// Blockchain database backend. Does not perform any validation.
// pub trait Backend<Block: BlockT>:HeaderBackend<Block> + HeaderMetadata<Block, Error = Error
// primatives/blockchains/src/backend
type BlockchainBackend[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N]] interface {
	HeaderBackend[Hash, N, H]
	Justifications(hash Hash) Justifications
}

/*
	Following is from primatives/runtime
*/

type Header[Hash constraints.Ordered, N constraints.Unsigned] interface {
	ParentHash() Hash
	Hash() Hash
	Number() N
}

type ConsensusEngineID [4]byte

// Justification An abstraction over justification for a block's validity under a consensus algorithm.
//
// Essentially a finality proof. The exact formulation will vary between consensus
// algorithms. In the case where there are multiple valid proofs, inclusion within
// the block itself would allow swapping justifications to change the block's hash
// (and thus fork the chain). Sending a `Justification` alongside a block instead
// bypasses this problem.
//
// Each justification is provided as an encoded blob, and is tagged with an ID
// to identify the consensus engine that generated the proof (we might have
// multiple justifications from different engines for the same block).
type Justification struct {
	EngineID             ConsensusEngineID
	EncodedJustification []byte
}

type Justifications []Justification

// IntoJustification Return a copy of the encoded justification for the given consensus
// engine, if it exists
func (j Justifications) IntoJustification(engineID ConsensusEngineID) *[]byte {
	for _, justification := range j {
		if justification.EngineID == engineID {
			return &justification.EncodedJustification
		}
	}
	return nil
}

/*
	Following is from test-utils/client
*/

// TODO Need interface as well I believe, should define here or in interfaces file?
// Investigate if we need interface here or just struct

type ClientBuilder[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N], B BlockchainBackend[Hash, N, H], BE Backend[Hash, N, H, B]] interface {
	Backend() BE // TODO I think this should be BE, verify
}

// TestClientBuilder A builder for creating a test client instance
type TestClientBuilder[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N], B BlockchainBackend[Hash, N, H], BE Backend[Hash, N, H, B]] struct {
	backend BE // TODO what type is backend
}

// Backend Give access to the underlying backend of these clients
func (tcb TestClientBuilder[Hash, N, H, B, BE]) Backend() BE {
	return tcb.backend
}

/*
	Following is from test-utils/runtime/client
*/

// TODO embed interface

// TestClientBuilderExt is an extension of client.TestClientBuilder
type TestClientBuilderExt[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N], B BlockchainBackend[Hash, N, H], BE Backend[Hash, N, H, B]] interface {
	Build() Client[Hash]
}

// DummyClientBuilderExt is a dummy struct that contains a clientBuilder and fulfills the TestClientBuilderExt interface
type DummyClientBuilderExt[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N], B BlockchainBackend[Hash, N, H], BE Backend[Hash, N, H, B]] struct {
	ClientBuilder ClientBuilder[Hash, N, H, B, BE]
}

// Build the test client.
func (cle DummyClientBuilderExt[Hash, N, H, B, BE]) Build() Client[Hash] {
	panic("unimplemented")
}

// NewTestClientBuilderExt constructs a new DummyClientBuilderExt
func NewTestClientBuilderExt[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N], B BlockchainBackend[Hash, N, H], BE Backend[Hash, N, H, B]]() DummyClientBuilderExt[Hash, N, H, B, BE] {
	return DummyClientBuilderExt[Hash, N, H, B, BE]{
		ClientBuilder: TestClientBuilder[Hash, N, H, B, BE]{},
	}
}

/*
	Following is from test-utils
*/

// TODO need to figure out the funcs/where in substrate it is
type Client[Hash constraints.Ordered] interface {
	// Think this is in client/service/src/client/client.rs
	// Returns a BlockBuilder
	NewBlock() types.Block
	// Need to build origin enum, also find where client is in substrate impl
	Import(origin BlockOrigin, block types.Block)
	// This seems to be at client/db/src/lib.rs
	FinalizeBlock(hash Hash, justification *Justification)
}

type TestClient[Hash constraints.Ordered] struct{}

func NewTestClient[Hash constraints.Ordered]() TestClient[Hash] {
	return TestClient[Hash]{}
}
