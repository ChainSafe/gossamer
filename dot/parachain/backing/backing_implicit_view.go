// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// NOTE: this is a temp file, will be a separate package for backing implicit view
package backing

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type ImplicitView interface {
	knownAllowedRelayParentsUnder(blockHash common.Hash, paraID *parachaintypes.ParaID) []common.Hash
}
