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
	"errors"
	"os"
	"sync"
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
	lightID         = "/light/2"
	blockAnnounceID = "/block-announces/1"
	transactionsID  = "/transactions/1"
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

// Service describes a network service
type Service struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg                    *Config
	host                   *host
	mdns                   *mdns
	gossip                 *gossip
	requestTracker         *requestTracker
	errCh                  chan<- error
	notificationsProtocols map[byte]*notificationsProtocol // map of sub-protocol msg ID to protocol info
	notificationsMu        sync.RWMutex

	// Service interfaces
	blockState         BlockState
	syncer             Syncer
	transactionHandler TransactionHandler

	// Interface for inter-process communication
	messageHandler MessageHandler // TODO: remove with cleanup

	// Configuration options
	noBootstrap bool
	noMDNS      bool
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
		gossip:                 newGossip(),
		requestTracker:         newRequestTracker(logger),
		blockState:             cfg.BlockState,
		messageHandler:         cfg.MessageHandler,
		transactionHandler:     cfg.TransactionHandler,
		noBootstrap:            cfg.NoBootstrap,
		noMDNS:                 cfg.NoMDNS,
		syncer:                 cfg.Syncer,
		errCh:                  cfg.ErrChan,
		notificationsProtocols: make(map[byte]*notificationsProtocol),
	}

	return network, err
}

// SetSyncer sets the Syncer used by the network service
func (s *Service) SetSyncer(syncer Syncer) {
	s.syncer = syncer
}

// SetTransactionHandler sets the TransactionHandler used by the network service
func (s *Service) SetTransactionHandler(handler TransactionHandler) {
	s.transactionHandler = handler
}

// Start starts the network service
func (s *Service) Start() error {
	if s.syncer == nil {
		return errors.New("service Syncer is nil")
	}

	if s.transactionHandler == nil {
		return errors.New("service TransactionHandler is nil")
	}

	if s.IsStopped() {
		s.ctx, s.cancel = context.WithCancel(context.Background())
	}

	s.host.registerStreamHandler("", s.handleStream)
	s.host.registerStreamHandler(syncID, s.handleSyncStream)
	s.host.registerStreamHandler(lightID, s.handleLightStream)

	// register block announce protocol
	err := s.RegisterNotificationsProtocol(
		blockAnnounceID,
		BlockAnnounceMsgType,
		s.getBlockAnnounceHandshake,
		decodeBlockAnnounceHandshake,
		s.validateBlockAnnounceHandshake,
		decodeBlockAnnounceMessage,
		s.handleBlockAnnounceMessage,
	)
	if err != nil {
		logger.Error("failed to register notifications protocol", "sub-protocol", blockAnnounceID, "error", err)
	}

	// register transactions protocol
	err = s.RegisterNotificationsProtocol(
		transactionsID,
		TransactionMsgType,
		s.getTransactionHandshake,
		decodeTransactionHandshake,
		validateTransactionHandshake,
		decodeTransactionMessage,
		s.handleTransactionMessage,
	)
	if err != nil {
		logger.Error("failed to register notifications protocol", "sub-protocol", blockAnnounceID, "error", err)
	}

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

// RegisterNotificationsProtocol registers a protocol with the network service with the given handler
// messageID is a user-defined message ID for the message passed over this protocol.
func (s *Service) RegisterNotificationsProtocol(sub protocol.ID,
	messageID byte,
	handshakeGetter HandshakeGetter,
	handshakeDecoder HandshakeDecoder,
	handshakeValidator HandshakeValidator,
	messageDecoder MessageDecoder,
	messageHandler NotificationsMessageHandler,
) error {
	s.notificationsMu.Lock()
	defer s.notificationsMu.Unlock()

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

		decoder := createDecoder(info, handshakeDecoder, messageDecoder)
		handlerWithValidate := s.createNotificationsMessageHandler(info, handshakeValidator, messageHandler)

		s.readStream(stream, p, decoder, handlerWithValidate)
	})

	logger.Info("registered notifications sub-protocol", "sub-protocol", sub)
	return nil
}

// IsStopped returns true if the service is stopped
func (s *Service) IsStopped() bool {
	return s.ctx.Err() != nil
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
	s.notificationsMu.RLock()
	defer s.notificationsMu.RUnlock()

	for msgID, prtl := range s.notificationsProtocols {
		if msg.Type() != msgID || prtl == nil {
			continue
		}

		s.broadcastExcluding(prtl, peer.ID(""), msg)
		return
	}

	// broadcast message to connected peers
	s.host.broadcast(msg)
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

// handleLightStream handles streams with the <protocol-id>/light/2 protocol ID
func (s *Service) handleLightStream(stream libp2pnetwork.Stream) {
	conn := stream.Conn()
	if conn == nil {
		logger.Error("Failed to get connection from stream")
		return
	}

	peer := conn.RemotePeer()
	s.readStream(stream, peer, decodeMessageBytes, s.handleLightSyncMsg)
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
func (s *Service) handleLightSyncMsg(peer peer.ID, msg Message) error {
	lr, ok := msg.(*LightRequest)
	if !ok {
		logger.Error("failed to get the request message from peer ", peer)
		return nil
	}

	var resp LightResponse
	var err error
	switch {
	case lr.RmtCallRequest != nil:
		resp.RmtCallResponse, err = remoteCallResp(peer, lr.RmtCallRequest)
	case lr.RmtHeaderRequest != nil:
		resp.RmtHeaderResponse, err = remoteHeaderResp(peer, lr.RmtHeaderRequest)
	case lr.RmtChangesRequest != nil:
		resp.RmtChangeResponse, err = remoteChangeResp(peer, lr.RmtChangesRequest)
	case lr.RmtReadRequest != nil:
		resp.RmtReadResponse, err = remoteReadResp(peer, lr.RmtReadRequest)
	case lr.RmtReadChildRequest != nil:
		resp.RmtReadResponse, err = remoteReadChildResp(peer, lr.RmtReadChildRequest)
	default:
		logger.Error("ignoring request without request data from peer {}", peer)
		return nil
	}

	if err != nil {
		logger.Error("failed to get the response", "err", err)
		return err
	}

	// TODO(arijit): Remove once we implement the internal APIs. Added to increase code coverage.
	logger.Debug("LightResponse: ", resp.String())

	err = s.host.send(peer, lightID, &resp)
	if err != nil {
		logger.Error("failed to send LightResponse message", "peer", peer, "err", err)
	}
	return err
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
	if s.messageHandler == nil {
		logger.Crit("Failed to handle message", "error", "message handler is nil")
		return nil
	}
	s.messageHandler.HandleMessage(msg)

	return nil
}

// Health returns information about host needed for the rpc server
func (s *Service) Health() common.Health {

	return common.Health{
		Peers:           s.host.peerCount(),
		IsSyncing:       !s.syncer.IsSynced(),
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
		// TODO: update this based on BlockAnnounce handshake info
		peers = append(peers, common.PeerInfo{
			PeerID: p.String(),
			// Roles:           msg.Roles,
			// ProtocolVersion: msg.ProtocolVersion,
			// BestHash:        msg.BestBlockHash,
			// BestNumber:      msg.BestBlockNumber,
		})
	}
	return peers
}

// NodeRoles Returns the roles the node is running as.
func (s *Service) NodeRoles() byte {
	return s.cfg.Roles
}
