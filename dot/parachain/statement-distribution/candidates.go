// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//nolint:unused
package statementdistribution

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type confirmedCandidate struct {
	receipt         parachaintypes.CommittedCandidateReceiptV2
	pvd             *parachaintypes.PersistedValidationData
	assignedGroup   parachaintypes.GroupIndex
	parentHash      common.Hash
	importableUnder map[common.Hash]struct{}
}
