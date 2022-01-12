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
	_ Message = (*AfgReceivedPrecommitTM)(nil)
	_ Message = (*AfgReceivedPrevoteTM)(nil)
	_ Message = (*AfgReceivedCommit)(nil)
)

type afgReceived struct {
	TargetHash   common.Hash `json:"target_hash"`
	TargetNumber string      `json:"target_number"`
	Voter        string      `json:"voter"`
}

// AfgReceivedPrecommitTM holds `afg.received_precommit` telemetry message which is
// supposed to be sent when grandpa client receives a precommit.
type AfgReceivedPrecommitTM afgReceived

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

func (afg AfgReceivedPrecommitTM) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		afgReceived
		Timestamp   time.Time `json:"ts"`
		MessageType string    `json:"msg"`
	}{
		Timestamp:   time.Now(),
		MessageType: afg.messageType(),
		afgReceived: afgReceived(afg),
	}

	return json.Marshal(telemetryData)
}

// AfgReceivedPrevoteTM holds `afg.received_prevote` telemetry message which is
// supposed to be sent when grandpa client receives a prevote.
type AfgReceivedPrevoteTM afgReceived

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

func (afg AfgReceivedPrevoteTM) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		afgReceived
		Timestamp   time.Time `json:"ts"`
		MessageType string    `json:"msg"`
	}{
		Timestamp:   time.Now(),
		MessageType: afg.messageType(),
		afgReceived: afgReceived(afg),
	}

	return json.Marshal(telemetryData)
}

type AfgReceivedCommitTM AfgReceivedCommit

// AfgReceivedCommit holds `afg.received_commit` telemetry message which is
// supposed to be sent when grandpa client receives a commit.
type AfgReceivedCommit struct {
	TargetHash                 common.Hash `json:"target_hash"`
	TargetNumber               string      `json:"target_number"`
	ContainsPrecommitsSignedBy []string    `json:"contains_precommits_signed_by"`
}

// NewAfgReceivedCommitTM gets a new AfgReceivedCommitTM struct.
func NewAfgReceivedCommit(targetHash common.Hash, targetNumber string,
	containsPrecommitsSignedBy []string) *AfgReceivedCommit {
	return &AfgReceivedCommit{
		TargetHash:                 targetHash,
		TargetNumber:               targetNumber,
		ContainsPrecommitsSignedBy: containsPrecommitsSignedBy,
	}
}

func (AfgReceivedCommit) messageType() string {
	return afgReceivedCommitMsg
}

func (afg AfgReceivedCommit) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		AfgReceivedCommitTM
		Timestamp   time.Time `json:"ts"`
		MessageType string    `json:"msg"`
	}{
		Timestamp:           time.Now(),
		MessageType:         afg.messageType(),
		AfgReceivedCommitTM: AfgReceivedCommitTM(afg),
	}

	return json.Marshal(telemetryData)
}
