// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// NOTE: this is a temp file, will be a separate package for backing implicit view
package backing

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// ImplicitView handles the implicit view of the relay chain derived from the immediate/explicit view,
// which is composed of active leaves, and the minimum relay-parents allowed for candidates of various
// parachains at those leaves
type ImplicitView interface {
	// Get the known, allowed relay-parents that are valid for parachain candidates
	// which could be backed in a child of a given block for a given para ID.
	//
	// This is expressed as a contiguous slice of relay-chain block hashes which may
	// include the provided block hash itself.
	//
	// If paraID is nil, return all valid relay-parents across all parachains for the leaf.
	KnownAllowedRelayParentsUnder(blockHash common.Hash, paraID *parachaintypes.ParaID) []common.Hash
	// Get active leaves in the view
	Leaves() []common.Hash
	// Activate a leaf in the view.
	// This will request the minimum relay parents the leaf and will load headers in the
	// ancestry of the leaf as needed. These are the 'implicit ancestors' of the leaf.
	//
	// To maximise reuse of outdated leaves, it's best to activate new leaves before
	// deactivating old ones.
	ActivateLeaf(leafHash common.Hash) error
	// Deactivate a leaf in the view. This prunes any outdated implicit ancestors as well.
	// Returns hashes of blocks pruned from storage.
	deactivateLeaf(leafHash common.Hash) []common.Hash
	// Get all allowed relay-parents in the view with no particular order.
	//
	// Important: not all blocks are guaranteed to be allowed for some leaves, it may
	// happen that a block info is only kept in the view storage because of a retaining rule.
	AllAllowedRelayParents() []common.Hash
}
