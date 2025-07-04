// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"slices"

	"github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	"github.com/libp2p/go-libp2p/core/peer"
)

type taggedResponse struct {
	identifier    candidateIdentifier
	requestedPeer peer.ID
	props         requestProperties
	// payload is AttestedCandidateResponse, maybe ReqRespResult should be generic over the response type?
	response chan messages.ReqRespResult
}

// Unlike in polkadot-sdk, this type does not simply wrap [taggedResponse]. The reason for that is the response channel
// in [taggedResponse]. In order for [responseManager.incoming] to find the next available response, it needs to read
// from the channel. Once it found a channel containing a message it could write it back to the channel and return
// an unhandledResponse wrapping the [taggedResponse]. But the channel has already been closed by the networking layer
// that wrote the [messages.ReqRespResult] to the channel.
type unhandledResponse struct {
	identifier    candidateIdentifier
	requestedPeer peer.ID
	props         requestProperties
	response      messages.ReqRespResult
}

type responseManager struct {
	pendingResponses []taggedResponse
	activePeers      map[peer.ID]struct{}
}

func newResponseManager() *responseManager {
	return &responseManager{
		pendingResponses: make([]taggedResponse, 0),
		activePeers:      make(map[peer.ID]struct{}),
	}
}

func (rm *responseManager) incoming() *unhandledResponse {
	if len(rm.pendingResponses) == 0 {
		return nil
	}

	for i, tr := range rm.pendingResponses {
		select {
		case response, ok := <-tr.response:
			rm.pendingResponses = slices.Delete(rm.pendingResponses, i, i+1)
			delete(rm.activePeers, tr.requestedPeer)

			// ok == false: The channel was closed without being written to.
			// This should only happen when a request is cancelled.
			if ok {
				return &unhandledResponse{
					identifier:    tr.identifier,
					requestedPeer: tr.requestedPeer,
					props:         tr.props,
					response:      response,
				}
			}
		default:
		}
	}
	return nil
}

func (rm *responseManager) push(tr taggedResponse) {
	rm.pendingResponses = append(rm.pendingResponses, tr)
	rm.activePeers[tr.requestedPeer] = struct{}{}
}

func (rm *responseManager) isSendingTo(p peer.ID) bool {
	_, ok := rm.activePeers[p]
	return ok
}
