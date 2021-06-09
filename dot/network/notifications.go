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
	"time"
	"unsafe"

	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

var errCannotValidateHandshake = errors.New("failed to validate handshake")

const maxHandshakeSize = unsafe.Sizeof(BlockAnnounceHandshake{}) //nolint
const handshakeTimeout = time.Second * 10

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
	HandshakeValidator = func(peer.ID, Handshake) error

	// MessageDecoder is a custom decoder for a message
	MessageDecoder = func([]byte) (NotificationsMessage, error)

	// NotificationsMessageHandler is called when a (non-handshake) message is received over a notifications stream.
	NotificationsMessageHandler = func(peer peer.ID, msg NotificationsMessage) (propagate bool, err error)
)

type handshakeReader struct {
	hs  Handshake
	err error
}

type notificationsProtocol struct {
	protocolID         protocol.ID
	getHandshake       HandshakeGetter
	handshakeDecoder   HandshakeDecoder
	handshakeValidator HandshakeValidator

	inboundHandshakeData  *sync.Map //map[peer.ID]*handshakeData
	outboundHandshakeData *sync.Map //map[peer.ID]*handshakeData
}

func (n *notificationsProtocol) getHandshakeData(pid peer.ID, inbound bool) (handshakeData, bool) {
	var (
		data interface{}
		has  bool
	)

	if inbound {
		data, has = n.inboundHandshakeData.Load(pid)
	} else {
		data, has = n.outboundHandshakeData.Load(pid)
	}

	if !has {
		return handshakeData{}, false
	}

	return data.(handshakeData), true
}

type handshakeData struct {
	received  bool
	validated bool
	handshake Handshake
	stream    libp2pnetwork.Stream
	*sync.Mutex
}

func newHandshakeData(received, validated bool, stream libp2pnetwork.Stream) handshakeData {
	return handshakeData{
		received:  received,
		validated: validated,
		stream:    stream,
		Mutex:     new(sync.Mutex),
	}
}

func createDecoder(info *notificationsProtocol, handshakeDecoder HandshakeDecoder, messageDecoder MessageDecoder) messageDecoder {
	return func(in []byte, peer peer.ID, inbound bool) (Message, error) {
		// if we don't have handshake data on this peer, or we haven't received the handshake from them already,
		// assume we are receiving the handshake
		if hsData, has := info.getHandshakeData(peer, inbound); !has || !hsData.received {
			return handshakeDecoder(in)
		}

		// otherwise, assume we are receiving the Message
		return messageDecoder(in)
	}
}

func (s *Service) createNotificationsMessageHandler(info *notificationsProtocol, messageHandler NotificationsMessageHandler) messageHandler {
	return func(stream libp2pnetwork.Stream, m Message) error {
		if m == nil || info == nil || info.handshakeValidator == nil || messageHandler == nil {
			return nil
		}

		var (
			ok   bool
			msg  NotificationsMessage
			peer = stream.Conn().RemotePeer()
		)

		if msg, ok = m.(NotificationsMessage); !ok {
			return errors.New("message is not NotificationsMessage")
		}

		if msg.IsHandshake() {
			logger.Trace("received handshake on notifications sub-protocol", "protocol", info.protocolID,
				"message", msg,
				"peer", stream.Conn().RemotePeer(),
			)

			hs, ok := msg.(Handshake)
			if !ok {
				return errors.New("failed to convert message to Handshake")
			}

			// if we are the receiver and haven't received the handshake already, validate it
			// note: if this function is being called, it's being called via SetStreamHandler,
			// ie it is an inbound stream and we only send the handshake over it.
			// we do not send any other data over this stream, we would need to open a new outbound stream.
			if _, has := info.getHandshakeData(peer, true); !has {
				logger.Trace("receiver: validating handshake", "protocol", info.protocolID)

				hsData := newHandshakeData(true, false, stream)
				info.inboundHandshakeData.Store(peer, hsData)

				err := info.handshakeValidator(peer, hs)
				if err != nil {
					logger.Trace("failed to validate handshake", "protocol", info.protocolID, "peer", peer, "error", err)
					return errCannotValidateHandshake
				}

				hsData.validated = true
				info.inboundHandshakeData.Store(peer, hsData)

				// once validated, send back a handshake
				resp, err := info.getHandshake()
				if err != nil {
					logger.Warn("failed to get handshake", "protocol", info.protocolID, "error", err)
					return err
				}

				err = s.host.writeToStream(stream, resp)
				if err != nil {
					logger.Trace("failed to send handshake", "protocol", info.protocolID, "peer", peer, "error", err)
					return err
				}
				logger.Trace("receiver: sent handshake", "protocol", info.protocolID, "peer", peer)
			}

			return nil
		}

		logger.Trace("received message on notifications sub-protocol", "protocol", info.protocolID,
			"message", msg,
			"peer", stream.Conn().RemotePeer(),
		)

		propagate, err := messageHandler(peer, msg)
		if err != nil {
			return err
		}

		if !propagate || s.noGossip {
			return nil
		}

		seen := s.gossip.hasSeen(msg)
		if !seen {
			s.broadcastExcluding(info, peer, msg)
		}

		return nil
	}
}

func (s *Service) sendData(peer peer.ID, hs Handshake, info *notificationsProtocol, msg NotificationsMessage) {
	if support, err := s.host.supportsProtocol(peer, info.protocolID); err != nil || !support {
		return
	}

	hsData, has := info.getHandshakeData(peer, false)
	if has && !hsData.validated {
		// peer has sent us an invalid handshake in the past, ignore
		return
	}

	if !has || !hsData.received || hsData.stream == nil {
		if !has {
			hsData = newHandshakeData(false, false, nil)
		}

		hsData.Lock()
		defer hsData.Unlock()

		logger.Trace("sending outbound handshake", "protocol", info.protocolID, "peer", peer, "message", hs)
		stream, err := s.host.send(peer, info.protocolID, hs)
		if err != nil {
			logger.Trace("failed to send message to peer", "peer", peer, "error", err)
			return
		}

		hsData.stream = stream
		info.outboundHandshakeData.Store(peer, hsData)

		if info.handshakeValidator == nil {
			return
		}

		hsTimer := time.NewTimer(handshakeTimeout)

		var hs Handshake
		select {
		case <-hsTimer.C:
			logger.Trace("handshake timeout reached", "protocol", info.protocolID, "peer", peer)
			_ = stream.Close()
			info.outboundHandshakeData.Delete(peer)
			return
		case hsResponse := <-s.readHandshake(stream, info.handshakeDecoder):
			hsTimer.Stop()
			if hsResponse.err != nil {
				logger.Trace("failed to read handshake", "protocol", info.protocolID, "peer", peer, "error", err)
				_ = stream.Close()
				info.outboundHandshakeData.Delete(peer)
				return
			}

			hs = hsResponse.hs
			hsData.received = true
		}

		err = info.handshakeValidator(peer, hs)
		if err != nil {
			logger.Trace("failed to validate handshake", "protocol", info.protocolID, "peer", peer, "error", err)
			hsData.validated = false
			info.outboundHandshakeData.Store(peer, hsData)
			return
		}

		hsData.validated = true
		info.outboundHandshakeData.Store(peer, hsData)
		logger.Trace("sender: validated handshake", "protocol", info.protocolID, "peer", peer)
	}

	if s.host.messageCache != nil {
		added, err := s.host.messageCache.put(peer, msg)
		if err != nil {
			logger.Error("failed to add message to cache", "peer", peer, "error", err)
			return
		}

		if !added {
			return
		}
	}

	// we've completed the handshake with the peer, send message directly
	logger.Trace("sending message", "protocol", info.protocolID, "peer", peer, "message", msg)

	err := s.host.writeToStream(hsData.stream, msg)
	if err != nil {
		logger.Trace("failed to send message to peer", "peer", peer, "error", err)
	}
}

// broadcastExcluding sends a message to each connected peer except the given peer,
// and peers that have previously sent us the message or who we have already sent the message to.
// used for notifications sub-protocols to gossip a message
func (s *Service) broadcastExcluding(info *notificationsProtocol, excluding peer.ID, msg NotificationsMessage) {
	logger.Trace(
		"broadcasting message from notifications sub-protocol",
		"protocol", info.protocolID,
	)

	hs, err := info.getHandshake()
	if err != nil {
		logger.Error("failed to get handshake", "protocol", info.protocolID, "error", err)
		return
	}

	peers := s.host.peers()
	for _, peer := range peers {
		if peer == excluding {
			continue
		}

		go s.sendData(peer, hs, info, msg)
	}
}

func (s *Service) readHandshake(stream libp2pnetwork.Stream, decoder HandshakeDecoder) <-chan *handshakeReader {
	hsC := make(chan *handshakeReader)

	go func() {
		msgBytes := s.bufPool.get()
		defer func() {
			s.bufPool.put(&msgBytes)
			close(hsC)
		}()

		tot, err := readStream(stream, msgBytes[:])
		if err != nil {
			hsC <- &handshakeReader{hs: nil, err: err}
			return
		}

		hs, err := decoder(msgBytes[:tot])
		if err != nil {
			hsC <- &handshakeReader{hs: nil, err: err}
			return
		}

		hsC <- &handshakeReader{hs: hs, err: nil}
	}()

	return hsC
}
