// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package chainspec

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/executor"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

var ErrMissingRuntime = errors.New("runtime missing from initial storage, could not read state version")

func ResolveStateVersionFromWasm[Hasher runtime.Hasher[H], H runtime.Hash](
	storage storage.Storage,
	executor executor.RuntimeVersionOf,
) (storage.StateVersion, error) {
	panic("unimpl")
}

// ConstructGenesisBlock creates a genesis block, given the initial storage.
func ConstructGenesisBlock[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	stateRoot H,
	stateVersion storage.StateVersion,
) runtime.Block[H, N, E, Header] {
	extrinsicsRoot := (*(new(Hasher))).TrieRoot(
		make([]runtime.KeyValue, 0),
		stateVersion,
	)

	var parentHash H
	var digest runtime.Digest
	var extrinsics []E
	header := generic.NewHeader[N, H, Hasher](0, extrinsicsRoot, stateRoot, parentHash, digest)
	block := generic.NewBlock[Hasher, E, N, H, Header](
		any(header).(Header),
		extrinsics,
	)
	return block
}

// BuildGenesisBlock is the interface for building the genesis block.
type BuildGenesisBlock[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] interface {
	// Returns the built genesis block along with the block import operation
	// after setting the genesis storage.
	BuildGenesisBlock(runtime.Block[H, N, E, Header], api.BlockImportOperation[N, H, Hasher, Header, E])
}

// Default genesis block builder in Substrate.
type GenesisBlockBuilder[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	gensisStorage      storage.Storage
	commitGenesisState bool
	backend            api.Backend[H, N, Hasher, Header, E]
	executor           executor.RuntimeVersionOf
}

// Constructs a new instance of [GenesisBlockBuilder].
func NewGenesisBlockBuilder[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	buildGenesisStorage runtime.BuildStorage,
	commitGenesisState bool,
	backend api.Backend[H, N, Hasher, Header, E],
	executor executor.RuntimeVersionOf,
) (*GenesisBlockBuilder[H, N, Hasher, Header, E], error) {
	genesisStorage, err := buildGenesisStorage.BuildStorage()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", blockchain.ErrStorage, err)
	}
	return NewGenesisBlockBuilderWithStorage(
		genesisStorage,
		commitGenesisState,
		backend,
		executor,
	), nil
}

// Constructs a new instance of [GenesisBlockBuilder] using provided storage.
func NewGenesisBlockBuilderWithStorage[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	genesisStorage storage.Storage,
	commitGenesisState bool,
	backend api.Backend[H, N, Hasher, Header, E],
	executor executor.RuntimeVersionOf,
) *GenesisBlockBuilder[H, N, Hasher, Header, E] {
	return &GenesisBlockBuilder[H, N, Hasher, Header, E]{
		gensisStorage:      genesisStorage,
		commitGenesisState: commitGenesisState,
		backend:            backend,
		executor:           executor,
	}
}

func (g *GenesisBlockBuilder[H, N, Hasher, Header, E]) BuildGenesisBlock() (runtime.Block[H, N, E, Header], api.BlockImportOperation[N, H, Hasher, Header, E], error) {
	genesisStateVersion, err := ResolveStateVersionFromWasm[Hasher, H](g.gensisStorage, g.executor)
	if err != nil {
		return nil, nil, err
	}
	op, err := g.backend.BeginOperation()
	stateRoot, err := op.SetGenesisState(g.gensisStorage, g.commitGenesisState, genesisStateVersion)
	if err != nil {
		return nil, nil, err
	}
	genesisBlock := ConstructGenesisBlock[H, N, Hasher, Header, E](stateRoot, genesisStateVersion)
	return genesisBlock, op, nil
}
