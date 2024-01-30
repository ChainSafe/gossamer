// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"errors"
	"time"
)

type SubSystemName string

const (
	CandidateBacking      SubSystemName = "CandidateBacking"
	CollationProtocol     SubSystemName = "CollationProtocol"
	AvailabilityStore     SubSystemName = "AvailabilityStore"
	NetworkBridgeSender   SubSystemName = "NetworkBridgeSender"
	NetworkBridgeReceiver SubSystemName = "NetworkBridgeReceiver"
	ChainAPI          SubSystemName = "ChainAPI"
	RuntimeAPI        SubSystemName = "RuntimeAPI"
)

var SubsystemRequestTimeout = 1 * time.Second
var ErrSubsystemRequestTimeout = errors.New("subsystem request timed out")
