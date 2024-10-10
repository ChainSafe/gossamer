// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/lib/common"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const MaxAllowedSameRequestPerPeer = 5

type WarpSyncVerificationResult struct {
	SetId         grandpa.SetID
	AuthorityList grandpa.AuthorityList
	Header        generic.Header[uint, hash.H256, runtime.BlakeTwo256]
	Completed     bool
}

// WarpSyncProvider is an interface for generating warp sync proofs
type WarpSyncProvider interface {
	// Generate proof starting at given block hash. The proof is accumulated until maximum proof
	// size is reached.
	Generate(start common.Hash) (encodedProof []byte, err error)
	Verify(
		encodedProof []byte,
		setId grandpa.SetID,
		authorities grandpa.AuthorityList,
	) (*WarpSyncVerificationResult, error)
}

func (s *Service) handleWarpSyncRequest(req messages.WarpProofRequest) ([]byte, error) {
	// use the backend to generate the warp proof
	proof, err := s.warpSyncProvider.Generate(req.Begin)
	if err != nil {
		return nil, err
	}
	// send the response through pendingResponse channel
	return proof, nil
}

func (s *Service) handleWarpSyncStream(stream libp2pnetwork.Stream) {
	if stream == nil {
		return
	}

	s.readStream(stream, decodeWarpSyncMessage, s.handleWarpSyncMessage, MaxBlockResponseSize)
}

func decodeWarpSyncMessage(in []byte, _ peer.ID, _ bool) (messages.P2PMessage, error) {
	msg := new(messages.WarpProofRequest)
	err := msg.Decode(in)
	return msg, err
}

func (s *Service) handleWarpSyncMessage(stream libp2pnetwork.Stream, msg messages.P2PMessage) error {
	if msg == nil {
		return nil
	}

	defer func() {
		err := stream.Close()
		if err != nil && errors.Is(err, ErrStreamReset) {
			logger.Warnf("failed to close stream: %s", err)
		}
	}()

	reqId := fmt.Sprintf("%s-%s", stream.Conn().RemotePeer(), msg.String())
	hashedreqId := common.MustBlake2bHash([]byte(reqId))

	if req, ok := msg.(*messages.WarpProofRequest); ok {
		// Check if this peer has exceeded the limit of requests
		if s.warpSyncSpamLimiter.IsLimitExceeded(hashedreqId) {
			logger.Debugf("same warp sync request exceeded for peer: %s", stream.Conn().RemotePeer())
			return nil
		}

		// Add the request to the spam limiter
		s.warpSyncSpamLimiter.AddRequest(hashedreqId)

		// Handle request
		resp, err := s.handleWarpSyncRequest(*req)
		if err != nil {
			logger.Debugf("cannot create response for request: %s", err)
			return nil
		}

		if _, err = stream.Write(resp); err != nil {
			logger.Debugf("failed to send WarpSyncResponse message to peer %s: %s", stream.Conn().RemotePeer(), err)
			return err
		}

		logger.Debugf("successfully respond with WarpSyncResponse message to peer %s with proof %v",
			stream.Conn().RemotePeer(),
			resp,
		)
	} else {
		logger.Debugf("received invalid message in warp sync handler: %v", msg)
	}

	return nil
}
