// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

import (
	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type ValidateFromChainState struct {
	CandidateReceipt parachaintypes.CandidateReceipt
	RuntimeInstance  parachainruntime.RuntimeInstance
	PovRequestor     PoVRequestor
	Sender           chan any
}

type ValidateFromExhaustive struct{}

type PreCheck struct{}
