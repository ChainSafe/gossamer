// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import "github.com/ChainSafe/gossamer/lib/common"

// AfG ("Al's Finality Gadget") is synonymous with GRANDPA.

var (
	_ Message = (*AfgReceivedPrecommitTM)(nil)
	_ Message = (*AfgReceivedPrevoteTM)(nil)
	_ Message = (*AfgReceivedCommitTM)(nil)
)

type afgReceivedTM struct {
	TargetHash   common.Hash `json:"target_hash"`
	TargetNumber string      `json:"target_number"`
	Voter        string      `json:"voter"`
}

// AfgReceivedPrecommitTM holds `afg.received_precommit` telemetry message which is
// supposed to be sent when grandpa client receives a precommit.
type AfgReceivedPrecommitTM afgReceivedTM

// NewAfgReceivedPrecommitTM gets a new AfgReceivedPrecommitTM struct.
func NewAfgReceivedPrecommitTM(targetHash common.Hash, targetNumber, voter string) *AfgReceivedPrecommitTM {
	return &AfgReceivedPrecommitTM{
		TargetHash:   targetHash,
		TargetNumber: targetNumber,
		Voter:        voter,
	}
}

func (AfgReceivedPrecommitTM) messageType() string {
	return afgReceivedPrecommitMsg
}

// AfgReceivedPrevoteTM holds `afg.received_prevote` telemetry message which is
// supposed to be sent when grandpa client receives a prevote.
type AfgReceivedPrevoteTM afgReceivedTM

// NewAfgReceivedPrevoteTM gets a new AfgReceivedPrevoteTM struct.
func NewAfgReceivedPrevoteTM(targetHash common.Hash, targetNumber, voter string) *AfgReceivedPrevoteTM {
	return &AfgReceivedPrevoteTM{
		TargetHash:   targetHash,
		TargetNumber: targetNumber,
		Voter:        voter,
	}
}

func (AfgReceivedPrevoteTM) messageType() string {
	return afgReceivedPrevoteMsg
}

// AfgReceivedCommitTM holds `afg.received_commit` telemetry message which is
// supposed to be sent when grandpa client receives a commit.
type AfgReceivedCommitTM struct {
	TargetHash                 common.Hash `json:"target_hash"`
	TargetNumber               string      `json:"target_number"`
	ContainsPrecommitsSignedBy []string    `json:"contains_precommits_signed_by"`
}

// NewAfgReceivedCommitTM gets a new AfgReceivedCommitTM struct.
func NewAfgReceivedCommitTM(targetHash common.Hash, targetNumber string,
	containsPrecommitsSignedBy []string) *AfgReceivedCommitTM {
	return &AfgReceivedCommitTM{
		TargetHash:                 targetHash,
		TargetNumber:               targetNumber,
		ContainsPrecommitsSignedBy: containsPrecommitsSignedBy,
	}
}

func (AfgReceivedCommitTM) messageType() string {
	return afgReceivedCommitMsg
}
