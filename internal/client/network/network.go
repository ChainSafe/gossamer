package network

// The protocol name transmitted on the wire.
type ProtocolName string

// ReputationChange is a description of a reputation adjustment for a node.
type ReputationChange struct {
	// Reputation delta.
	Value int32
	// Reason for reputation change.
	Reason string
}

// New reputation change with given delta and reason.
func NewReputationChange(value int32, reason string) ReputationChange {
	return ReputationChange{Value: value, Reason: reason}
}
