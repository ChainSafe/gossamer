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

package p2p

import (
	"bufio"
	"context"
	"time"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/internal/services"
	log "github.com/ChainSafe/log15"

	"github.com/libp2p/go-libp2p-core/network"
	net "github.com/libp2p/go-libp2p-core/network"
)

var _ services.Service = &Service{}

// SendStatusInterval is the time between broadcasting status messages
const SendStatusInterval = 5 * time.Minute

// Service describes a p2p service
type Service struct {
	ctx              context.Context
	host             *host
	msgRec           <-chan Message
	msgSend          chan<- Message
	blockAnnounceRec map[string]bool
	blockReqRec      map[string]bool
	blockRespRec     map[string]bool
	txMessageRec     map[string]bool
}

// TODO: use generated status message
var statusMessage = &StatusMessage{
	ProtocolVersion:     0,
	MinSupportedVersion: 0,
	Roles:               0,
	BestBlockNumber:     0,
	BestBlockHash:       common.Hash{0x00},
	GenesisHash:         common.Hash{0x00},
	ChainStatus:         []byte{0},
}

// NewService creates a new p2p service using the provided configuration and message channels
func NewService(conf *Config, msgSend chan<- Message, msgRec <-chan Message) (*Service, error) {
	ctx := context.Background()

	h, err := newHost(ctx, conf)
	if err != nil {
		return nil, err
	}

	s := &Service{
		ctx:              ctx,
		host:             h,
		msgRec:           msgRec,
		msgSend:          msgSend,
		blockAnnounceRec: make(map[string]bool),
		blockReqRec:      make(map[string]bool),
		blockRespRec:     make(map[string]bool),
		txMessageRec:     make(map[string]bool),
	}

	h.registerStreamHandler(s.handleStream)

	return s, err
}

// Start starts the service
func (s *Service) Start() error {
	s.host.startMdns()
	s.host.bootstrap()
	s.host.printHostAddresses()

	// TODO: decide whether we need errors to surface here
	// e := make(chan error)

	// start sending status messages to each connected peer
	go s.broadcastStatusMessages()

	// start sending received messages to each connected peer
	go s.broadcastReceivedMessages()

	return nil
}

// Stop shuts down the host and the message send channel
func (s *Service) Stop() error {
	err := s.host.close()
	if err != nil {
		log.Error("close host", "error", err)
	}

	if s.msgSend != nil {
		close(s.msgSend)
	}

	return nil
}

// broadcastStatusMessages starts broadcasting status messages
func (s *Service) broadcastStatusMessages() {
	for {
		// wait between sending messages
		time.Sleep(SendStatusInterval)

		// TODO: use generated message
		msg := statusMessage

		// send message to each connected peer
		s.host.broadcast(msg)
	}
}

// broadcastReceivedMessages starts polling the msgRec channel for messages
// from the core service and broadcasts new messages to connected peers
func (s *Service) broadcastReceivedMessages() {
	for {
		// receive message from core service
		msg := <-s.msgRec

		log.Debug(
			"received message",
			"host", s.host.id(),
			"channel", "msgRec",
			"message", msg,
		)

		// check and store message, returns true if valid new message
		if !s.checkAndStoreReceived(msg) {
			log.Debug(
				"invalid message",
				"host", s.host.id(),
				"channel", "msgRec",
				"message", msg,
			)
		}

		// send new message to each connected peer
		s.host.broadcast(msg)
	}
}

// `checkAndStoreReceived` checks if message is new with valid type, storing
// the result and returning true if valid new message
func (s *Service) checkAndStoreReceived(msg Message) bool {

	msgType := msg.GetType()

	switch msgType {
	case BlockRequestMsgType:
		if s.blockReqRec[msg.Id()] {
			return false
		}
		s.blockReqRec[msg.Id()] = true
	case BlockResponseMsgType:
		if s.blockRespRec[msg.Id()] {
			return false
		}
		s.blockRespRec[msg.Id()] = true
	case BlockAnnounceMsgType:
		if s.blockAnnounceRec[msg.Id()] {
			return false
		}
		s.blockAnnounceRec[msg.Id()] = true
	case TransactionMsgType:
		if s.txMessageRec[msg.Id()] {
			return false
		}
		s.txMessageRec[msg.Id()] = true
	default:
		// do not broadcast status message type
		return false
	}

	return true
}

// handleStream parses the message written to the data stream and calls the
// associated message handler (status or non-status) based on message type.
func (s *Service) handleStream(stream net.Stream) {

	// parse message and return on error
	msg, _, err := parseMessage(stream)
	if err != nil {
		log.Debug("parse message", "error", err)
		return
	}

	log.Debug(
		"handle stream",
		"host", stream.Conn().LocalPeer(),
		"peer", stream.Conn().RemotePeer(),
		"protocol", stream.Protocol(),
		"message", msg,
	)

	if msg.GetType() == StatusMsgType {
		// handle status message type
		s.handleStreamStatus(stream, msg)
	} else {
		// handle other message types
		s.handleStreamNonStatus(stream, msg)
	}

}

// handleStreamStatus handles status messages written to the stream
func (s *Service) handleStreamStatus(stream network.Stream, msg Message) {

	// TODO: use generated status message
	hostStatus := statusMessage

	switch {

	case hostStatus.String() == msg.String():
		log.Debug(
			"status match",
			"host", stream.Conn().LocalPeer(),
			"peer", stream.Conn().RemotePeer(),
		)

		// TODO: store status in peer metadata
		s.host.peerStatus[stream.Conn().RemotePeer()] = true

	default:
		log.Debug(
			"status mismatch",
			"host", stream.Conn().LocalPeer(),
			"peer", stream.Conn().RemotePeer(),
		)

		// TODO: store status in peer metadata
		s.host.peerStatus[stream.Conn().RemotePeer()] = false

		// TODO: drop peer if status mismatch

	}
}

// handleStreamNonStatus handles non-status messages written to the stream
func (s *Service) handleStreamNonStatus(stream network.Stream, msg Message) {

	// TODO: get peer status from peer metadata
	status := s.host.peerStatus[stream.Conn().RemotePeer()]

	// exit if status message has not been confirmed
	if !status {
		log.Debug(
			"message blocked",
			"host", stream.Conn().LocalPeer(),
			"peer", stream.Conn().RemotePeer(),
			"protocol", stream.Protocol(),
			"message", msg,
		)
		return
	}

	// check and store message, returns true if valid new message
	if !s.checkAndStoreReceived(msg) {
		log.Debug(
			"invalid message",
			"host", s.host.id(),
			"channel", "msgRec",
			"message", msg,
		)
	}

	// send new message to each connected peer
	s.host.broadcast(msg)
}

// ID returns the host id
func (s *Service) ID() string {
	return s.host.id()
}

// Peers returns connected peers
func (s *Service) Peers() []string {
	return PeerIdToStringArray(s.host.h.Network().Peers())
}

// `eerCount returns the number of connected peers
func (s *Service) PeerCount() int {
	return s.host.peerCount()
}

// NoBootstrapping returns true if bootstrapping is disabled, otherwise false
func (s *Service) NoBootstrapping() bool {
	return s.host.noBootstrap
}

// `arseMessage reads message length, message type, decodes message based on
// type, and returns the decoded message
func parseMessage(stream net.Stream) (Message, []byte, error) {

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	lengthByte, err := rw.Reader.ReadByte()
	if err != nil {
		log.Error(
			"failed to read message length",
			"host", stream.Conn().LocalPeer(),
			"peer", stream.Conn().RemotePeer(),
			"error", err,
		)
		return nil, nil, err
	}

	// decode message length using LEB128
	length := LEB128ToUint64([]byte{lengthByte})

	// read message type byte
	_, err = rw.Reader.Peek(1)
	if err != nil {
		log.Error(
			"failed to read message type",
			"host", stream.Conn().LocalPeer(),
			"peer", stream.Conn().RemotePeer(),
			"err", err,
		)
		return nil, nil, err
	}

	// read entire message
	rawMsg, err := rw.Reader.Peek(int(length))
	if err != nil {
		log.Error(
			"failed to read message",
			"host", stream.Conn().LocalPeer(),
			"peer", stream.Conn().RemotePeer(),
			"err", err,
		)
		return nil, nil, err
	}

	// decode message
	msg, err := DecodeMessage(rw.Reader)
	if err != nil {
		log.Error(
			"failed to decode message",
			"host", stream.Conn().LocalPeer(),
			"peer", stream.Conn().RemotePeer(),
			"error", err,
		)
		return nil, nil, err
	}

	return msg, rawMsg, nil
}
