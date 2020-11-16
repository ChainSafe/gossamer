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
	"bytes"
	"context"
	"errors"
	"math/big"
	"os"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/services"

	log "github.com/ChainSafe/log15"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

const (
	// NetworkStateTimeout is the set time interval that we update network state
	NetworkStateTimeout = time.Minute

	// the following are sub-protocols used by the node
	syncID          = "/sync/2"
	blockAnnounceID = "/block-announces/1"
)

var (
	_        services.Service = &Service{}
	logger                    = log.New("pkg", "network")
	maxReads                  = 16
)

type (
	// messageDecoder is passed on readStream to decode the data from the stream into a message.
	// since messages are decoded based on context, this is different for every sub-protocol.
	messageDecoder = func([]byte, peer.ID) (Message, error)
	// messageHandler is passed on readStream to handle the resulting message. it should return an error only if the stream is to be closed
	messageHandler = func(peer peer.ID, msg Message) error
)

type notificationsProtocol struct {
	subProtocol   protocol.ID
	getHandshake  HandshakeGetter
	handshakeData map[peer.ID]*handshakeData
}

type handshakeData struct {
	received    bool
	validated   bool
	outboundMsg Message
}

// Service describes a network service
type Service struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg                    *Config
	host                   *host
	mdns                   *mdns
	status                 *status
	gossip                 *gossip
	requestTracker         *requestTracker
	errCh                  chan<- error
	notificationsProtocols map[byte]*notificationsProtocol // map of sub-protocol msg ID to protocol info

	// Service interfaces
	blockState   BlockState
	networkState NetworkState
	syncer       Syncer

	// Interface for inter-process communication
	messageHandler MessageHandler

	// Configuration options
	noBootstrap bool
	noMDNS      bool
	noStatus    bool // internal option
	noGossip    bool // internal option
}

// NewService creates a new network service from the configuration and message channels
func NewService(cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background()) //nolint

	h := log.StreamHandler(os.Stdout, log.TerminalFormat())
	h = log.CallerFileHandler(h)
	logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, h))
	cfg.logger = logger

	// build configuration
	err := cfg.build()
	if err != nil {
		return nil, err //nolint
	}

	if cfg.Syncer == nil {
		return nil, errors.New("cannot have nil Syncer")
	}

	// create a new host instance
	host, err := newHost(ctx, cfg)
	if err != nil {
		return nil, err
	}

	network := &Service{
		ctx:                    ctx,
		cancel:                 cancel,
		cfg:                    cfg,
		host:                   host,
		mdns:                   newMDNS(host),
		status:                 newStatus(host),
		gossip:                 newGossip(host),
		requestTracker:         newRequestTracker(logger),
		blockState:             cfg.BlockState,
		networkState:           cfg.NetworkState,
		messageHandler:         cfg.MessageHandler,
		noBootstrap:            cfg.NoBootstrap,
		noMDNS:                 cfg.NoMDNS,
		noStatus:               cfg.NoStatus,
		syncer:                 cfg.Syncer,
		errCh:                  cfg.ErrChan,
		notificationsProtocols: make(map[byte]*notificationsProtocol),
	}

	return network, err
}

// Start starts the network service
func (s *Service) Start() error {
	if s.IsStopped() {
		s.ctx, s.cancel = context.WithCancel(context.Background())
	}

	// update network state
	go s.updateNetworkState()

	s.host.registerConnHandler(s.handleConn)
	s.host.registerStreamHandler("", s.handleStream)
	s.host.registerStreamHandler(syncID, s.handleSyncStream)

	// register block announce protocol
	s.RegisterNotificationsProtocol(
		blockAnnounceID,
		BlockAnnounceMsgType,
		s.getBlockAnnounceHandshake,
		decodeBlockAnnounceHandshake,
		s.validateBlockAnnounceHandshake,
		decodeBlockAnnounceMessage,
		s.handleBlockAnnounceMessage,
	)

	// log listening addresses to console
	for _, addr := range s.host.multiaddrs() {
		logger.Info("Started listening", "address", addr)
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
	s.cancel()

	// close mDNS discovery service
	err := s.mdns.close()
	if err != nil {
		logger.Error("Failed to close mDNS discovery service", "error", err)
	}

	// close host and host services
	err = s.host.close()
	if err != nil {
		logger.Error("Failed to close host", "error", err)
	}

	return nil
}

// RegisterNotificationsProtocol registers a protocol with the network service with the given handler and returns its msg ID
func (s *Service) RegisterNotificationsProtocol(sub protocol.ID,
	messageID byte,
	//handshakeID byte,
	handshakeGetter HandshakeGetter,
	handshakeDecoder HandshakeDecoder,
	handshakeValidator HandshakeValidator,
	messageDecoder MessageDecoder,
	messageHandler NotificationsMessageHandler,
) error {
	if _, has := s.notificationsProtocols[messageID]; has {
		return errors.New("notifications protocol with message type already exists")
	}

	s.notificationsProtocols[messageID] = &notificationsProtocol{
		subProtocol:   sub,
		getHandshake:  handshakeGetter,
		handshakeData: make(map[peer.ID]*handshakeData),
	}

	info := s.notificationsProtocols[messageID]

	s.host.registerStreamHandler(sub, func(stream libp2pnetwork.Stream) {
		logger.Info("received stream", "sub-protocol", sub)
		conn := stream.Conn()
		if conn == nil {
			logger.Error("Failed to get connection from stream")
			return
		}

		p := conn.RemotePeer()

		decoder := func(in []byte, peer peer.ID) (Message, error) {
			r := &bytes.Buffer{}
			_, err := r.Write(in)
			if err != nil {
				return nil, err
			}

			// if we don't have handshake data on this peer, or we haven't received the handshake from them already,
			// assume we are receiving the handshake
			if hsData, has := info.handshakeData[peer]; !has || !hsData.received {
				return handshakeDecoder(r)
			}

			// otherwise, assume we are receiving the Message
			return messageDecoder(r)
		}

		handlerWithValidate := func(peer peer.ID, msg Message) error {
			logger.Info("received message on notifications sub-protocol", "sub-protocol", info.subProtocol, "message", msg)

			hs, err := handshakeGetter()
			if err != nil {
				logger.Error("failed to gwt handshake", "sub-protocol", info.subProtocol, "error", err)
				return err
			}

			if msg.Type() == hs.Type() {
				// if we are the receiver and haven't received the handshake already, validate it
				if _, has := info.handshakeData[peer]; !has {
					logger.Trace("receiver: validating handshake", "sub-protocol", info.subProtocol)
					err := handshakeValidator(hs)
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
					resp, err := handshakeGetter()
					if err != nil {
						logger.Error("failed to get handshake", "sub-protocol", info.subProtocol, "error", err)
						return nil
					}

					err = s.host.send(peer, sub, resp)
					if err != nil {
						logger.Error("failed to send handshake", "sub-protocol", info.subProtocol, "peer", peer, "error", err)
					}
					logger.Trace("receiver: sent handshake", "sub-protocol", info.subProtocol, "peer", peer)
				}

				// if we are the initiator and haven't received the handshake already, validate it
				if hsData, has := info.handshakeData[peer]; has && !hsData.validated {
					logger.Trace("sender: validating handshake")
					err := handshakeValidator(hs)
					if err != nil {
						logger.Error("failed to validate handshake", "sub-protocol", info.subProtocol, "peer", peer, "error", err)
						// TODO: also delete on stream close
						delete(info.handshakeData, peer)
						return errCannotValidateHandshake
					}

					info.handshakeData[peer].validated = true
					info.handshakeData[peer].received = true
					logger.Trace("sender: validated handshake", "sub-protocol", info.subProtocol, "peer", peer)
				} else if hsData.received {
					return nil
				}

				// if we are the initiator, send the message
				if hsData, has := info.handshakeData[peer]; has && hsData.validated && hsData.received && hsData.outboundMsg != nil {
					logger.Trace("sender: sending message", "sub-protocol", info.subProtocol)
					err := s.host.send(peer, sub, hsData.outboundMsg)
					if err != nil {
						logger.Error("failed to send message", "sub-protocol", info.subProtocol, "peer", peer, "error", err)
					}
					return nil
				}
			}

			return messageHandler(p, msg)
		}

		s.readStream(stream, p, decoder, handlerWithValidate)
	})

	logger.Info("registered notifications sub-protocol", "sub-protocol", sub)
	return nil
}

// IsStopped returns true if the service is stopped
func (s *Service) IsStopped() bool {
	return s.ctx.Err() != nil
}

// updateNetworkState updates the network state at the set time interval
func (s *Service) updateNetworkState() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(NetworkStateTimeout):
			s.networkState.SetHealth(s.Health())
			s.networkState.SetNetworkState(s.NetworkState())
			s.networkState.SetPeers(s.Peers())
		}
	}
}

// SendMessage implementation of interface to handle receiving messages
func (s *Service) SendMessage(msg Message) {
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
	for msgID, prtl := range s.notificationsProtocols {
		if msg.Type() != msgID {
			continue
		}

		hs, err := prtl.getHandshake()
		if err != nil {
			logger.Error("failed to get handshake", "protocol", prtl.subProtocol, "error", err)
			return
		}

		for _, peer := range s.host.peers() { // TODO: check if stream is open, if not, open and send handshake
			if _, has := prtl.handshakeData[peer]; !has {
				prtl.handshakeData[peer] = &handshakeData{
					validated:   false,
					outboundMsg: msg,
				}

				logger.Trace("sending handshake", "protocol", prtl.subProtocol, "peer", peer, "message", hs)
				err = s.host.send(peer, prtl.subProtocol, hs)
			} else {
				// we've already completed the handshake with the peer, send BlockAnnounce directly
				err = s.host.send(peer, prtl.subProtocol, msg)
			}

			if err != nil {
				logger.Error("failed to send message to peer", "peer", peer, "error", err)
			}
		}

		return
	}

	// broadcast message to connected peers
	s.host.broadcast(msg)
}

// handleConn starts processes that manage the connection
func (s *Service) handleConn(conn libp2pnetwork.Conn) {
	// check if status is enabled
	if !s.noStatus {

		// get latest block header from block state
		latestBlock, err := s.blockState.BestBlockHeader()
		if err != nil || (latestBlock == nil || latestBlock.Number == nil) {
			logger.Error("Failed to get chain head", "error", err)
			return
		}

		// update host status message
		msg := &StatusMessage{
			ProtocolVersion:     s.cfg.ProtocolVersion,
			MinSupportedVersion: s.cfg.MinSupportedVersion,
			Roles:               s.cfg.Roles,
			BestBlockNumber:     latestBlock.Number.Uint64(),
			BestBlockHash:       latestBlock.Hash(),
			GenesisHash:         s.blockState.GenesisHash(),
			ChainStatus:         []byte{0}, // TODO
		}

		// update host status message
		s.status.setHostMessage(msg)

		// manage status messages for new connection
		s.status.handleConn(conn)
	}
}

// handleStream starts reading from the inbound message stream and continues
// reading until the inbound message stream is closed or reset.
func (s *Service) handleStream(stream libp2pnetwork.Stream) {
	conn := stream.Conn()
	if conn == nil {
		logger.Error("Failed to get connection from stream")
		return
	}

	peer := conn.RemotePeer()
	s.readStream(stream, peer, decodeMessageBytes, s.handleMessage)
	// the stream stays open until closed or reset
}

// handleSyncStream handles streams with the <protocol-id>/sync/2 protocol ID
func (s *Service) handleSyncStream(stream libp2pnetwork.Stream) {
	conn := stream.Conn()
	if conn == nil {
		logger.Error("Failed to get connection from stream")
		return
	}

	peer := conn.RemotePeer()
	s.readStream(stream, peer, decodeMessageBytes, s.handleSyncMessage)
	// the stream stays open until closed or reset
}

func (s *Service) readStream(stream libp2pnetwork.Stream, peer peer.ID, decoder messageDecoder, handler messageHandler) {
	// create buffer stream for non-blocking read
	r := bufio.NewReader(stream)

	for {
		length, err := readLEB128ToUint64(r)
		if err != nil {
			logger.Error("Failed to read LEB128 encoding", "error", err)
			_ = stream.Close()
			s.errCh <- err
			return
		}

		if length == 0 {
			continue
		}

		msgBytes := make([]byte, length)
		tot := uint64(0)
		for i := 0; i < maxReads; i++ {
			n, err := r.Read(msgBytes[tot:]) //nolint
			if err != nil {
				logger.Error("Failed to read message from stream", "error", err)
				_ = stream.Close()
				s.errCh <- err
				return
			}

			tot += uint64(n)
			if tot == length {
				break
			}
		}

		if tot != length {
			logger.Error("Failed to read entire message", "length", length, "read" /*n*/, tot)
			continue
		}

		// decode message based on message type
		msg, err := decoder(msgBytes, peer)
		if err != nil {
			logger.Error("Failed to decode message from peer", "peer", peer, "err", err)
			continue
		}

		logger.Trace(
			"Received message from peer",
			"host", s.host.id(),
			"peer", peer,
			"type", msg.Type(),
		)

		// handle message based on peer status and message type
		err = handler(peer, msg)
		if err != nil {
			logger.Error("Failed to handle message from stream", "message", msg, "error", err)
			_ = stream.Close()
			s.errCh <- err
			return
		}
	}
}

// handleSyncMessage handles synchronization message types (BlockRequest and BlockResponse)
func (s *Service) handleSyncMessage(peer peer.ID, msg Message) error {
	if msg == nil {
		return nil
	}

	// if it's a BlockResponse with an ID corresponding to a BlockRequest we sent, forward
	// message to the sync service
	if resp, ok := msg.(*BlockResponseMessage); ok && s.requestTracker.hasRequestedBlockID(resp.ID) {
		s.requestTracker.removeRequestedBlockID(resp.ID)
		req := s.syncer.HandleBlockResponse(resp)
		if req != nil {
			s.requestTracker.addRequestedBlockID(req.ID)
			err := s.host.send(peer, syncID, req)
			if err != nil {
				logger.Error("failed to send BlockRequest message", "peer", peer)
			}
		}
	}

	// if it's a BlockRequest, call core for processing
	if req, ok := msg.(*BlockRequestMessage); ok {
		resp, err := s.syncer.CreateBlockResponse(req)
		if err != nil {
			logger.Debug("cannot create response for request", "id", req.ID)
			return nil
		}

		err = s.host.send(peer, syncID, resp)
		if err != nil {
			logger.Error("failed to send BlockResponse message", "peer", peer)
		}
	}

	return nil
}

// handleMessage handles the message based on peer status and message type
// TODO: deprecate this handler, messages will be handled via their sub-protocols
func (s *Service) handleMessage(peer peer.ID, msg Message) error {
	if msg.Type() != StatusMsgType {

		// check if status is disabled or peer status is confirmed
		if s.noStatus || s.status.confirmed(peer) {
			if s.messageHandler == nil {
				logger.Crit("Failed to handle message", "error", "message handler is nil")
				return nil
			}
			s.messageHandler.HandleMessage(msg)
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

			// check if peer status confirmed
			if s.status.confirmed(peer) {

				// send a block request message if peer best block number is greater than host best block number
				req := s.handleStatusMesssage(msg.(*StatusMessage))
				if req != nil {
					s.requestTracker.addRequestedBlockID(req.ID)
					err := s.host.send(peer, syncID, req)
					if err != nil {
						logger.Error("failed to send BlockRequest message", "peer", peer)
					}
				}
			}
		}
	}

	return nil
}

// handleStatusMesssage returns a block request message if peer best block
// number is greater than host best block number
func (s *Service) handleStatusMesssage(statusMessage *StatusMessage) *BlockRequestMessage {
	// get latest block header from block state
	latestHeader, err := s.blockState.BestBlockHeader()
	if err != nil {
		logger.Error("Failed to get best block header from block state", "error", err)
		return nil
	}

	bestBlockNum := big.NewInt(int64(statusMessage.BestBlockNumber))

	// check if peer block number is greater than host block number
	if latestHeader.Number.Cmp(bestBlockNum) == -1 {
		logger.Debug("sending new block to syncer", "number", statusMessage.BestBlockNumber)
		return s.syncer.HandleSeenBlocks(bestBlockNum)
	}

	return nil
}

// Health returns information about host needed for the rpc server
func (s *Service) Health() common.Health {
	return common.Health{
		Peers:           s.host.peerCount(),
		IsSyncing:       false, // TODO
		ShouldHavePeers: !s.noBootstrap,
	}
}

// NetworkState returns information about host needed for the rpc server and the runtime
func (s *Service) NetworkState() common.NetworkState {
	return common.NetworkState{
		PeerID:     s.host.id().String(),
		Multiaddrs: s.host.multiaddrs(),
	}
}

// Peers returns information about connected peers needed for the rpc server
func (s *Service) Peers() []common.PeerInfo {
	peers := []common.PeerInfo{}

	for _, p := range s.host.peers() {
		if s.status.confirmed(p) {
			if m, ok := s.status.peerMessage.Load(p); ok {
				msg, ok := m.(*StatusMessage)
				if !ok {
					return peers
				}

				peers = append(peers, common.PeerInfo{
					PeerID:          p.String(),
					Roles:           msg.Roles,
					ProtocolVersion: msg.ProtocolVersion,
					BestHash:        msg.BestBlockHash,
					BestNumber:      msg.BestBlockNumber,
				})
			}
		}
	}
	return peers
}

// NodeRoles Returns the roles the node is running as.
func (s *Service) NodeRoles() byte {
	return s.cfg.Roles
}

//SetMessageHandler sets the given MessageHandler for this service
func (s *Service) SetMessageHandler(handler MessageHandler) {
	s.messageHandler = handler
}
