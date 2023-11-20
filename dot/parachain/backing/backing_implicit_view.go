// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// NOTE: this is a temp file, will be a separate package for backing implicit view
package backing

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// type ImplicitView struct {
// }

// func (view *ImplicitView) knownAllowedRelayParents(blockHash common.Hash, paraID *parachaintypes.ParaID) []common.Hash {
// 	// todo: implement this by referring
// 	// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/polkadot/node/subsystem-util/src/backing_implicit_view.rs#L217-L227 //nolint:lll
// 	return nil
// }

type ImplicitView interface {
	knownAllowedRelayParentsUnder(blockHash common.Hash, paraID *parachaintypes.ParaID) []common.Hash
}
