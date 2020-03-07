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
	"bufio"
	"context"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/services"

	log "github.com/ChainSafe/log15"
	"github.com/libp2p/go-libp2p-core/network"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

var _ services.Service = &Service{}

// Service describes a network service
type Service struct {
	ctx               context.Context
	cfg               *Config
	host              *host
	mdns              *mdns
	status            *status
	gossip            *gossip
	msgRec            <-chan Message
	msgSend           chan<- Message
	noBootstrap       bool
	noMDNS            bool
	noStatus          bool            // internal option
	noGossip          bool            // internal option
	requestedBlockIDs map[uint64]bool // track requested block id messages
}

// NewService creates a new network service from the configuration and message channels
func NewService(cfg *Config, msgSend chan<- Message, msgRec <-chan Message) (*Service, error) {
	ctx := context.Background()

	// build configuration
	err := cfg.build()
	if err != nil {
		return nil, err
	}

	// create a new host instance
	host, err := newHost(ctx, cfg)
	if err != nil {
		return nil, err
	}

	network := &Service{
		ctx:               ctx,
		cfg:               cfg,
		host:              host,
		mdns:              newMDNS(host),
		status:            newStatus(host),
		gossip:            newGossip(host),
		msgRec:            msgRec,
		msgSend:           msgSend,
		noBootstrap:       cfg.NoBootstrap,
		noMDNS:            cfg.NoMDNS,
		noStatus:          cfg.NoStatus,
		requestedBlockIDs: make(map[uint64]bool),
	}

	return network, err
}

// Start starts the network service
func (s *Service) Start() error {

	// receive messages from core service
	go s.receiveCoreMessages()

	s.host.registerConnHandler(s.handleConn)
	s.host.registerStreamHandler(s.handleStream)

	// log listening addresses to console
	for _, addr := range s.host.multiaddrs() {
		log.Info("[network] Started listening", "address", addr)
	}

	if !s.noBootstrap {
		s.host.bootstrap()
	}

	// TODO: ensure bootstrap has connected to bootnodes and addresses have been
	// registered by the host before mDNS attempts to connect to bootnodes

	if !s.noMDNS {
		s.mdns.start()
	}

	return nil
}

// Stop closes running instances of the host and network services as well as
// the message channel from the network service to the core service (services that
// are dependent on the host instance should be closed first)
func (s *Service) Stop() error {

	// close mDNS discovery service
	err := s.mdns.close()
	if err != nil {
		log.Error("[network] Failed to close mDNS discovery service", "error", err)
	}

	// close host and host services
	err = s.host.close()
	if err != nil {
		log.Error("[network] Failed to close host", "error", err)
	}

	// TODO: close s.msgSend, need channel close handling
	return nil
}

// receiveCoreMessages broadcasts messages from the core service
func (s *Service) receiveCoreMessages() {
	for {
		// receive message from core service
		msg := <-s.msgRec

		log.Debug(
			"[network] Broadcasting message from core service",
			"host", s.host.id(),
			"type", msg.GetType(),
		)

		// broadcast message to connected peers
		s.host.broadcast(msg)
	}
}

// handleConn starts processes that manage the connection
func (s *Service) handleConn(conn network.Conn) {
	// check if status is enabled
	if !s.noStatus {

		// get latest block header from block state
		latestBlock, err := s.cfg.BlockState.BestBlockHeader()
		if err != nil || (latestBlock == nil || latestBlock.Number == nil) {
			log.Error("[network] Failed to get chain head", "error", err)
			return
		}

		// update host status message
		msg := &StatusMessage{
			ProtocolVersion:     s.cfg.ProtocolVersion,
			MinSupportedVersion: s.cfg.MinSupportedVersion,
			Roles:               s.cfg.Roles,
			BestBlockNumber:     latestBlock.Number.Uint64(),
			BestBlockHash:       latestBlock.Hash(),
			GenesisHash:         s.cfg.BlockState.GenesisHash(),
			ChainStatus:         []byte{0}, // TODO
		}

		// update host status message
		s.status.setHostMessage(msg)

		// manage status messages for new connection
		s.status.handleConn(conn)
	}
}

// handleStream starts reading from the inbound message stream (substream with
// a matching protocol id that was opened by the connected peer) and continues
// reading until the inbound message stream is closed or reset.
func (s *Service) handleStream(stream libp2pnetwork.Stream) {
	conn := stream.Conn()
	if conn == nil {
		log.Error("[network] Failed to get connection from stream")
		return
	}

	peer := conn.RemotePeer()

	// create buffer stream for non-blocking read
	r := bufio.NewReader(stream)

	go s.readStream(r, peer)
	// the stream stays open until closed or reset
}

func (s *Service) readStream(r *bufio.Reader, peer peer.ID) {
	for {
		// TODO: re-add leb128 variable-length encoding #484
		length, err := readLEB128ToUint64(r)
		if err != nil {
			log.Error("[network] Failed to read LEB128 encoding", "error", err)
			return
		}

		msgBytes := make([]byte, length)
		n, err := r.Read(msgBytes)
		if err != nil {
			log.Error("[network] Failed to read message from stream", "error", err)
			return
		}

		if uint64(n) != length {
			log.Error("[network] Failed to read entire message", "length", length, "read", n)
			return
		}

		// decode message based on message type
		msg, err := decodeMessageBytes(msgBytes)
		if err != nil {
			log.Error("[network] Failed to decode message from peer", "peer", peer, "err", err)
			return // exit
		}

		// handle message based on peer status and message type
		s.handleMessage(peer, msg)
	}

}

// handleMessage handles the message based on peer status and message type
func (s *Service) handleMessage(peer peer.ID, msg Message) {
	log.Trace(
		"[network] Received message from peer",
		"host", s.host.id(),
		"peer", peer,
		"type", msg.GetType(),
	)

	if msg.GetType() != StatusMsgType {

		// check if status is disabled or peer status is confirmed
		if s.noStatus || s.status.confirmed(peer) {

			// send non-status message from confirmed peer to core service
			s.msgSend <- msg
		}

		// check if gossip is enabled
		if !s.noGossip {

			// handle non-status message from peer with gossip submodule
			s.gossip.handleMessage(msg, peer)
		}

	} else {

		// check if status is enabled
		if !s.noStatus {

			// handle status message from peer with status submodule
			s.status.handleMessage(peer, msg.(*StatusMessage))

			// send a block request message if peer best block number is greater than host best block number
			s.sendBlockRequestMessage(peer, msg.(*StatusMessage))
		}
	}
}

// Health returns information about host needed for the rpc server
func (s *Service) Health() *common.Health {
	return &common.Health{
		Peers:           s.host.peerCount(),
		IsSyncing:       false, // TODO
		ShouldHavePeers: !s.noBootstrap,
	}
}

// NetworkState returns information about host needed for the rpc server and the runtime
func (s *Service) NetworkState() *common.NetworkState {
	return &common.NetworkState{
		PeerID: s.host.id().String(),
	}
}

// Peers returns information about connected peers needed for the rpc server
func (s *Service) Peers() []common.PeerInfo {
	peers := []common.PeerInfo{}

	for _, p := range s.host.peers() {
		if s.status.confirmed(p) {
			msg := s.status.peerMessage[p]
			peers = append(peers, common.PeerInfo{
				PeerID:          p.String(),
				Roles:           msg.Roles,
				ProtocolVersion: msg.ProtocolVersion,
				BestHash:        msg.BestBlockHash,
				BestNumber:      msg.BestBlockNumber,
			})
		}
	}
	return peers
}
