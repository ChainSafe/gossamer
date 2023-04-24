// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	blockRequestTimeout = time.Second * 20
)

func (s *Service) RequestWarpProof(to peer.ID, request *WarpSyncProofRequestMessage) (warpSyncResponse interface{}, err error) {
	legacyWarpSyncID := s.host.protocolID + warpSyncID

	s.host.p2pHost.ConnManager().Protect(to, "")
	defer s.host.p2pHost.ConnManager().Unprotect(to, "")

	ctx, cancel := context.WithTimeout(s.ctx, blockRequestTimeout)
	defer cancel()

	stream, err := s.host.p2pHost.NewStream(ctx, to, legacyWarpSyncID)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := stream.Close()
		if err != nil {
			logger.Warnf("failed to close stream: %s", err)
		}
	}()

	if err = s.host.writeToStream(stream, request); err != nil {
		return nil, err
	}

	return s.handleWarpSyncProofResponse(stream)
}

// DoBlockRequest sends a request to the given peer.
// If a response is received within a certain time period, it is returned,
// otherwise an error is returned.
func (s *Service) DoBlockRequest(to peer.ID, req *BlockRequestMessage) (*BlockResponseMessage, error) {
	fullSyncID := s.host.protocolID + syncID

	s.host.p2pHost.ConnManager().Protect(to, "")
	defer s.host.p2pHost.ConnManager().Unprotect(to, "")

	ctx, cancel := context.WithTimeout(s.ctx, blockRequestTimeout)
	defer cancel()

	stream, err := s.host.p2pHost.NewStream(ctx, to, fullSyncID)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := stream.Close()
		if err != nil {
			logger.Warnf("failed to close stream: %s", err)
		}
	}()

	if err = s.host.writeToStream(stream, req); err != nil {
		return nil, err
	}

	return s.receiveBlockResponse(stream)
}

func (s *Service) handleWarpSyncProofResponse(stream libp2pnetwork.Stream) (interface{}, error) {
	s.blockResponseBufMu.Lock()
	defer s.blockResponseBufMu.Unlock()

	// TODO: should we create another buffer pool for warp proof response buffers?
	buf := s.blockResponseBuf

	n, err := readStream(stream, &buf, warpSyncMaxResponseSize)
	if err != nil {
		return nil, fmt.Errorf("reading warp sync stream: %w", err)
	}

	if n == 0 {
		return nil, fmt.Errorf("empty warp sync proof")
	}

	fmt.Printf("WARP PROOF BYTES ---> %v\n", buf[:n])
	warpProof := new(WarpSyncProofResponse)
	err = warpProof.Decode(buf[:n])
	if err != nil {
		panic(fmt.Sprintf("failed to decode warp proof: %s", err))
	}
	fmt.Printf("WARP PROOF ---> %v\n", warpProof)
	return nil, nil
}

var ErrReceivedEmptyMessage = errors.New("received empty message")

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

	n, err := readStream(stream, &buf, maxBlockResponseSize)
	if err != nil {
		return nil, fmt.Errorf("read stream error: %w", err)
	}

	if n == 0 {
		return nil, fmt.Errorf("%w", ErrReceivedEmptyMessage)
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

	s.readStream(stream, decodeSyncMessage, s.handleSyncMessage, maxBlockResponseSize)
}

func (s *Service) handleWarpSyncStream(stream libp2pnetwork.Stream) {
	if stream == nil {
		return
	}

	fmt.Printf("====> %v\n", stream)
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
		if err != nil {
			logger.Warnf("failed to close stream: %s", err)
		}
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
