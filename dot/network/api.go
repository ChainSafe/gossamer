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
	"errors"

	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

// SendCatchUpRequest sends a NotificationsMessage to a peer and returns a response, if there is one.
// Note: this is only used by grandpa at the moment, since it sends catch up requests/responses over the same
// notifications stream as the rest of the messages.
func (s *Service) SendCatchUpRequest(peer peer.ID, msgType byte, req NotificationsMessage) (NotificationsMessage, error) {
	s.host.h.ConnManager().Protect(peer, "")
	defer s.host.h.ConnManager().Unprotect(peer, "")

	s.notificationsMu.RLock()
	notifications := s.notificationsProtocols[msgType]
	s.notificationsMu.RUnlock()

	data, has := notifications.handshakeData.Load(peer)
	if !has {
		return nil, errors.New("stream not established with peer")
	}

	hsData, ok := data.(*handshakeData)
	if !ok {
		return nil, errors.New("failed to get handshake data for peer")
	}

	err := s.host.writeToStream(hsData.stream, req)
	if err != nil {
		return nil, err
	}

	return s.receiveResponse(hsData.stream)
}

func (s *Service) receiveResponse(stream libp2pnetwork.Stream) (*ConsensusMessage, error) {
	// TODO: don't always allocate this
	buf := make([]byte, maxBlockResponseSize)
	n, err := readStream(stream, buf)
	if err != nil {
		return nil, err
	}

	msg := new(ConsensusMessage)
	err = msg.Decode(buf[:n])
	return msg, err
}

// SendMessage ...
func (s *Service) SendMessage(msg NotificationsMessage) {
	if s.host == nil {
		return
	}
	if s.IsStopped() {
		return
	}
	if msg == nil {
		logger.Debug("Received nil message from core service")
		return
	}
	logger.Debug(
		"Broadcasting message from core service",
		"host", s.host.id(),
		"type", msg.Type(),
	)

	// check if the message is part of a notifications protocol
	s.notificationsMu.Lock()
	defer s.notificationsMu.Unlock()

	for msgID, prtl := range s.notificationsProtocols {
		if msg.Type() != msgID || prtl == nil {
			continue
		}

		s.broadcastExcluding(prtl, peer.ID(""), msg)
		return
	}

	logger.Error("message not supported by any notifications protocol", "msg type", msg.Type())
}
