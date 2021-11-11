// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"context"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	maxBlockResponseSize uint64 = 1024 * 1024 * 4 // 4mb
	blockRequestTimeout         = time.Second * 5
)

// DoBlockRequest sends a request to the given peer. If a response is received within a certain time period, it is returned, otherwise an error is returned.
func (s *Service) DoBlockRequest(to peer.ID, req *BlockRequestMessage) (*BlockResponseMessage, error) {
	fullSyncID := s.host.protocolID + syncID

	s.host.h.ConnManager().Protect(to, "")
	defer s.host.h.ConnManager().Unprotect(to, "")

	ctx, cancel := context.WithTimeout(s.ctx, blockRequestTimeout)
	defer cancel()

	stream, err := s.host.h.NewStream(ctx, to, fullSyncID)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = stream.Close()
	}()

	if err = s.host.writeToStream(stream, req); err != nil {
		return nil, err
	}

	return s.receiveBlockResponse(stream)
}

func (s *Service) receiveBlockResponse(stream libp2pnetwork.Stream) (*BlockResponseMessage, error) {
	// allocating a new (large) buffer every time slows down the syncing by a dramatic amount,
	// as malloc is one of the most CPU intensive tasks.
	// thus we should allocate buffers at startup and re-use them instead of allocating new ones each time.
	//
	// TODO: should we create another buffer pool for block response buffers?
	// for bootstrap this is ok since it's not parallelized, but will need to be updated for tip-mode (#1858)
	s.blockResponseBufMu.Lock()
	defer s.blockResponseBufMu.Unlock()

	buf := s.blockResponseBuf

	n, err := readStream(stream, buf)
	if err != nil {
		return nil, fmt.Errorf("read stream error: %w", err)
	}

	if n == 0 {
		return nil, fmt.Errorf("received empty message")
	}

	msg := new(BlockResponseMessage)
	err = msg.Decode(buf[:n])
	if err != nil {
		s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
			Value:  peerset.BadMessageValue,
			Reason: peerset.BadMessageReason,
		}, stream.Conn().RemotePeer())
		return nil, fmt.Errorf("failed to decode block response: %w", err)
	}

	return msg, nil
}

// handleSyncStream handles streams with the <protocol-id>/sync/2 protocol ID
func (s *Service) handleSyncStream(stream libp2pnetwork.Stream) {
	if stream == nil {
		return
	}

	s.readStream(stream, decodeSyncMessage, s.handleSyncMessage)
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
		_ = stream.Close()
	}()

	if req, ok := msg.(*BlockRequestMessage); ok {
		resp, err := s.syncer.CreateBlockResponse(req)
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
