// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// TODO: https://github.com/ChainSafe/gossamer/issues/3537
// https://github.com/paritytech/polkadot-sdk/blob/d2fc1d7c91971e6e630a9db8cb627f8fdc91e8a4/polkadot/node/subsystem-util/src/backing_implicit_view.rs#L102 //nolint
type ImplicitView struct {
}

// Get the known, allowed relay-parents that are valid for parachain candidates
// which could be backed in a child of a given block for a given para ID.
//
// This is expressed as a contiguous slice of relay-chain block hashes which may
// include the provided block hash itself.
//
// If `para_id` is `None`, this returns all valid relay-parents across all paras
// for the leaf.
//
// `None` indicates that the block hash isn't part of the implicit view or that
// there are no known allowed relay parents.
//
// This always returns `Some` for active leaves or for blocks that previously
// were active leaves.
//
// This can return the empty slice, which indicates that no relay-parents are allowed
// for the para, e.g. if the para is not scheduled at the given block hash.
func (iview ImplicitView) KnownAllowedRelayParentsUnder(hash common.Hash,
	paraID parachaintypes.ParaID) common.Hash {

	return common.Hash{}
}
