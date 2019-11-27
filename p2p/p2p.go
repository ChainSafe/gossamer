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
	log "github.com/ChainSafe/log15"

	"github.com/libp2p/go-libp2p-core/network"
	net "github.com/libp2p/go-libp2p-core/network"
)

// var _ module.P2pApi = &Service{}
// var _ services.Service = &Service{}

// `Service` describes a p2p service, which includes a host wrapper, message
// channels, and message mappings used for tracking received messages.
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

// TODO: Use generated status message
var statusMessage = &StatusMessage{
	ProtocolVersion:     0,
	MinSupportedVersion: 0,
	Roles:               0,
	BestBlockNumber:     0,
	BestBlockHash:       common.Hash{0x00},
	GenesisHash:         common.Hash{0x00},
	ChainStatus:         []byte{0},
}

// `NewService` creates a new p2p service from the provided configuration and message channels
func NewService(conf *Config, msgSend chan<- Message, msgRec <-chan Message) (*Service, error) {

	// Create background context
	ctx := context.Background()

	// Create new host instance
	h, err := newHost(ctx, conf)
	if err != nil {
		return nil, err
	}

	// Set service
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

	// Set stream handler on host instance
	h.registerStreamHandler(s.handleStream)

	return s, err
}

// `Start` starts the service
func (s *Service) Start() error {
	s.host.startMdns()
	s.host.bootstrap()
	s.host.logAddrs()

	// Create error channel. Errors from goroutines are received through an
	// error channel at the network level and never returned.
	// e := make(chan error)

	// Start sending status messages to connected peers
	go s.sendStatusMessages()

	// Start broadcasting received messages to all peers
	go s.broadcastReceivedMessages()

	return nil
}

// `Stop` stops the service
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

// `sendStatusMessages` starts a loop that sends the current network state as
// a status message to each connected peer every 5 seconds.
func (s *Service) sendStatusMessages() {
	for {

		// Send status messages every 5 seconds
		time.Sleep(5 * time.Second)

		// TODO: Use generated status message
		msg := statusMessage

		// Loop through connected peers
		for _, peer := range s.host.h.Network().Peers() {

			log.Debug(
				"sending message",
				"host", s.host.h.ID(),
				"peer", peer,
				"message", msg,
			)

			// Write status message to data stream
			err := s.host.send(peer, msg)
			if err != nil {
				log.Error("sending message", "error", err)
				break
			}
		}
	}
}

// `broadcastReceivedMessages` starts a loop that polls the `msgRec` channel,
// checks whether the message is a status message or the message has already
// been received, and then broadcasts new non-status messages to all peers.
func (s *Service) broadcastReceivedMessages() {
	for {

		// Receive message from babe
		msg := <-s.msgRec

		log.Debug(
			"received message",
			"host", s.host.id(),
			"channel", "msgRec",
			"message", msg,
		)

		// Broadcast new non-status messages
		err := s.Broadcast(msg)
		if err != nil {
			log.Error("broadcast", "error", err)
			break
		}
	}
}

// `Broadcast` checks whether a message has been added to the corresponding
// list of received messages. If the host has not received the message, the
// message will be saved to the list and then broadcasted to all peers.
func (s *Service) Broadcast(msg Message) (err error) {

	msgType := msg.GetType()

	switch msgType {
	case BlockRequestMsgType:
		if s.blockReqRec[msg.Id()] {
			return nil
		}
		s.blockReqRec[msg.Id()] = true
	case BlockResponseMsgType:
		if s.blockRespRec[msg.Id()] {
			return nil
		}
		s.blockRespRec[msg.Id()] = true
	case BlockAnnounceMsgType:
		if s.blockAnnounceRec[msg.Id()] {
			return nil
		}
		s.blockAnnounceRec[msg.Id()] = true
	case TransactionMsgType:
		if s.txMessageRec[msg.Id()] {
			return nil
		}
		s.txMessageRec[msg.Id()] = true
	default:
		log.Error("Invalid message type", "type", msgType)
		return nil
	}

	log.Debug(
		"broadcast",
		"host", s.host.id(),
		"message", msg,
	)

	s.host.broadcast(msg)

	return err
}

// `handleStream` parses the message written to the data stream and calls the
// associated message handler (status or non-status) based on message type.
func (s *Service) handleStream(stream net.Stream) {

	// Parse message and exit on error
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
		// Handle status message
		s.handleStreamStatus(stream, msg)
	} else {
		// Handle non-status message
		s.handleStreamNonStatus(stream, msg)
	}

}

// `handleStreamStatus` handles status messages written to the stream.
func (s *Service) handleStreamStatus(stream network.Stream, msg Message) {

	// TODO: Use generated status message
	hostStatus := statusMessage

	switch {

	case hostStatus.String() == msg.String():
		log.Debug(
			"status match",
			"host", stream.Conn().LocalPeer(),
			"peer", stream.Conn().RemotePeer(),
		)

		// TODO: Store peer status in peer metadata
		s.host.peerStatus[stream.Conn().RemotePeer()] = true

	default:
		log.Debug(
			"status mismatch",
			"host", stream.Conn().LocalPeer(),
			"peer", stream.Conn().RemotePeer(),
		)

		// TODO: Store peer status in peer metadata
		s.host.peerStatus[stream.Conn().RemotePeer()] = false

		// TODO: Drop peer if status mismatch

	}
}

// `handleStreamNonStatus` handles non-status messages written to the stream.
func (s *Service) handleStreamNonStatus(stream network.Stream, msg Message) {

	// TODO: Get peer status from peer metadata
	status := s.host.peerStatus[stream.Conn().RemotePeer()]

	// Exit if status message has not been confirmed
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

	// Check if message has already been received and broadcast if new message
	err := s.Broadcast(msg)
	if err != nil {
		log.Error(
			"broadcast message",
			"host", stream.Conn().LocalPeer(),
			"peer", stream.Conn().RemotePeer(),
			"protocol", stream.Protocol(),
			"error", err,
		)
	}
}

// `ID` returns the host id
func (s *Service) ID() string {
	return s.host.id()
}

// `Peers` returns connected peers
func (s *Service) Peers() []string {
	return PeerIdToStringArray(s.host.h.Network().Peers())
}

// `PeerCount` returns the number of connected peers
func (s *Service) PeerCount() int {
	return s.host.peerCount()
}

// `NoBootstrapping` returns true if bootstrapping is disabled, otherwise false
func (s *Service) NoBootstrapping() bool {
	return s.host.noBootstrap
}

// `parseMessage` reads message length, message type, decodes message based on
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

	// Decode message length using LEB128
	length := LEB128ToUint64([]byte{lengthByte})

	// Read message type byte
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

	// Read entire message
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

	// Decode message
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
