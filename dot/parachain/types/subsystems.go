// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"errors"
	"time"
)

type SubSystemName string

const (
	CandidateBacking         SubSystemName = "CandidateBacking"
	CollationProtocol        SubSystemName = "CollationProtocol"
	AvailabilityStore        SubSystemName = "AvailabilityStore"
	AvailabilityDistribution SubSystemName = "AvailabilityDistribution"
	NetworkBridgeSender      SubSystemName = "NetworkBridgeSender"
	NetworkBridgeReceiver    SubSystemName = "NetworkBridgeReceiver"
	ChainAPI                 SubSystemName = "ChainAPI"
	CandidateValidation      SubSystemName = "CandidateValidation"
	Provisioner              SubSystemName = "Provisioner"
	StatementDistribution    SubSystemName = "StatementDistribution"
	ProspectiveParachains    SubSystemName = "ProspectiveParachains"
	BitfieldSigning          SubSystemName = "BitfieldSigning"
	BitfieldDistribution     SubSystemName = "BitfieldDistribution"
	ApprovalDistribution     SubSystemName = "ApprovalDistribution"
	DisputesCoordinator      SubSystemName = "DisputesCoordinator"
)

var SubsystemRequestTimeout = 5 * time.Second
var ErrSubsystemRequestTimeout = errors.New("subsystem request timed out")
