// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
)

func (s *Service) readStream(stream libp2pnetwork.Stream, decoder messageDecoder, handler messageHandler) {
	// we NEED to reset the stream if we ever return from this function, as if we return,
	// the stream will never again be read by us, so we need to tell the remote side we're
	// done with this stream, and they should also forget about it.
	defer s.resetInboundStream(stream)
	s.streamManager.logNewStream(stream)

	peer := stream.Conn().RemotePeer()
	msgBytes := s.bufPool.get()
	defer s.bufPool.put(msgBytes)

	for {
		n, err := readStream(stream, msgBytes[:])
		if err != nil {
			logger.Tracef(
				"failed to read from stream id %s of peer %s using protocol %s: %s",
				stream.ID(), stream.Conn().RemotePeer(), stream.Protocol(), err)
			return
		}

		s.streamManager.logMessageReceived(stream.ID())

		// decode message based on message type
		msg, err := decoder(msgBytes[:n], peer, isInbound(stream)) // stream should always be inbound if it passes through service.readStream
		if err != nil {
			logger.Tracef("failed to decode message from stream id %s using protocol %s: %s",
				stream.ID(), stream.Protocol(), err)
			continue
		}

		logger.Tracef(
			"host %s received message from peer %s: %s",
			s.host.id(), peer, msg)

		if err = handler(stream, msg); err != nil {
			logger.Tracef("failed to handle message %s from stream id %s: %s", msg, stream.ID(), err)
			return
		}

		s.host.bwc.LogRecvMessage(int64(n))
	}
}

func (s *Service) resetInboundStream(stream libp2pnetwork.Stream) {
	protocolID := stream.Protocol()
	peerID := stream.Conn().RemotePeer()

	s.notificationsMu.Lock()
	defer s.notificationsMu.Unlock()

	for _, prtl := range s.notificationsProtocols {
		if prtl.protocolID != protocolID {
			continue
		}

		prtl.inboundHandshakeData.Delete(peerID)
		break
	}

	logger.Debugf(
		"cleaning up inbound handshake data for protocol=%s, peer=%s",
		stream.Protocol(),
		peerID,
	)

	_ = stream.Reset()
}
