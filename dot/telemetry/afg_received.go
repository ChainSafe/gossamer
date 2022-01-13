// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
)

// AfG ("Al's Finality Gadget") is synonymous with GRANDPA.

var (
	_ Message = (*AfgReceivedPrecommit)(nil)
	_ Message = (*AfgReceivedPrevote)(nil)
	_ Message = (*AfgReceivedCommit)(nil)
)

type afgReceived struct {
	TargetHash   common.Hash `json:"target_hash"`
	TargetNumber string      `json:"target_number"`
	Voter        string      `json:"voter"`
}

// AfgReceivedPrecommit holds `afg.received_precommit` telemetry message which is
// supposed to be sent when grandpa client receives a precommit.
type AfgReceivedPrecommit afgReceived

// NewAfgReceivedPrecommit gets a new AfgReceivedPrecommitTM struct.
func NewAfgReceivedPrecommit(targetHash common.Hash, targetNumber, voter string) *AfgReceivedPrecommit {
	return &AfgReceivedPrecommit{
		TargetHash:   targetHash,
		TargetNumber: targetNumber,
		Voter:        voter,
	}
}

func (afg AfgReceivedPrecommit) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		afgReceived
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		Timestamp:   time.Now(),
		MessageType: afgReceivedPrecommitMsg,
		afgReceived: afgReceived(afg),
	}

	return json.Marshal(telemetryData)
}

// AfgReceivedPrevote holds `afg.received_prevote` telemetry message which is
// supposed to be sent when grandpa client receives a prevote.
type AfgReceivedPrevote afgReceived

// NewAfgReceivedPrevote gets a new AfgReceivedPrevote* struct.
func NewAfgReceivedPrevote(targetHash common.Hash, targetNumber, voter string) *AfgReceivedPrevote {
	return &AfgReceivedPrevote{
		TargetHash:   targetHash,
		TargetNumber: targetNumber,
		Voter:        voter,
	}
}

func (afg AfgReceivedPrevote) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		afgReceived
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		Timestamp:   time.Now(),
		MessageType: afgReceivedPrevoteMsg,
		afgReceived: afgReceived(afg),
	}

	return json.Marshal(telemetryData)
}

type afgReceivedCommitTM AfgReceivedCommit

// AfgReceivedCommit holds `afg.received_commit` telemetry message which is
// supposed to be sent when grandpa client receives a commit.
type AfgReceivedCommit struct {
	TargetHash                 common.Hash `json:"target_hash"`
	TargetNumber               string      `json:"target_number"`
	ContainsPrecommitsSignedBy []string    `json:"contains_precommits_signed_by"`
}

// NewAfgReceivedCommit gets a new AfgReceivedCommit* struct.
func NewAfgReceivedCommit(targetHash common.Hash, targetNumber string,
	containsPrecommitsSignedBy []string) *AfgReceivedCommit {
	return &AfgReceivedCommit{
		TargetHash:                 targetHash,
		TargetNumber:               targetNumber,
		ContainsPrecommitsSignedBy: containsPrecommitsSignedBy,
	}
}

func (afg AfgReceivedCommit) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		afgReceivedCommitTM
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		Timestamp:           time.Now(),
		MessageType:         afgReceivedCommitMsg,
		afgReceivedCommitTM: afgReceivedCommitTM(afg),
	}

	return json.Marshal(telemetryData)
}
