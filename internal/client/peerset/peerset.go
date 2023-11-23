package peerset

// / Description of a reputation adjustment for a node.
type ReputationChange struct {
	/// Reputation delta.
	Value int32
	/// Reason for reputation change.
	Reason string
}
