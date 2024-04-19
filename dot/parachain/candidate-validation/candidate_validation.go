// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidate_validation

import (
	"context"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type CandidateValidation struct {
	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
}

func (cv *CandidateValidation) Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) {
}

func (*CandidateValidation) Name() parachaintypes.SubSystemName {
	return parachaintypes.CandidateValidation
}

func (*CandidateValidation) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	return nil
}

func (*CandidateValidation) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	return nil
}

func (*CandidateValidation) Stop() {}
