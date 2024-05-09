// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const maxNumberOfSameRequestPerPeer uint = 2

var ErrMaxNumberOfSameRequest = errors.New("max number of same request reached")

// handleSyncStream handles streams with the <protocol-id>/sync/2 protocol ID
func (s *Service) handleSyncStream(stream libp2pnetwork.Stream) {
	if stream == nil {
		return
	}

	s.readStream(stream, decodeSyncMessage, s.handleSyncMessage, MaxBlockResponseSize)
}

func decodeSyncMessage(in []byte, _ peer.ID, _ bool) (Message, error) {
	msg := new(BlockRequestMessage)
	err := msg.Decode(in)
	return msg, err
}

// handleSyncMessage handles inbound sync streams
// the only messages we should receive over an inbound stream are BlockRequestMessages, so we only need to handle those
func (s *Service) handleSyncMessage(stream libp2pnetwork.Stream, msg Message) error {
	if msg == nil {
		return nil
	}

	defer func() {
		err := stream.Close()
		if err != nil && err.Error() != ErrStreamReset.Error() {
			logger.Warnf("failed to close stream: %s", err)
		}
	}()

	encodedMessage, err := msg.Encode()
	if err != nil {
		return fmt.Errorf("encoding block request sync message: %w", err)
	}

	peerID := stream.Conn().RemotePeer()
	encodedKey := bytes.Join([][]byte{[]byte(peerID.String()), encodedMessage}, nil)

	requestHash, err := common.Blake2bHash(encodedKey)
	if err != nil {
		return fmt.Errorf("hashing encoded block request sync message: %w", err)
	}

	numOfRequests := s.seenBlockSyncRequests.Get(requestHash)
	if numOfRequests > maxNumberOfSameRequestPerPeer {

		s.ReportPeer(peerset.ReputationChange{
			Value:  peerset.SameBlockSyncRequest,
			Reason: peerset.SameBlockSyncRequestReason,
		}, peerID)

		logger.Debugf("max number of same request reached by: %s", peerID.String())
		return fmt.Errorf("%w: %s", ErrMaxNumberOfSameRequest, peerID.String())
	}

	s.seenBlockSyncRequests.Put(requestHash, numOfRequests+1)
	if req, ok := msg.(*BlockRequestMessage); ok {
		resp, err := s.syncer.CreateBlockResponse(stream.Conn().RemotePeer(), req)
		if err != nil {
			logger.Debugf("cannot create response for request: %s", err)
			return nil
		}

		if err = s.host.writeToStream(stream, resp); err != nil {
			logger.Debugf("failed to send BlockResponse message to peer %s: %s", stream.Conn().RemotePeer(), err)
			return err
		}
	}

	return nil
}
