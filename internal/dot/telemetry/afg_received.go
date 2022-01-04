// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import "github.com/ChainSafe/gossamer/lib/common"

// AfG ("Al's Finality Gadget") is synonymous with GRANDPA.

type afgReceivedTM struct {
	TargetHash   common.Hash `json:"target_hash"`
	TargetNumber string      `json:"target_number"`
	Voter        string      `json:"voter"`
}

// afgReceivedPrecommitTM holds `afg.received_precommit` telemetry message which is
// supposed to be sent when grandpa client receives a precommit.
type afgReceivedPrecommitTM afgReceivedTM

// NewAfgReceivedPrecommitTM gets a new afgReceivedPrecommitTM struct.
func NewAfgReceivedPrecommitTM(targetHash common.Hash, targetNumber, voter string) Message {
	return &afgReceivedPrecommitTM{
		TargetHash:   targetHash,
		TargetNumber: targetNumber,
		Voter:        voter,
	}
}

func (afgReceivedPrecommitTM) messageType() string {
	return afgReceivedPrecommitMsg
}

// afgReceivedPrevoteTM holds `afg.received_prevote` telemetry message which is
// supposed to be sent when grandpa client receives a prevote.
type afgReceivedPrevoteTM afgReceivedTM

// NewAfgReceivedPrevoteTM gets a new afgReceivedPrevoteTM struct.
func NewAfgReceivedPrevoteTM(targetHash common.Hash, targetNumber, voter string) Message {
	return &afgReceivedPrevoteTM{
		TargetHash:   targetHash,
		TargetNumber: targetNumber,
		Voter:        voter,
	}
}

func (afgReceivedPrevoteTM) messageType() string {
	return afgReceivedPrevoteMsg
}

// afgReceivedCommitTM holds `afg.received_commit` telemetry message which is
// supposed to be sent when grandpa client receives a commit.
type afgReceivedCommitTM struct {
	TargetHash                 common.Hash `json:"target_hash"`
	TargetNumber               string      `json:"target_number"`
	ContainsPrecommitsSignedBy []string    `json:"contains_precommits_signed_by"`
}

// NewAfgReceivedCommitTM gets a new afgReceivedCommitTM struct.
func NewAfgReceivedCommitTM(targetHash common.Hash, targetNumber string, containsPrecommitsSignedBy []string) Message {
	return &afgReceivedCommitTM{
		TargetHash:                 targetHash,
		TargetNumber:               targetNumber,
		ContainsPrecommitsSignedBy: containsPrecommitsSignedBy,
	}
}

func (afgReceivedCommitTM) messageType() string {
	return afgReceivedCommitMsg
}
