// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"golang.org/x/exp/constraints"
)

// ForkTree A Roots data structure that stores several Children across multiple branches.
//
// Top-level branches are called Roots. The Roots has functionality for
// finalizing Children, which means that node is traversed, and all competing
// branches are pruned. It also guarantees that Children in the Roots are finalized
// in order. Each node is uniquely identified by its hash but can be ordered by
// its number. In order to build the Roots an external function must be provided
// when interacting with the Roots to establish a node's ancestry.
type ForkTree[H comparable, N constraints.Unsigned] interface {
	Import(hash H, number N, change PendingChange[H, N], isDescendentOf IsDescendentOf[H]) (bool, error)
	Roots() []*PendingChangeNode[H, N]
	FinalizeAnyWithDescendentIf(hash *H, number N, isDescendentOf IsDescendentOf[H], predicate func(*PendingChange[H, N]) bool) (*bool, error)
	FinalizeWithDescendentIf(hash *H, number N, isDescendentOf IsDescendentOf[H], predicate func(*PendingChange[H, N]) bool) (*FinalizationResult[H, N], error)
	DrainFilter()

	// GetPreOrder Implemented this as part of interface, but can remove if we want since it's not part of the substrate interface
	GetPreOrder() []PendingChange[H, N]
}

// PublicKey interface
type PublicKey interface {
	Verify(msg, sig []byte) (bool, error)
	Encode() []byte
	Decode([]byte) error
	Address() string
	Hex() string
}

// AuxStore is part of the substrate backend.
// Provides access to an auxiliary database.
//
// This is a simple global database not aware of forks. Can be used for storing auxiliary
// information like total block weight/difficulty for fork resolution purposes as a common use
// case.
type AuxStore interface {
	// InsertAux Insert auxiliary data into key-Value store.
	//
	// Deletions occur after insertions.
	InsertAux(insert map[string][]byte, deleted []string) error
	// GetAux Query auxiliary data from key-Value store.
	GetAux(key []byte) *[]byte
}
