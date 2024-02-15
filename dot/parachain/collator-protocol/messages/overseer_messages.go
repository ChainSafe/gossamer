// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package messages

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type CollateOn parachaintypes.ParaID

type DistributeCollation struct {
	CandidateReceipt parachaintypes.CandidateReceipt
	PoV              parachaintypes.PoV
}

type ReportCollator parachaintypes.CollatorID

type NetworkBridgeUpdate struct {
	// TODO: not quite sure if we would need this or something similar to this
}

// Seconded represents that the candidate we recommended to be seconded was validated
// successfully.
type Seconded struct {
	Parent common.Hash
	Stmt   parachaintypes.UncheckedSignedFullStatement
}

// Backed message indicates that the candidate received enough validity votes from the backing group.
type Backed struct {
	ParaID parachaintypes.ParaID
	// Hash of the para head generated by candidate
	ParaHead common.Hash
}

func (b Backed) String() string {
	return fmt.Sprintf("para_id: %d,_para_head: %s", b.ParaID, b.ParaHead.String())
}

// Invalid represents an invalid candidata.
// We recommended a particular candidate to be seconded, but it was invalid; penalise the collator.
type Invalid struct {
	Parent           common.Hash
	CandidateReceipt parachaintypes.CandidateReceipt
}
