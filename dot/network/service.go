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
	"io"
	"math/big"
	"os"
	"sync"
	"time"

	gssmrmetrics "github.com/ChainSafe/gossamer/dot/metrics"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/services"
	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/metrics"
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

	maxMessageSize = 1024 * 63 // 63kb for now

	gssmrIsMajorSyncMetric = "gossamer/network/is_major_syncing"
)

var (
	_        services.Service = &Service{}
	logger                    = log.New("pkg", "network")
	maxReads                  = 256
)

type (
	// messageDecoder is passed on readStream to decode the data from the stream into a message.
	// since messages are decoded based on context, this is different for every sub-protocol.
	messageDecoder = func([]byte, peer.ID, bool) (Message, error)
	// messageHandler is passed on readStream to handle the resulting message. it should return an error only if the stream is to be closed
	messageHandler = func(stream libp2pnetwork.Stream, msg Message) error
)

// Service describes a network service
type Service struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg           *Config
	host          *host
	mdns          *mdns
	gossip        *gossip
	bufPool       *sizedBufferPool
	streamManager *streamManager

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

	// telemetry
	telemetryInterval time.Duration
	closeCh           chan interface{}

	blockResponseBuf   []byte
	blockResponseBufMu sync.Mutex

	batchSize int
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

	if cfg.DiscoveryInterval > 0 {
		connectToPeersTimeout = cfg.DiscoveryInterval
	}

	// create a new host instance
	host, err := newHost(ctx, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	// pre-allocate pool of buffers used to read from streams.
	// initially allocate as many buffers as liekly necessary which is the number inbound streams we will have,
	// which should equal average number of peers times the number of notifications protocols, which is currently 3.
	var bufPool *sizedBufferPool
	if cfg.noPreAllocate {
		bufPool = &sizedBufferPool{
			c: make(chan *[maxMessageSize]byte, cfg.MinPeers*3),
		}
	} else {
		bufPool = newSizedBufferPool(cfg.MinPeers*3, cfg.MaxPeers*3)
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
		telemetryInterval:      cfg.telemetryInterval,
		closeCh:                make(chan interface{}),
		bufPool:                bufPool,
		streamManager:          newStreamManager(ctx),
		blockResponseBuf:       make([]byte, maxBlockResponseSize),
		batchSize:              100,
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

	s.host.registerStreamHandler(s.host.protocolID+syncID, s.handleSyncStream)
	s.host.registerStreamHandler(s.host.protocolID+lightID, s.handleLightStream)

	// register block announce protocol
	err := s.RegisterNotificationsProtocol(
		s.host.protocolID+blockAnnounceID,
		BlockAnnounceMsgType,
		s.getBlockAnnounceHandshake,
		decodeBlockAnnounceHandshake,
		s.validateBlockAnnounceHandshake,
		decodeBlockAnnounceMessage,
		s.handleBlockAnnounceMessage,
		nil,
	)
	if err != nil {
		logger.Warn("failed to register notifications protocol", "sub-protocol", blockAnnounceID, "error", err)
	}

	txnBatch := make(chan *batchMessage, s.batchSize)
	txnBatchHandler := s.createBatchMessageHandler(txnBatch)

	// register transactions protocol
	err = s.RegisterNotificationsProtocol(
		s.host.protocolID+transactionsID,
		TransactionMsgType,
		s.getTransactionHandshake,
		decodeTransactionHandshake,
		validateTransactionHandshake,
		decodeTransactionMessage,
		s.handleTransactionMessage,
		txnBatchHandler,
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
			err = s.host.discovery.start()
			if err != nil {
				logger.Error("failed to begin DHT discovery", "error", err)
			}
		}()
	}

	time.Sleep(time.Millisecond * 500)

	logger.Info("started network service", "supported protocols", s.host.protocols())

	if s.cfg.PublishMetrics {
		go s.collectNetworkMetrics()
	}

	go s.logPeerCount()
	go s.publishNetworkTelemetry(s.closeCh)
	go s.sentBlockIntervalTelemetry()
	s.streamManager.start()

	return nil
}

func (s *Service) collectNetworkMetrics() {
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
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debug("peer count", "num", s.host.peerCount(), "min", s.cfg.MinPeers, "max", s.cfg.MaxPeers)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) publishNetworkTelemetry(done chan interface{}) {
	ticker := time.NewTicker(s.telemetryInterval)
	defer ticker.Stop()

main:
	for {
		select {
		case <-done:
			break main

		case <-ticker.C:
			o := s.host.bwc.GetBandwidthTotals()
			err := telemetry.GetInstance().SendMessage(telemetry.NewBandwidthTM(o.RateIn, o.RateOut, s.host.peerCount()))
			if err != nil {
				logger.Debug("problem sending system.interval telemetry message", "error", err)
			}

			err = telemetry.GetInstance().SendMessage(telemetry.NewNetworkStateTM(s.host.h, s.Peers()))
			if err != nil {
				logger.Debug("problem sending system.interval telemetry message", "error", err)
			}
		}
	}
}

func (s *Service) sentBlockIntervalTelemetry() {
	for {
		best, err := s.blockState.BestBlockHeader()
		if err != nil {
			continue
		}
		bestHash := best.Hash()

		finalized, err := s.blockState.GetHighestFinalisedHeader() //nolint
		if err != nil {
			continue
		}
		finalizedHash := finalized.Hash()

		err = telemetry.GetInstance().SendMessage(telemetry.NewBlockIntervalTM(
			&bestHash,
			best.Number,
			&finalizedHash,
			finalized.Number,
			big.NewInt(int64(s.transactionHandler.TransactionsCount())),
			big.NewInt(0), // todo (ed) determine where to get used_state_cache_size
		))
		if err != nil {
			logger.Debug("problem sending system.interval telemetry message", "error", err)
		}
		time.Sleep(s.telemetryInterval)
	}
}

func (*Service) handleConn(conn libp2pnetwork.Conn) {
	// TODO: update this for scoring
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

	// check if closeCh is closed, if not, close it.
mainloop:
	for {
		select {
		case _, hasMore := <-s.closeCh:
			if !hasMore {
				break mainloop
			}
		default:
			close(s.closeCh)
		}
	}

	return nil
}

// RegisterNotificationsProtocol registers a protocol with the network service with the given handler
// messageID is a user-defined message ID for the message passed over this protocol.
func (s *Service) RegisterNotificationsProtocol(
	protocolID protocol.ID,
	messageID byte,
	handshakeGetter HandshakeGetter,
	handshakeDecoder HandshakeDecoder,
	handshakeValidator HandshakeValidator,
	messageDecoder MessageDecoder,
	messageHandler NotificationsMessageHandler,
	batchHandler NotificationsMessageBatchHandler,
) error {
	s.notificationsMu.Lock()
	defer s.notificationsMu.Unlock()

	if _, has := s.notificationsProtocols[messageID]; has {
		return errors.New("notifications protocol with message type already exists")
	}

	np := &notificationsProtocol{
		protocolID:            protocolID,
		getHandshake:          handshakeGetter,
		handshakeValidator:    handshakeValidator,
		handshakeDecoder:      handshakeDecoder,
		inboundHandshakeData:  new(sync.Map),
		outboundHandshakeData: new(sync.Map),
	}
	s.notificationsProtocols[messageID] = np

	connMgr := s.host.h.ConnManager().(*ConnManager)
	connMgr.registerCloseHandler(protocolID, func(peerID peer.ID) {
		if _, ok := np.getInboundHandshakeData(peerID); ok {
			logger.Trace(
				"Cleaning up inbound handshake data",
				"peer", peerID,
				"protocol", protocolID,
			)
			np.inboundHandshakeData.Delete(peerID)
		}

		if _, ok := np.getOutboundHandshakeData(peerID); ok {
			logger.Trace(
				"Cleaning up outbound handshake data",
				"peer", peerID,
				"protocol", protocolID,
			)
			np.outboundHandshakeData.Delete(peerID)
		}
	})

	info := s.notificationsProtocols[messageID]

	decoder := createDecoder(info, handshakeDecoder, messageDecoder)
	handlerWithValidate := s.createNotificationsMessageHandler(info, messageHandler, batchHandler)

	s.host.registerStreamHandler(protocolID, func(stream libp2pnetwork.Stream) {
		logger.Trace("received stream", "sub-protocol", protocolID)
		conn := stream.Conn()
		if conn == nil {
			logger.Error("Failed to get connection from stream")
			return
		}

		s.readStream(stream, decoder, handlerWithValidate)
	})

	logger.Info("registered notifications sub-protocol", "protocol", protocolID)
	return nil
}

// IsStopped returns true if the service is stopped
func (s *Service) IsStopped() bool {
	return s.ctx.Err() != nil
}

// GossipMessage gossips a notifications protocol message to our peers
func (s *Service) GossipMessage(msg NotificationsMessage) {
	if s.host == nil || msg == nil || s.IsStopped() {
		return
	}

	logger.Debug(
		"gossiping message",
		"host", s.host.id(),
		"type", msg.Type(),
		"message", msg,
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

// SendMessage sends a message to the given peer
func (s *Service) SendMessage(to peer.ID, msg NotificationsMessage) error {
	s.notificationsMu.Lock()
	defer s.notificationsMu.Unlock()

	for msgID, prtl := range s.notificationsProtocols {
		if msg.Type() != msgID {
			continue
		}

		hs, err := prtl.getHandshake()
		if err != nil {
			return err
		}

		s.sendData(to, hs, prtl, msg)
		return nil
	}

	return errors.New("message not supported by any notifications protocol")
}

// handleLightStream handles streams with the <protocol-id>/light/2 protocol ID
func (s *Service) handleLightStream(stream libp2pnetwork.Stream) {
	s.readStream(stream, s.decodeLightMessage, s.handleLightMsg)
}

func (s *Service) decodeLightMessage(in []byte, peer peer.ID, _ bool) (Message, error) {
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

func isInbound(stream libp2pnetwork.Stream) bool {
	return stream.Stat().Direction == libp2pnetwork.DirInbound
}

func (s *Service) readStream(stream libp2pnetwork.Stream, decoder messageDecoder, handler messageHandler) {
	s.streamManager.logNewStream(stream)

	peer := stream.Conn().RemotePeer()
	msgBytes := s.bufPool.get()
	defer s.bufPool.put(&msgBytes)

	for {
		tot, err := readStream(stream, msgBytes[:])
		if errors.Is(err, io.EOF) {
			return
		} else if err != nil {
			logger.Trace("failed to read from stream", "id", stream.ID(), "peer", stream.Conn().RemotePeer(), "protocol", stream.Protocol(), "error", err)
			_ = stream.Close()
			return
		}

		s.streamManager.logMessageReceived(stream.ID())

		// decode message based on message type
		msg, err := decoder(msgBytes[:tot], peer, isInbound(stream))
		if err != nil {
			logger.Trace("failed to decode message from peer", "id", stream.ID(), "protocol", stream.Protocol(), "err", err)
			continue
		}

		logger.Trace(
			"received message from peer",
			"host", s.host.id(),
			"peer", peer,
			"msg", msg.String(),
		)

		err = handler(stream, msg)
		if err != nil {
			logger.Trace("failed to handle message from stream", "id", stream.ID(), "message", msg, "error", err)
			_ = stream.Close()
			return
		}

		s.host.bwc.LogRecvMessage(int64(tot))
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
	var peers []common.PeerInfo

	s.notificationsMu.RLock()
	np := s.notificationsProtocols[BlockAnnounceMsgType]
	s.notificationsMu.RUnlock()

	for _, p := range s.host.peers() {
		data, has := np.getInboundHandshakeData(p)
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

// AddReservedPeers insert new peers to the peerstore with PermanentAddrTTL
func (s *Service) AddReservedPeers(addrs ...string) error {
	return s.host.addReservedPeers(addrs...)
}

// RemoveReservedPeers closes all connections with the target peers and remove it from the peerstore
func (s *Service) RemoveReservedPeers(addrs ...string) error {
	return s.host.removeReservedPeers(addrs...)
}

// NodeRoles Returns the roles the node is running as.
func (s *Service) NodeRoles() byte {
	return s.cfg.Roles
}

// CollectGauge will be used to collect coutable metrics from network service
func (s *Service) CollectGauge() map[string]int64 {
	var isSynced int64
	if !s.syncer.IsSynced() {
		isSynced = 1
	} else {
		isSynced = 0
	}

	return map[string]int64{
		gssmrIsMajorSyncMetric: isSynced,
	}
}

// HighestBlock returns the highest known block number
func (*Service) HighestBlock() int64 {
	// TODO: refactor this to get the data from the sync service
	return 0
}

// StartingBlock return the starting block number that's currently being synced
func (*Service) StartingBlock() int64 {
	// TODO: refactor this to get the data from the sync service
	return 0
}

// IsSynced returns whether we are synced (no longer in bootstrap mode) or not
func (s *Service) IsSynced() bool {
	return s.syncer.IsSynced()
}
