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
	"reflect"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

var (
	errCannotValidateHandshake = errors.New("failed to validate handshake")
	maxHandshakeSize           = reflect.TypeOf(BlockAnnounceHandshake{}).Size()
)

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

	// NotificationsMessageBatchHandler is called when a (non-handshake) message is received over a notifications stream in batch processing mode.
	NotificationsMessageBatchHandler = func(peer peer.ID, msg NotificationsMessage) (batchMsgs []*BatchMessage, err error)
)

// BatchMessage is exported for the mocks of lib/grandpa/mocks/network.go
// to be able to compile.
// TODO: unexport if changing mock library to e.g. github.com/golang/gomock
type BatchMessage struct {
	msg  NotificationsMessage
	peer peer.ID
}

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

func (n *notificationsProtocol) getInboundHandshakeData(pid peer.ID) (handshakeData, bool) {
	var (
		data interface{}
		has  bool
	)

	data, has = n.inboundHandshakeData.Load(pid)
	if !has {
		return handshakeData{}, false
	}

	return data.(handshakeData), true
}

func (n *notificationsProtocol) getOutboundHandshakeData(pid peer.ID) (handshakeData, bool) {
	var (
		data interface{}
		has  bool
	)

	data, has = n.outboundHandshakeData.Load(pid)
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
		var (
			hsData handshakeData
			has    bool
		)

		if inbound {
			hsData, has = info.getInboundHandshakeData(peer)
		} else {
			hsData, has = info.getOutboundHandshakeData(peer)
		}

		if !has || !hsData.received {
			return handshakeDecoder(in)
		}

		// otherwise, assume we are receiving the Message
		return messageDecoder(in)
	}
}

func (s *Service) createNotificationsMessageHandler(info *notificationsProtocol, messageHandler NotificationsMessageHandler, batchHandler NotificationsMessageBatchHandler) messageHandler {
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
			logger.Tracef("received handshake on notifications sub-protocol %s from peer %s, message is: %s",
				info.protocolID, stream.Conn().RemotePeer(), msg)

			hs, ok := msg.(Handshake)
			if !ok {
				return errors.New("failed to convert message to Handshake")
			}

			// if we are the receiver and haven't received the handshake already, validate it
			// note: if this function is being called, it's being called via SetStreamHandler,
			// ie it is an inbound stream and we only send the handshake over it.
			// we do not send any other data over this stream, we would need to open a new outbound stream.
			if _, has := info.getInboundHandshakeData(peer); !has {
				logger.Tracef("receiver: validating handshake using protocol %s", info.protocolID)

				hsData := newHandshakeData(true, false, stream)
				info.inboundHandshakeData.Store(peer, hsData)

				err := info.handshakeValidator(peer, hs)
				if err != nil {
					logger.Tracef(
						"failed to validate handshake from peer %s using protocol %s: %s",
						peer, info.protocolID, err)
					return errCannotValidateHandshake
				}

				hsData.validated = true
				info.inboundHandshakeData.Store(peer, hsData)

				// once validated, send back a handshake
				resp, err := info.getHandshake()
				if err != nil {
					logger.Warnf("failed to get handshake using protocol %s: %s", info.protocolID, err)
					return err
				}

				err = s.host.writeToStream(stream, resp)
				if err != nil {
					logger.Tracef("failed to send handshake to peer %s using protocol %s: %s", peer, info.protocolID, err)
					return err
				}
				logger.Tracef("receiver: sent handshake to peer %s using protocol %s", peer, info.protocolID)
			}

			return nil
		}

		logger.Tracef("received message on notifications sub-protocol %s from peer %s, message is: %s",
			info.protocolID, stream.Conn().RemotePeer(), msg)

		var (
			propagate bool
			err       error
			msgs      []*BatchMessage
		)
		if batchHandler != nil {
			msgs, err = batchHandler(peer, msg)
			if err != nil {
				return err
			}

			propagate = len(msgs) > 0
		} else {
			propagate, err = messageHandler(peer, msg)
			if err != nil {
				return err
			}
			msgs = append(msgs, &BatchMessage{
				msg:  msg,
				peer: peer,
			})
		}

		if !propagate || s.noGossip {
			return nil
		}

		for _, data := range msgs {
			seen := s.gossip.hasSeen(data.msg)
			if !seen {
				s.broadcastExcluding(info, data.peer, data.msg)
			}

			// report peer if we get duplicate gossip message.
			s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
				Value:  peerset.DuplicateGossipValue,
				Reason: peerset.DuplicateGossipReason,
			}, peer)
		}

		return nil
	}
}

func (s *Service) sendData(peer peer.ID, hs Handshake, info *notificationsProtocol, msg NotificationsMessage) {
	if support, err := s.host.supportsProtocol(peer, info.protocolID); err != nil || !support {
		s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
			Value:  peerset.BadProtocolValue,
			Reason: peerset.BadProtocolReason,
		}, peer)

		return
	}

	hsData, has := info.getOutboundHandshakeData(peer)
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

		logger.Tracef("sending outbound handshake to peer %s using protocol %s, message: %s",
			peer, info.protocolID, hs)
		stream, err := s.host.send(peer, info.protocolID, hs)
		if err != nil {
			logger.Tracef("failed to send message to peer %s: %s", peer, err)
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
			s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
				Value:  peerset.TimeOutValue,
				Reason: peerset.TimeOutReason,
			}, peer)

			logger.Tracef("handshake timeout reached for peer %s using protocol %s", peer, info.protocolID)
			_ = stream.Close()
			info.outboundHandshakeData.Delete(peer)
			return
		case hsResponse := <-s.readHandshake(stream, info.handshakeDecoder):
			hsTimer.Stop()
			if hsResponse.err != nil {
				logger.Tracef("failed to read handshake from peer %s using protocol %s: %s", peer, info.protocolID, err)
				_ = stream.Close()
				info.outboundHandshakeData.Delete(peer)
				return
			}

			hs = hsResponse.hs
			hsData.received = true
		}

		err = info.handshakeValidator(peer, hs)
		if err != nil {
			logger.Tracef("failed to validate handshake from peer %s using protocol %s: %s", peer, info.protocolID, err)
			hsData.validated = false
			info.outboundHandshakeData.Store(peer, hsData)
			return
		}

		hsData.validated = true
		info.outboundHandshakeData.Store(peer, hsData)
		logger.Tracef("sender: validated handshake from peer %s using protocol %s", peer, info.protocolID)
	}

	if s.host.messageCache != nil {
		added, err := s.host.messageCache.put(peer, msg)
		if err != nil {
			logger.Errorf("failed to add message to cache for peer %s: %s", peer, err)
			return
		}

		// TODO: ensure grandpa stores *all* previously received votes and discards them
		// only when they are for already finalised rounds; currently this causes issues
		// because a vote might be received slightly too early, causing a round mismatch err,
		// causing grandpa to discard the vote. (#1855)
		_, isConsensusMsg := msg.(*ConsensusMessage)
		if !added && !isConsensusMsg {
			return
		}
	}

	// we've completed the handshake with the peer, send message directly
	logger.Tracef("sending message to peer %s using protocol %s: %s", peer, info.protocolID, msg)

	err := s.host.writeToStream(hsData.stream, msg)
	if err != nil {
		logger.Debugf("failed to send message to peer %s: %s", peer, err)
	}

	s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
		Value:  peerset.GossipSuccessValue,
		Reason: peerset.GossipSuccessReason,
	}, peer)
}

// broadcastExcluding sends a message to each connected peer except the given peer,
// and peers that have previously sent us the message or who we have already sent the message to.
// used for notifications sub-protocols to gossip a message
func (s *Service) broadcastExcluding(info *notificationsProtocol, excluding peer.ID, msg NotificationsMessage) {
	logger.Tracef("broadcasting message from notifications sub-protocol %s", info.protocolID)

	hs, err := info.getHandshake()
	if err != nil {
		logger.Errorf("failed to get handshake using protocol %s: %s", info.protocolID, err)
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
			s.bufPool.put(msgBytes)
			close(hsC)
		}()

		tot, err := readStream(stream, msgBytes[:])
		if err != nil {
			hsC <- &handshakeReader{hs: nil, err: err}
			return
		}

		hs, err := decoder(msgBytes[:tot])
		if err != nil {
			s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
				Value:  peerset.BadMessageValue,
				Reason: peerset.BadMessageReason,
			}, stream.Conn().RemotePeer())

			hsC <- &handshakeReader{hs: nil, err: err}
			return
		}

		hsC <- &handshakeReader{hs: hs, err: nil}
	}()

	return hsC
}
