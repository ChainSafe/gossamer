// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package messages

import (
	"context"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"

	"github.com/ChainSafe/gossamer/dot/network"
)

type ReqProtocolName uint

const (
	ChunkFetchingV1 ReqProtocolName = iota
	ChunkFetchingV2
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
	case ChunkFetchingV2:
		return "req_chunk/2"
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
	Recipient parachaintypes.RecipientID
	Payload   ReqProtocolMessage
	Result    chan ReqRespResult

	ctx    context.Context
	cancel context.CancelFunc
}

// Done returns a channel that is closed when the request is cancelled.
func (or *OutgoingRequest) Done() <-chan struct{} {
	return or.ctx.Done()
}

// Cancel cancels the request.
func (or *OutgoingRequest) Cancel() {
	or.cancel()
}

// IsCancelled returns true if the request has been cancelled.
func (or *OutgoingRequest) IsCancelled() bool {
	return or.ctx.Err() != nil
}

// NewOutgoingRequest creates a new outgoing request.
func NewOutgoingRequest(recipient parachaintypes.RecipientID, payload ReqProtocolMessage) *OutgoingRequest {
	result := make(chan ReqRespResult, 1)
	ctx, cancel := context.WithCancel(context.Background())

	return &OutgoingRequest{
		Recipient: recipient,
		Payload:   payload,
		Result:    result,
		ctx:       ctx,
		cancel:    cancel,
	}
}
