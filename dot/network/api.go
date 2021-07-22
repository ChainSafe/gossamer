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

// import (
// 	"errors"
// 	"fmt"
// 	"time"

// 	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
// 	"github.com/libp2p/go-libp2p-core/peer"
// )

// var resCh = make(chan interface{})

// // SendCatchUpRequest sends a NotificationsMessage to a peer and returns a response, if there is one.
// // Note: this is only used by grandpa at the moment, since it sends catch up requests/responses over the same
// // notifications stream as the rest of the messages.
// func (s *Service) SendCatchUpRequest(peer peer.ID, msgType byte, req *ConsensusMessage) (*ConsensusMessage, error) {
// 	s.host.h.ConnManager().Protect(peer, "")
// 	defer s.host.h.ConnManager().Unprotect(peer, "")

// 	s.notificationsMu.RLock()
// 	notifications, has := s.notificationsProtocols[msgType]
// 	s.notificationsMu.RUnlock()

// 	if !has {
// 		return nil, errors.New("notifications protocol not registered")
// 	}

// 	hs, err := notifications.getHandshake()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// TODO: probably remove this, there needs to be a better way to do this.
// 	s.host.registerStreamHandlerWithOverwrite(notifications.protocolID, true, s.receiveResponse)

// 	// write to outbound stream, establish handshake if needed
// 	err = s.sendData(peer, hs, notifications, req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	hsData, has := notifications.getHandshakeData(peer, true) // get inbound stream to read response from
// 	if !has {
// 		// inbound stream should established already, since we received a NeighbourMessage from this peer
// 		return nil, errors.New("inbound stream not established with peer")
// 	}

// 	fmt.Println("SendCatchUpRequest inbound", hsData.stream.ID())

// 	// go func() {
// 	// 	resp, err := s.receiveResponse(hsData.stream)
// 	// 	if err != nil {
// 	// 		resCh <- err
// 	// 		return
// 	// 	}

// 	// 	resCh <- resp
// 	// }()

// 	select {
// 	case <-time.After(time.Second * 5):
// 		return nil, errors.New("timeout")
// 	case res := <-resCh:
// 		if msg, ok := res.(*ConsensusMessage); ok {
// 			return msg, nil
// 		}

// 		if err, ok := res.(error); ok {
// 			return nil, err
// 		}
// 	}

// 	return nil, errors.New("failed to receive response")
// }

// func (s *Service) receiveResponse(stream libp2pnetwork.Stream) {
// 	// TODO: don't always allocate this
// 	buf := make([]byte, 1024*1024)
// 	n, err := readStream(stream, buf)
// 	if err != nil {
// 		resCh <- err
// 		return
// 	}

// 	logger.Info("got catch up response!", "data", buf[:n])

// 	msg := new(ConsensusMessage)
// 	err = msg.Decode(buf[:n])
// 	if err != nil {
// 		resCh <- err
// 		return
// 	}

// 	resCh <- msg
// }

// // SendMessage gossips a message to our peers
// func (s *Service) SendMessage(msg NotificationsMessage) {
// 	if s.host == nil {
// 		return
// 	}
// 	if s.IsStopped() {
// 		return
// 	}
// 	if msg == nil {
// 		logger.Debug("Received nil message from core service")
// 		return
// 	}
// 	logger.Debug(
// 		"gossiping message",
// 		"type", msg.Type(),
// 		"message", msg,
// 	)

// 	// check if the message is part of a notifications protocol
// 	s.notificationsMu.Lock()
// 	defer s.notificationsMu.Unlock()

// 	for msgID, prtl := range s.notificationsProtocols {
// 		if msg.Type() != msgID || prtl == nil {
// 			continue
// 		}

// 		s.broadcastExcluding(prtl, peer.ID(""), msg)
// 		return
// 	}

// 	logger.Error("message not supported by any notifications protocol", "msg type", msg.Type())
// }
