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
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/services"

	log "github.com/ChainSafe/log15"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	discovery "github.com/libp2p/go-libp2p-discovery"
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

	cfg       *Config
	host      *host
	mdns      *mdns
	gossip    *gossip
	syncQueue *syncQueue

	notificationsProtocols map[byte]*notificationsProtocol // map of sub-protocol msg ID to protocol info
	notificationsMu        sync.RWMutex

	lightRequest   map[peer.ID]struct{} // set if we have sent a light request message to the given peer
	lightRequestMu sync.RWMutex

	// Service interfaces
	blockState         BlockState
	syncer             Syncer
	transactionHandler TransactionHandler

	// Configuration options
	noBootstrap bool
	noDiscover  bool
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
		cancel()
		return nil, err //nolint
	}

	if cfg.MinPeers == 0 {
		cfg.MinPeers = DefaultMinPeerCount
	}

	if cfg.MaxPeers == 0 {
		cfg.MaxPeers = DefaultMaxPeerCount
	}

	if cfg.MinPeers > cfg.MaxPeers {
		logger.Warn("min peers higher than max peers; setting to default")
		cfg.MinPeers = DefaultMinPeerCount
		cfg.MaxPeers = DefaultMaxPeerCount
	}

	// create a new host instance
	host, err := newHost(ctx, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	network := &Service{
		ctx:                    ctx,
		cancel:                 cancel,
		cfg:                    cfg,
		host:                   host,
		mdns:                   newMDNS(host),
		gossip:                 newGossip(),
		blockState:             cfg.BlockState,
		transactionHandler:     cfg.TransactionHandler,
		noBootstrap:            cfg.NoBootstrap,
		noMDNS:                 cfg.NoMDNS,
		syncer:                 cfg.Syncer,
		notificationsProtocols: make(map[byte]*notificationsProtocol),
		lightRequest:           make(map[peer.ID]struct{}),
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

	s.syncQueue = newSyncQueue(s)
	s.syncQueue.start()

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
		logger.Warn("failed to register notifications protocol", "sub-protocol", blockAnnounceID, "error", err)
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
		logger.Warn("failed to register notifications protocol", "sub-protocol", blockAnnounceID, "error", err)
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

	if !s.noDiscover {
		go func() {
			err = s.beginDiscovery()
			if err != nil {
				logger.Error("failed to begin DHT discovery", "error", err)
			}
		}()
	}

	time.Sleep(time.Millisecond * 500)

	logger.Info("started network service", "supported protocols", s.host.protocols())

	go s.logPeerCount()
	return nil
}

func (s *Service) logPeerCount() {
	for {
		logger.Debug("peer count", "num", s.host.peerCount(), "min", s.cfg.MinPeers, "max", s.cfg.MaxPeers)
		time.Sleep(time.Second * 30)
	}
}

func (s *Service) beginDiscovery() error {
	rd := discovery.NewRoutingDiscovery(s.host.dht)

	err := s.host.dht.Bootstrap(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// wait to connect to bootstrap peers
	time.Sleep(time.Second)

	go func() {
		peerCh, err := rd.FindPeers(s.ctx, s.cfg.ProtocolID)
		if err != nil {
			logger.Error("failed to begin finding peers via DHT", "err", err)
		}

		for peer := range peerCh {
			if peer.ID == s.host.id() {
				return
			}

			logger.Debug("found new peer via DHT", "peer", peer.ID)

			// found a peer, try to connect if we need more peers
			if s.host.peerCount() < s.cfg.MaxPeers {
				err = s.host.connect(peer)
				if err != nil {
					logger.Debug("failed to connect to discovered peer", "peer", peer.ID, "err", err)
				}
			} else {
				s.host.addToPeerstore(peer)
			}
		}
	}()

	logger.Debug("DHT discovery started!")
	return nil
}

// Stop closes running instances of the host and network services as well as
// the message channel from the network service to the core service (services that
// are dependent on the host instance should be closed first)
func (s *Service) Stop() error {
	s.syncQueue.stop()
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

	np := &notificationsProtocol{
		subProtocol:   sub,
		getHandshake:  handshakeGetter,
		handshakeData: make(map[peer.ID]*handshakeData),
	}
	s.notificationsProtocols[messageID] = np

	connMgr := s.host.h.ConnManager().(*ConnManager)
	connMgr.RegisterCloseHandler(s.host.protocolID+sub, func(peerID peer.ID) {
		np.mapMu.Lock()
		defer np.mapMu.Unlock()

		if _, ok := np.handshakeData[peerID]; ok {
			logger.Trace(
				"Cleaning up handshake data",
				"peer", peerID,
				"protocol", s.host.protocolID+sub,
			)
			delete(np.handshakeData, peerID)
		}
	})

	info := s.notificationsProtocols[messageID]

	s.host.registerStreamHandler(sub, func(stream libp2pnetwork.Stream) {
		logger.Trace("received stream", "sub-protocol", sub)
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

	logger.Info("registered notifications sub-protocol", "protocol", s.host.protocolID+sub)
	return nil
}

// IsStopped returns true if the service is stopped
func (s *Service) IsStopped() bool {
	return s.ctx.Err() != nil
}

// SendMessage implementation of interface to handle receiving messages
func (s *Service) SendMessage(msg NotificationsMessage) {
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
	s.notificationsMu.Lock()
	defer s.notificationsMu.Unlock()

	for msgID, prtl := range s.notificationsProtocols {
		if msg.Type() != msgID || prtl == nil {
			continue
		}

		s.broadcastExcluding(prtl, peer.ID(""), msg)
		return
	}

	logger.Warn("message not supported by any notifications protocol", "msg type", msg.Type())

	// TODO: deprecate
	// broadcast message to connected peers
	s.host.broadcast(msg)
}

// handleLightStream handles streams with the <protocol-id>/light/2 protocol ID
func (s *Service) handleLightStream(stream libp2pnetwork.Stream) {
	conn := stream.Conn()
	if conn == nil {
		logger.Error("Failed to get connection from stream")
		_ = stream.Close()
		return
	}

	peer := conn.RemotePeer()
	s.readStream(stream, peer, s.decodeLightMessage, s.handleLightMsg)
}

func (s *Service) decodeLightMessage(in []byte, peer peer.ID) (Message, error) {
	s.lightRequestMu.RLock()
	defer s.lightRequestMu.RUnlock()

	// check if we are the requester
	if _, requested := s.lightRequest[peer]; requested {
		// if we are, decode the bytes as a LightResponse
		msg := NewLightResponse()
		err := msg.Decode(in)
		return msg, err
	}

	// otherwise, decode bytes as LightRequest
	msg := NewLightRequest()
	err := msg.Decode(in)
	return msg, err
}

func (s *Service) readStream(stream libp2pnetwork.Stream, peer peer.ID, decoder messageDecoder, handler messageHandler) {
	// create buffer stream for non-blocking read
	r := bufio.NewReader(stream)

	var (
		msgBytes       []byte
		tot            uint64
		maxMessageSize uint64 = 1024 * 64 // TODO: determine actual max message size
	)

	for {
		length, err := readLEB128ToUint64(r)
		if err == io.EOF {
			continue
		} else if err != nil {
			logger.Debug("Failed to read LEB128 encoding", "protocol", stream.Protocol(), "error", err)
			_ = stream.Close()
			return
		}

		if length == 0 {
			continue
		}

		if length > maxMessageSize {
			logger.Warn("received message with size greater than max, discarding", "length", length)
			for {
				_, err = r.Discard(int(maxMessageSize))
				if err != nil {
					break
				}
			}
			continue
		}

		msgBytes = make([]byte, length)
		tot = uint64(0)
		for i := 0; i < maxReads; i++ {
			n, err := r.Read(msgBytes[tot:]) //nolint
			if err != nil {
				logger.Warn("Failed to read message from stream", "error", err)
				_ = stream.Close()
				return
			}

			tot += uint64(n)
			if tot == length {
				break
			}
		}

		if tot != length {
			logger.Debug("Failed to read entire message", "length", length, "read" /*n*/, tot)
			continue
		}

		if tot == 0 {
			continue
		}

		// decode message based on message type
		msg, err := decoder(msgBytes[:tot], peer)
		if err != nil {
			logger.Debug("Failed to decode message from peer", "peer", peer, "err", err)
			continue
		}

		logger.Trace(
			"Received message from peer",
			"host", s.host.id(),
			"peer", peer,
			"msg", msg.String(),
		)

		go func() {
			// handle message based on peer status and message type
			err = handler(peer, msg)
			if err != nil {
				logger.Warn("Failed to handle message from stream", "message", msg, "error", err)
				_ = stream.Close()
				return
			}
		}()
	}
}
func (s *Service) handleLightMsg(peer peer.ID, msg Message) error {
	lr, ok := msg.(*LightRequest)
	if !ok {
		logger.Warn("failed to get the request message from peer ", peer)
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
		logger.Warn("ignoring request without request data from peer {}", peer)
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
		logger.Warn("failed to send LightResponse message", "peer", peer, "err", err)
		s.host.closeStream(peer, lightID)
	}
	return err
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
