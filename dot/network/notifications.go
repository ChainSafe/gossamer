// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/ChainSafe/gossamer/dot/peerset"
)

const handshakeTimeout = time.Second * 10

// Handshake is the interface all handshakes for notifications protocols must implement
type Handshake interface {
	Message
	IsValid() bool
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

	// NotificationsMessageBatchHandler is called when a (non-handshake) message is received over a notifications
	// stream in batch processing mode.
	NotificationsMessageBatchHandler = func(peer peer.ID, msg NotificationsMessage)
)

type batchMessage struct {
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
	peersData          *peersData
	maxSize            uint64
}

func newNotificationsProtocol(protocolID protocol.ID, handshakeGetter HandshakeGetter,
	handshakeDecoder HandshakeDecoder, handshakeValidator HandshakeValidator, maxSize uint64) *notificationsProtocol {
	return &notificationsProtocol{
		protocolID:         protocolID,
		getHandshake:       handshakeGetter,
		handshakeValidator: handshakeValidator,
		handshakeDecoder:   handshakeDecoder,
		peersData:          newPeersData(),
		maxSize:            maxSize,
	}
}

type handshakeData struct {
	received  bool
	validated bool
	handshake Handshake
	stream    network.Stream
}

func newHandshakeData(received, validated bool, stream network.Stream) *handshakeData {
	return &handshakeData{
		received:  received,
		validated: validated,
		stream:    stream,
	}
}

// createDecoder combines the notification message decoder and the handshake decoder. The combined
// decoder decodes using the handshake decoder if we already have handshake data stored for a given
// peer, otherwise it decodes using the notification message decoder.
func createDecoder(info *notificationsProtocol, handshakeDecoder HandshakeDecoder,
	messageDecoder MessageDecoder) messageDecoder {
	return func(in []byte, peer peer.ID, inbound bool) (Message, error) {
		// if we don't have handshake data on this peer, or we haven't received the handshake from them already,
		// assume we are receiving the handshake

		var hsData *handshakeData
		if inbound {
			hsData = info.peersData.getInboundHandshakeData(peer)
		} else {
			hsData = info.peersData.getOutboundHandshakeData(peer)
		}

		if hsData == nil || !hsData.received {
			return handshakeDecoder(in)
		}

		// otherwise, assume we are receiving the Message
		return messageDecoder(in)
	}
}

// createNotificationsMessageHandler returns a function that is called by the handler of *inbound* streams.
func (s *Service) createNotificationsMessageHandler(
	info *notificationsProtocol,
	notificationsMessageHandler NotificationsMessageHandler,
	batchHandler NotificationsMessageBatchHandler,
) messageHandler {
	return func(stream network.Stream, m Message) error {
		if m == nil || info == nil || info.handshakeValidator == nil || notificationsMessageHandler == nil {
			return nil
		}

		var (
			ok   bool
			msg  NotificationsMessage
			peer = stream.Conn().RemotePeer()
		)

		hs, ok := m.(Handshake)
		if ok {
			if !hs.IsValid() {
				return errInvalidRole
			}
			err := s.handleHandshake(info, stream, hs, peer)
			if err != nil {
				return fmt.Errorf("handling handshake: %w", err)
			}
			return nil
		}

		if msg, ok = m.(NotificationsMessage); !ok {
			return fmt.Errorf("%w: expected %T but got %T", errMessageTypeNotValid, (NotificationsMessage)(nil), msg)
		}

		hasSeen, err := s.gossip.hasSeen(msg)
		if err != nil {
			return fmt.Errorf("could not check if message was seen before: %w", err)
		}

		if hasSeen {
			// report peer if we get duplicate gossip message.
			s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
				Value:  peerset.DuplicateGossipValue,
				Reason: peerset.DuplicateGossipReason,
			}, peer)
			return nil
		}

		logger.Tracef("received message on notifications sub-protocol %s from peer %s, message is: %s",
			info.protocolID, stream.Conn().RemotePeer(), msg)

		if batchHandler != nil {
			batchHandler(peer, msg)
			return nil
		}

		propagate, err := notificationsMessageHandler(peer, msg)
		if err != nil {
			return err
		}

		if !propagate || s.noGossip {
			return nil
		}

		s.broadcastExcluding(info, peer, msg)
		return nil
	}
}

func (s *Service) handleHandshake(info *notificationsProtocol, stream network.Stream,
	hs Handshake, peer peer.ID) error {
	logger.Tracef("received handshake on notifications sub-protocol %s from peer %s, message is: %s",
		info.protocolID, stream.Conn().RemotePeer(), hs)

	// if we are the receiver and haven't received the handshake already, validate it
	// note: if this function is being called, it's being called via SetStreamHandler,
	// ie it is an inbound stream and we only send the handshake over it.
	// we do not send any other data over this stream, we would need to open a new outbound stream.
	hsData := info.peersData.getInboundHandshakeData(peer)
	if hsData != nil {
		return fmt.Errorf("%w: for peer id %s", errInboundHanshakeExists, peer)
	}

	logger.Tracef("receiver: validating handshake using protocol %s", info.protocolID)

	hsData = newHandshakeData(true, false, stream)
	info.peersData.setInboundHandshakeData(peer, hsData)

	err := info.handshakeValidator(peer, hs)
	if err != nil {
		return fmt.Errorf("%w from peer %s using protocol %s: %s",
			errCannotValidateHandshake, peer, info.protocolID, err)
	}

	hsData.validated = true
	info.peersData.setInboundHandshakeData(peer, hsData)

	// once validated, send back a handshake
	resp, err := info.getHandshake()
	if err != nil {
		return fmt.Errorf("failed to get handshake using protocol %s: %s", info.protocolID, err)
	}

	err = s.host.writeToStream(stream, resp)
	if err != nil {
		return fmt.Errorf("failed to send handshake to peer %s using protocol %s: %w", peer, info.protocolID, err)
	}

	logger.Tracef("receiver: sent handshake to peer %s using protocol %s", peer, info.protocolID)

	if err := stream.CloseWrite(); err != nil {
		return fmt.Errorf("failed to close stream for writing: %s", err)
	}

	return nil
}

func closeOutboundStream(info *notificationsProtocol, peerID peer.ID, stream network.Stream) {
	logger.Debugf(
		"cleaning up outbound handshake data for protocol=%s, peer=%s",
		stream.Protocol(),
		peerID,
	)

	info.peersData.deleteOutboundHandshakeData(peerID)

	err := stream.Close()
	if err != nil && err.Error() != ErrStreamReset.Error() {
		logger.Warnf("failed to close outbound stream: %s", err)
	}
}

func (s *Service) sendData(peer peer.ID, hs Handshake, info *notificationsProtocol, msg NotificationsMessage) {
	if info.handshakeValidator == nil {
		logger.Errorf("handshakeValidator is not set for protocol %s", info.protocolID)
		return
	}


	// Is there a chance we are acting prematurely here? Do we need to be connected, or at least be a certain
	// amount into the handshake process to be able to check this info?
	// I think the support protocol func is worth looking into potentially. Not sure what the
	// length check means
	support, err := s.host.supportsProtocol(peer, info.protocolID)
	if err != nil {
		logger.Errorf("could not check if protocol %s is supported by peer %s: %s", info.protocolID, peer, err)
		return
	}

	if !support {
		s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
			Value:  peerset.BadProtocolValue,
			Reason: peerset.BadProtocolReason,
		}, peer)

		return
	}

	stream, err := s.sendHandshake(peer, hs, info)
	if err != nil {
		logger.Debugf("failed to send handshake to peer %s on protocol %s: %s", peer, info.protocolID, err)
		return
	}

	_, isConsensusMsg := msg.(*ConsensusMessage)

	if s.host.messageCache != nil && s.host.messageCache.exists(peer, msg) && !isConsensusMsg {
		logger.Tracef("message has already been sent, ignoring: peer=%s msg=%s", peer, msg)
		return
	}

	// we've completed the handshake with the peer, send message directly
	logger.Tracef("sending message to peer %s using protocol %s: %s", peer, info.protocolID, msg)
	if err := s.host.writeToStream(stream, msg); err != nil {
		logger.Debugf("failed to send message to peer %s: %s", peer, err)

		// the stream was closed or reset, close it on our end and delete it from our peer's data
		if errors.Is(err, io.EOF) || errors.Is(err, network.ErrReset) {
			closeOutboundStream(info, peer, stream)
		}
		return
	} else if s.host.messageCache != nil {
		if _, err := s.host.messageCache.put(peer, msg); err != nil {
			logger.Errorf("failed to add message to cache for peer %s: %w", peer, err)
			return
		}
	}

	logger.Tracef("successfully sent message on protocol %s to peer %s: message=", info.protocolID, peer, msg)
	s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
		Value:  peerset.GossipSuccessValue,
		Reason: peerset.GossipSuccessReason,
	}, peer)
}

var errPeerDisconnected = errors.New("peer disconnected")

func (s *Service) sendHandshake(peer peer.ID, hs Handshake, info *notificationsProtocol) (network.Stream, error) {
	// multiple processes could each call this upcoming section, opening multiple streams and
	// sending multiple handshakes. thus, we need to have a per-peer and per-protocol lock

	// Note: we need to extract the mutex here since some sketchy test code
	// sometimes deletes it from its peerid->mutex map in info.peersData
	// so we cannot have a method on peersData to lock and unlock the mutex
	// from the map
	peerMutex := info.peersData.getMutex(peer)
	if peerMutex == nil {
		// Note: the only place the mutex is deleted is when the peer disconnects.
		// If it doesn't exist, the peer never connected either.
		return nil, fmt.Errorf("%w: peer id %s", errPeerDisconnected, peer)
	}

	peerMutex.Lock()
	defer peerMutex.Unlock()

	hsData := info.peersData.getOutboundHandshakeData(peer)
	switch {
	case hsData != nil && !hsData.validated:
		// peer has sent us an invalid handshake in the past, ignore
		return nil, errInvalidHandshakeForPeer
	case hsData != nil && hsData.validated:
		return hsData.stream, nil
	case hsData == nil:
		hsData = newHandshakeData(false, false, nil)
	}

	logger.Tracef("sending outbound handshake to peer %s on protocol %s, message: %s",
		peer, info.protocolID, hs)
	stream, err := s.host.send(peer, info.protocolID, hs)
	if err != nil {
		logger.Tracef("failed to send handshake to peer %s: %s", peer, err)
		// don't need to close the stream here, as it's nil!
		return nil, err
	}

	hsData.stream = stream

	hsTimer := time.NewTimer(handshakeTimeout)

	var resp Handshake
	select {
	case <-hsTimer.C:
		s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
			Value:  peerset.TimeOutValue,
			Reason: peerset.TimeOutReason,
		}, peer)

		logger.Tracef("handshake timeout reached for peer %s using protocol %s", peer, info.protocolID)
		closeOutboundStream(info, peer, stream)
		return nil, errHandshakeTimeout
	case hsResponse := <-s.readHandshake(stream, info.handshakeDecoder, info.maxSize):
		hsTimer.Stop()

		if hsResponse.err != nil {
			logger.Tracef("failed to read handshake from peer %s using protocol %s: %s", peer, info.protocolID, hsResponse.err)
			closeOutboundStream(info, peer, stream)
			return nil, hsResponse.err
		}

		resp = hsResponse.hs
		hsData.received = true
	}

	if err := stream.CloseRead(); err != nil {
		logger.Tracef("failed to close stream for reading: %s", err)
	}

	if err = info.handshakeValidator(peer, resp); err != nil {
		logger.Tracef("failed to validate handshake from peer %s using protocol %s: %s", peer, info.protocolID, err)
		hsData.validated = false
		hsData.stream = nil
		_ = stream.Reset()
		info.peersData.setOutboundHandshakeData(peer, hsData)
		// don't delete handshake data, as we want to store that the handshake for this peer was invalid
		// and not to exchange messages over this protocol with it
		return nil, err
	}

	hsData.validated = true
	hsData.handshake = resp
	info.peersData.setOutboundHandshakeData(peer, hsData)
	logger.Tracef("sender: validated handshake from peer %s using protocol %s", peer, info.protocolID)
	return hsData.stream, nil
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

		info.peersData.setMutex(peer)

		go s.sendData(peer, hs, info, msg)
	}
}

func (s *Service) readHandshake(stream network.Stream, decoder HandshakeDecoder, maxSize uint64,
) <-chan *handshakeReader {
	hsC := make(chan *handshakeReader)

	go func() {
		defer close(hsC)

		buffer := s.bufPool.Get().(*[]byte)
		defer s.bufPool.Put(buffer)

		tot, err := readStream(stream, buffer, maxSize)
		if err != nil {
			hsC <- &handshakeReader{hs: nil, err: err}
			return
		}

		msgBytes := *buffer
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
