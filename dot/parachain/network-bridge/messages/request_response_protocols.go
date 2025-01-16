package messages

import (
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type ReqProtocolName uint

const (
	ChunkFetchingV1 ReqProtocolName = iota
	CollationFetchingV1
	PoVFetchingV1
	AvailableDataFetchingV1
	StatementFetchingV1
	DisputeSendingV1
)

func (n ReqProtocolName) String() string {
	switch n {
	case ChunkFetchingV1:
		return "req_chunk/1"
	case CollationFetchingV1:
		return "req_collation/1"
	case PoVFetchingV1:
		return "req_pov/1"
	case AvailableDataFetchingV1:
		return "req_available_data/1"
	case StatementFetchingV1:
		return "req_statement/1"
	case DisputeSendingV1:
		return "send_dispute/1"
	default:
		panic("unknown protocol")
	}
}

// ReqProtocolMessage is a network message that can be sent over a request response protocol.
type ReqProtocolMessage interface {
	network.Message
	// Response returns an instance of the response type for this message, for the purpose of decoding into it.
	Response() network.ResponseMessage
	Protocol() ReqProtocolName
}

// ReqRespResult is the result of sending a request over a request response protocol. It contains either a response
// message or an error.
type ReqRespResult struct {
	Response network.ResponseMessage
	Error    error
}

// OutgoingRequest contains all data required to send a request over a request response protocol and receive the result.
type OutgoingRequest struct {
	Recipient peer.ID // TODO use a type that can contain either a peer ID or an authority ID
	Payload   ReqProtocolMessage
	Result    chan ReqRespResult
}

func NewOutgoingRequest(recipient peer.ID, payload ReqProtocolMessage) *OutgoingRequest {
	result := make(chan ReqRespResult, 1)

	return &OutgoingRequest{
		Recipient: recipient,
		Payload:   payload,
		Result:    result,
	}
}
