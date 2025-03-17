// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

// ProtocolName is the protocol name transmitted on the wire.
type ProtocolName string

// ReputationChange is a description of a reputation adjustment for a node.
type ReputationChange struct {
	// Reputation delta.
	Value int32
	// Reason for reputation change.
	Reason string
}

// NewReputationChange constructs a [ReputationChange] with given delta and reason.
func NewReputationChange(value int32, reason string) ReputationChange {
	return ReputationChange{Value: value, Reason: reason}
}
