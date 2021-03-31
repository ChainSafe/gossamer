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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	gssmrmetrics "github.com/ChainSafe/gossamer/dot/metrics"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ethereum/go-ethereum/metrics"

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
	messageHandler = func(stream libp2pnetwork.Stream, msg Message) error
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

	network.syncQueue = newSyncQueue(network)

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

	connMgr := s.host.h.ConnManager().(*ConnManager)
	connMgr.registerDisconnectHandler(func(p peer.ID) {
		s.syncQueue.peerScore.Delete(p)
	})

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
		false,
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
		false,
	)
	if err != nil {
		logger.Warn("failed to register notifications protocol", "sub-protocol", blockAnnounceID, "error", err)
	}

	// since this opens block announce streams, it should happen after the protocol is registered
	s.host.h.Network().SetConnHandler(s.handleConn)

	// log listening addresses to console
	for _, addr := range s.host.multiaddrs() {
		logger.Info("Started listening", "address", addr)
	}

	if !s.noBootstrap {
		s.host.bootstrap()
	}

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
	s.syncQueue.start()

	if s.cfg.PublishMetrics {
		go s.collectNetworkMetrics()
	}

	go s.logPeerCount()
	return nil
}

func (s *Service) collectNetworkMetrics() {
	metrics.Enabled = true
	for {
		peerCount := metrics.GetOrRegisterGauge("network/node/peerCount", metrics.DefaultRegistry)
		totalConn := metrics.GetOrRegisterGauge("network/node/totalConnection", metrics.DefaultRegistry)
		networkLatency := metrics.GetOrRegisterGauge("network/node/latency", metrics.DefaultRegistry)
		syncedBlocks := metrics.GetOrRegisterGauge("service/blocks/sync", metrics.DefaultRegistry)

		peerCount.Update(int64(s.host.peerCount()))
		totalConn.Update(int64(len(s.host.h.Network().Conns())))
		networkLatency.Update(int64(s.host.h.Peerstore().LatencyEWMA(s.host.id())))

		num, err := s.blockState.BestBlockNumber()
		if err != nil {
			syncedBlocks.Update(0)
		} else {
			syncedBlocks.Update(num.Int64())
		}

		time.Sleep(gssmrmetrics.Refresh)
	}
}

func (s *Service) logPeerCount() {
	for {
		logger.Debug("peer count", "num", s.host.peerCount(), "min", s.cfg.MinPeers, "max", s.cfg.MaxPeers)
		time.Sleep(time.Second * 30)
	}
}

func (s *Service) handleConn(conn libp2pnetwork.Conn) {
	// give new peers a slight weight
	s.syncQueue.updatePeerScore(conn.RemotePeer(), 1)

	s.notificationsMu.Lock()
	defer s.notificationsMu.Unlock()

	info, has := s.notificationsProtocols[BlockAnnounceMsgType]
	if !has {
		// this shouldn't happen
		logger.Warn("block announce protocol is not yet registered!")
		return
	}

	// open block announce substream
	hs, err := info.getHandshake()
	if err != nil {
		logger.Warn("failed to get handshake", "protocol", blockAnnounceID, "error", err)
		return
	}

	info.mapMu.RLock()
	defer info.mapMu.RUnlock()

	peer := conn.RemotePeer()
	if hsData, has := info.getHandshakeData(peer); !has || !hsData.received { //nolint
		info.handshakeData.Store(peer, &handshakeData{
			validated: false,
		})

		logger.Trace("sending handshake", "protocol", info.protocolID, "peer", peer, "message", hs)
		err = s.host.send(peer, info.protocolID, hs)
		if err != nil {
			logger.Trace("failed to send block announce handshake to peer", "peer", peer, "error", err)
		}
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
	overwriteProtocol bool,
) error {
	s.notificationsMu.Lock()
	defer s.notificationsMu.Unlock()

	if _, has := s.notificationsProtocols[messageID]; has {
		return errors.New("notifications protocol with message type already exists")
	}

	var protocolID protocol.ID
	if overwriteProtocol {
		protocolID = sub
	} else {
		protocolID = s.host.protocolID + sub
	}

	np := &notificationsProtocol{
		protocolID:    protocolID,
		getHandshake:  handshakeGetter,
		handshakeData: new(sync.Map),
	}
	s.notificationsProtocols[messageID] = np

	connMgr := s.host.h.ConnManager().(*ConnManager)
	connMgr.registerCloseHandler(protocolID, func(peerID peer.ID) {
		np.mapMu.Lock()
		defer np.mapMu.Unlock()

		if _, ok := np.getHandshakeData(peerID); ok {
			logger.Trace(
				"Cleaning up handshake data",
				"peer", peerID,
				"protocol", protocolID,
			)
			np.handshakeData.Delete(peerID)
		}
	})

	info := s.notificationsProtocols[messageID]

	decoder := createDecoder(info, handshakeDecoder, messageDecoder)
	handlerWithValidate := s.createNotificationsMessageHandler(info, handshakeValidator, messageHandler)

	s.host.registerStreamHandlerWithOverwrite(sub, overwriteProtocol, func(stream libp2pnetwork.Stream) {
		logger.Trace("received stream", "sub-protocol", sub)
		conn := stream.Conn()
		if conn == nil {
			logger.Error("Failed to get connection from stream")
			return
		}

		p := conn.RemotePeer()
		s.readStream(stream, p, decoder, handlerWithValidate)
	})

	logger.Info("registered notifications sub-protocol", "protocol", protocolID)
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

	logger.Error("message not supported by any notifications protocol", "msg type", msg.Type())
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
	var (
		maxMessageSize uint64 = maxBlockResponseSize // TODO: determine actual max message size
		msgBytes              = make([]byte, maxMessageSize)
	)

	for {
		tot, err := readStream(stream, msgBytes)
		if err == io.EOF {
			continue
		} else if err != nil {
			logger.Trace("failed to read from stream", "protocol", stream.Protocol(), "error", err)
			_ = stream.Close()
			return
		}

		// decode message based on message type
		msg, err := decoder(msgBytes[:tot], peer)
		if err != nil {
			logger.Trace("failed to decode message from peer", "protocol", stream.Protocol(), "err", err)
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
			err = handler(stream, msg)
			if err != nil {
				logger.Trace("Failed to handle message from stream", "message", msg, "error", err)
				_ = stream.Close()
				return
			}
		}()
	}
}

func (s *Service) handleLightMsg(stream libp2pnetwork.Stream, msg Message) error {
	defer func() {
		_ = stream.Close()
	}()

	lr, ok := msg.(*LightRequest)
	if !ok {
		return nil
	}

	var resp LightResponse
	var err error
	switch {
	case lr.RmtCallRequest != nil:
		resp.RmtCallResponse, err = remoteCallResp(lr.RmtCallRequest)
	case lr.RmtHeaderRequest != nil:
		resp.RmtHeaderResponse, err = remoteHeaderResp(lr.RmtHeaderRequest)
	case lr.RmtChangesRequest != nil:
		resp.RmtChangeResponse, err = remoteChangeResp(lr.RmtChangesRequest)
	case lr.RmtReadRequest != nil:
		resp.RmtReadResponse, err = remoteReadResp(lr.RmtReadRequest)
	case lr.RmtReadChildRequest != nil:
		resp.RmtReadResponse, err = remoteReadChildResp(lr.RmtReadChildRequest)
	default:
		logger.Warn("ignoring LightRequest without request data")
		return nil
	}

	if err != nil {
		logger.Error("failed to get the response", "err", err)
		return err
	}

	// TODO(arijit): Remove once we implement the internal APIs. Added to increase code coverage.
	logger.Debug("LightResponse", "msg", resp.String())

	err = s.host.writeToStream(stream, &resp)
	if err != nil {
		logger.Warn("failed to send LightResponse message", "peer", stream.Conn().RemotePeer(), "err", err)
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

	s.notificationsMu.RLock()
	np := s.notificationsProtocols[BlockAnnounceMsgType]
	s.notificationsMu.RUnlock()

	for _, p := range s.host.peers() {
		data, has := np.getHandshakeData(p)
		if !has || data.handshake == nil {
			peers = append(peers, common.PeerInfo{
				PeerID: p.String(),
			})

			continue
		}

		peerHandshakeMessage := data.handshake
		peers = append(peers, common.PeerInfo{
			PeerID:     p.String(),
			Roles:      peerHandshakeMessage.(*BlockAnnounceHandshake).Roles,
			BestHash:   peerHandshakeMessage.(*BlockAnnounceHandshake).BestBlockHash,
			BestNumber: uint64(peerHandshakeMessage.(*BlockAnnounceHandshake).BestBlockNumber),
		})
	}

	return peers
}

// NodeRoles Returns the roles the node is running as.
func (s *Service) NodeRoles() byte {
	return s.cfg.Roles
}
