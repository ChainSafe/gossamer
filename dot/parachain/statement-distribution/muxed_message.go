package statementdistribution

import "time"

// MuxedMessage represents the kinds of messages
// the statement distribution can handle, these messages
// can have different origin and types, so the interface
// acts like a union that helps the subsytem to handle
// each message properly
type MuxedMessage interface {
	isMuxedMessage()
}

type overseerMessage struct {
	inner any
}

func (*overseerMessage) isMuxedMessage() {}

// responderMessage is a message from the request handler
// indicating we received a request and we should produce
// a proper response and send it back
type responderMessage struct {
	inner any // should be replaced with AttestedCandidateRequest type
}

func (*responderMessage) isMuxedMessage() {}

// reputationChangeMessage is a message indicating we should
// batch the reputation changes to the network bridge via
// Reputation Aggregator
type reputationChangeMessage struct{}

func (*reputationChangeMessage) isMuxedMessage() {}

// awaitMessageFrom waits for messages from either the overseerToSubSystem, responderCh, or reputationDelay
func (s *StatementDistribution) awaitMessageFrom(
	overseerToSubSystem <-chan any,
	responderCh chan any,
	reputationDelay <-chan time.Time,
) MuxedMessage {
	select {
	case msg := <-overseerToSubSystem:
		return &overseerMessage{inner: msg}
	case msg := <-responderCh:
		return &responderMessage{inner: msg}
	case <-reputationDelay:
		return &reputationChangeMessage{}
	}
}
