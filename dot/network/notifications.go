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
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

var errCannotValidateHandshake = errors.New("failed to validate handshake")

// Handshake is the interface all handshakes for notifications protocols must implement
type Handshake interface {
	NotificationsMessage
}

// the following are used for RegisterNotificationsProtocol
type (
	// HandshakeGetter is a function that returns a custom handshake
	HandshakeGetter = func() (Handshake, error)

	// HandshakeDecoder is a custom decoder for a handshake
	HandshakeDecoder = func([]byte) (Handshake, error)

	// HandshakeValidator validates a handshake. It returns an error if it is invalid
	HandshakeValidator = func(Handshake) (Message, error)

	// MessageDecoder is a custom decoder for a message
	MessageDecoder = func([]byte) (NotificationsMessage, error)

	// NotificationsMessageHandler is called when a (non-handshake) message is received over a notifications stream.
	NotificationsMessageHandler = func(peer peer.ID, msg NotificationsMessage) error
)

type notificationsProtocol struct {
	subProtocol   protocol.ID
	getHandshake  HandshakeGetter
	handshakeData map[peer.ID]*handshakeData
	mapMu         sync.RWMutex
}

type handshakeData struct {
	received    bool
	validated   bool
	outboundMsg NotificationsMessage
}

func createDecoder(info *notificationsProtocol, handshakeDecoder HandshakeDecoder, messageDecoder MessageDecoder) messageDecoder {
	return func(in []byte, peer peer.ID) (Message, error) {
		// if we don't have handshake data on this peer, or we haven't received the handshake from them already,
		// assume we are receiving the handshake
		info.mapMu.RLock()
		defer info.mapMu.RUnlock()

		if hsData, has := info.handshakeData[peer]; !has || !hsData.received {
			return handshakeDecoder(in)
		}

		// otherwise, assume we are receiving the Message
		return messageDecoder(in)
	}
}

func (s *Service) createNotificationsMessageHandler(info *notificationsProtocol, handshakeValidator HandshakeValidator, messageHandler NotificationsMessageHandler) messageHandler {
	return func(peer peer.ID, m Message) error {
		if m == nil || info == nil || handshakeValidator == nil || messageHandler == nil {
			return nil
		}

		var (
			ok  bool
			msg NotificationsMessage
		)

		if msg, ok = m.(NotificationsMessage); !ok {
			return errors.New("message is not NotificationsMessage")
		}

		logger.Debug("received message on notifications sub-protocol", "sub-protocol", info.subProtocol, "message", msg)

		if msg.IsHandshake() {
			hs, ok := msg.(Handshake)
			if !ok {
				return errors.New("failed to convert message to Handshake")
			}

			info.mapMu.Lock()
			defer info.mapMu.Unlock()

			// if we are the receiver and haven't received the handshake already, validate it
			if _, has := info.handshakeData[peer]; !has {
				logger.Trace("receiver: validating handshake", "sub-protocol", info.subProtocol)
				handshakeResp, err := handshakeValidator(hs)
				if err != nil {
					logger.Error("failed to validate handshake", "sub-protocol", info.subProtocol, "peer", peer, "error", err)
					info.handshakeData[peer] = &handshakeData{
						validated: false,
						received:  true,
					}
					return errCannotValidateHandshake
				}

				info.handshakeData[peer] = &handshakeData{
					validated: true,
					received:  true,
				}

				// once validated, send back a handshake
				resp, err := info.getHandshake()
				if err != nil {
					logger.Error("failed to get handshake", "sub-protocol", info.subProtocol, "error", err)
					return nil
				}

				err = s.host.send(peer, info.subProtocol, resp)
				if err != nil {
					logger.Error("failed to send handshake", "sub-protocol", info.subProtocol, "peer", peer, "error", err)
					return err
				}
				logger.Trace("receiver: sent handshake", "sub-protocol", info.subProtocol, "peer", peer)

				if handshakeResp != nil {
					logger.Debug("sending message to peer in response to handshake", "peer", peer, "msg", handshakeResp)
					err = s.host.send(peer, info.subProtocol, handshakeResp)
					if err != nil {
						logger.Error("failed to send response to peer's handshake", "sub-protocol", info.subProtocol, "peer", peer, "error", err)
						return err
					}
				}
			}

			// if we are the initiator and haven't received the handshake already, validate it
			if hsData, has := info.handshakeData[peer]; has && !hsData.validated {
				logger.Trace("sender: validating handshake")
				handshakeResp, err := handshakeValidator(hs)
				if err != nil {
					logger.Error("failed to validate handshake", "sub-protocol", info.subProtocol, "peer", peer, "error", err)
					delete(info.handshakeData, peer)
					return errCannotValidateHandshake
				}

				info.handshakeData[peer].validated = true
				info.handshakeData[peer].received = true
				logger.Trace("sender: validated handshake", "sub-protocol", info.subProtocol, "peer", peer)

				if handshakeResp != nil {
					logger.Debug("sending message to peer in response to handshake", "peer", peer, "msg", handshakeResp)
					err = s.host.send(peer, info.subProtocol, handshakeResp)
					if err != nil {
						logger.Error("failed to send response to peer's handshake", "sub-protocol", info.subProtocol, "peer", peer, "error", err)
						return err
					}
				}
			} else if hsData.received {
				return nil
			}

			// if we are the initiator, send the message
			if hsData, has := info.handshakeData[peer]; has && hsData.validated && hsData.received && hsData.outboundMsg != nil {
				logger.Trace("sender: sending message", "sub-protocol", info.subProtocol)
				err := s.host.send(peer, info.subProtocol, hsData.outboundMsg)
				if err != nil {
					logger.Error("failed to send message", "sub-protocol", info.subProtocol, "peer", peer, "error", err)
					return err
				}
				return nil
			}

			return nil
		}

		err := messageHandler(peer, msg)
		if err != nil {
			return err
		}

		// TODO: improve this by keeping track of who you've received/sent messages from
		// if !s.noGossip {
		// 	seen := s.gossip.hasSeen(msg)
		// 	if !seen {
		// 		s.broadcastExcluding(info, peer, msg)
		// 	}
		// }

		return nil
	}
}

// broadcastExcluding sends a message to each connected peer except the given peer
// Used for notifications sub-protocols to gossip a message
func (s *Service) broadcastExcluding(info *notificationsProtocol, excluding peer.ID, msg NotificationsMessage) {
	logger.Trace(
		"broadcasting message from notifications sub-protocol",
		"sub-protocol", info.subProtocol,
	)

	hs, err := info.getHandshake()
	if err != nil {
		logger.Error("failed to get handshake", "protocol", info.subProtocol, "error", err)
		return
	}

	for _, peer := range s.host.peers() { // TODO: check if stream is open, if not, open and send handshake
		if peer == excluding {
			continue
		}

		info.mapMu.RLock()
		defer info.mapMu.RUnlock()

		if hsData, has := info.handshakeData[peer]; !has || !hsData.received {
			info.handshakeData[peer] = &handshakeData{
				validated:   false,
				outboundMsg: msg,
			}

			logger.Trace("sending handshake", "protocol", info.subProtocol, "peer", peer, "message", hs)
			err = s.host.send(peer, info.subProtocol, hs)
		} else {
			// we've already completed the handshake with the peer, send message directly
			logger.Trace("sending message", "protocol", info.subProtocol, "peer", peer, "message", msg)
			err = s.host.send(peer, info.subProtocol, msg)
		}

		if err != nil {
			logger.Error("failed to send message to peer", "peer", peer, "error", err)
		}
	}
}
