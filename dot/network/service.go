// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	gssmrmetrics "github.com/ChainSafe/gossamer/dot/metrics"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/services"
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
	logger                    = log.NewFromGlobal(log.AddContext("pkg", "network"))
	maxReads                  = 256
)

type (
	// messageDecoder is passed on readStream to decode the data from the stream into a message.
	// since messages are decoded based on context, this is different for every sub-protocol.
	messageDecoder = func([]byte, peer.ID, bool) (Message, error)
	// messageHandler is passed on readStream to handle the resulting message.
	// It should return an error only if the stream is to be closed
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
	closeCh           chan struct{}

	blockResponseBuf   []byte
	blockResponseBufMu sync.Mutex
}

// NewService creates a new network service from the configuration and message channels
func NewService(cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	logger.Patch(log.SetLevel(cfg.LogLvl))
	cfg.logger = logger

	// build configuration
	err := cfg.build()
	if err != nil {
		cancel()
		return nil, err
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

	if cfg.batchSize == 0 {
		cfg.batchSize = defaultTxnBatchSize
	}
	// create a new host instance
	host, err := newHost(ctx, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	// pre-allocate pool of buffers used to read from streams.
	// initially allocate as many buffers as likely necessary which is the number of inbound streams we will have,
	// which should equal the average number of peers times the number of notifications protocols, which is currently 3.
	preAllocateInPool := cfg.MinPeers * 3
	poolSize := cfg.MaxPeers * 3
	if cfg.noPreAllocate { // testing
		preAllocateInPool = 0
		poolSize = cfg.MinPeers * 3
	}
	bufPool := newSizedBufferPool(preAllocateInPool, poolSize)

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
		closeCh:                make(chan struct{}),
		bufPool:                bufPool,
		streamManager:          newStreamManager(ctx),
		blockResponseBuf:       make([]byte, maxBlockResponseSize),
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
		logger.Warnf("failed to register notifications protocol with block announce id %s: %s",
			blockAnnounceID, err)
	}

	txnBatch := make(chan *BatchMessage, s.cfg.batchSize)
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
		logger.Warnf("failed to register notifications protocol with transaction id %s: %s", transactionsID, err)
	}

	// since this opens block announce streams, it should happen after the protocol is registered
	// NOTE: this only handles *incoming* connections
	s.host.h.Network().SetConnHandler(s.handleConn)

	// this handles all new connections (incoming and outgoing)
	// it creates a per-protocol mutex for sending outbound handshakes to the peer
	s.host.cm.connectHandler = func(peerID peer.ID) {
		for _, prtl := range s.notificationsProtocols {
			prtl.outboundHandshakeMutexes.Store(peerID, new(sync.Mutex))
		}
	}

	// when a peer gets disconnected, we should clear all handshake data we have for it.
	s.host.cm.disconnectHandler = func(peerID peer.ID) {
		for _, prtl := range s.notificationsProtocols {
			prtl.outboundHandshakeMutexes.Delete(peerID)
			prtl.inboundHandshakeData.Delete(peerID)
			prtl.outboundHandshakeData.Delete(peerID)
		}
	}

	// log listening addresses to console
	for _, addr := range s.host.multiaddrs() {
		logger.Infof("Started listening on %s", addr)
	}

	s.startPeerSetHandler()

	if !s.noMDNS {
		s.mdns.start()
	}

	if !s.noDiscover {
		go func() {
			err = s.host.discovery.start()
			if err != nil {
				logger.Errorf("failed to begin DHT discovery: %s", err)
			}
		}()
	}

	time.Sleep(time.Millisecond * 500)

	logger.Info("started network service with supported protocols " + strings.Join(s.host.protocols(), ", "))

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
		syncedBlocks := metrics.GetOrRegisterGauge(
			"service/blocks/sync",
			metrics.DefaultRegistry)
		numInboundBlockAnnounceStreams := metrics.GetOrRegisterGauge(
			"network/streams/block_announce/inbound",
			metrics.DefaultRegistry)
		numOutboundBlockAnnounceStreams := metrics.GetOrRegisterGauge(
			"network/streams/block_announce/outbound",
			metrics.DefaultRegistry)
		numInboundGrandpaStreams := metrics.GetOrRegisterGauge("network/streams/grandpa/inbound", metrics.DefaultRegistry)
		numOutboundGrandpaStreams := metrics.GetOrRegisterGauge("network/streams/grandpa/outbound", metrics.DefaultRegistry)
		totalInboundStreams := metrics.GetOrRegisterGauge("network/streams/total/inbound", metrics.DefaultRegistry)
		totalOutboundStreams := metrics.GetOrRegisterGauge("network/streams/total/outbound", metrics.DefaultRegistry)

		peerCount.Update(int64(s.host.peerCount()))
		totalConn.Update(int64(len(s.host.h.Network().Conns())))
		networkLatency.Update(int64(s.host.h.Peerstore().LatencyEWMA(s.host.id())))

		numInboundBlockAnnounceStreams.Update(s.getNumStreams(BlockAnnounceMsgType, true))
		numOutboundBlockAnnounceStreams.Update(s.getNumStreams(BlockAnnounceMsgType, false))
		numInboundGrandpaStreams.Update(s.getNumStreams(ConsensusMsgType, true))
		numOutboundGrandpaStreams.Update(s.getNumStreams(ConsensusMsgType, false))
		totalInboundStreams.Update(s.getTotalStreams(true))
		totalOutboundStreams.Update(s.getTotalStreams(false))

		num, err := s.blockState.BestBlockNumber()
		if err != nil {
			syncedBlocks.Update(0)
		} else {
			syncedBlocks.Update(num.Int64())
		}

		time.Sleep(gssmrmetrics.RefreshInterval)
	}
}

func (s *Service) getTotalStreams(inbound bool) (count int64) {
	for _, conn := range s.host.h.Network().Conns() {
		for _, stream := range conn.GetStreams() {
			streamIsInbound := isInbound(stream)
			if (streamIsInbound && inbound) || (!streamIsInbound && !inbound) {
				count++
			}
		}
	}
	return count
}

func (s *Service) getNumStreams(protocolID byte, inbound bool) (count int64) {
	np, has := s.notificationsProtocols[protocolID]
	if !has {
		return 0
	}

	var hsData *sync.Map
	if inbound {
		hsData = np.inboundHandshakeData
	} else {
		hsData = np.outboundHandshakeData
	}

	hsData.Range(func(_, data interface{}) bool {
		if data == nil {
			return true
		}

		if data.(*handshakeData).stream != nil {
			count++
		}

		return true
	})

	return count
}

func (s *Service) logPeerCount() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debugf("peer count %d, min=%d and max=%d", s.host.peerCount(), s.cfg.MinPeers, s.cfg.MaxPeers)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) publishNetworkTelemetry(done <-chan struct{}) {
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
				logger.Debugf("problem sending system.interval telemetry message: %s", err)
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

		finalised, err := s.blockState.GetHighestFinalisedHeader()
		if err != nil {
			continue
		}
		finalizedHash := finalised.Hash()

		err = telemetry.GetInstance().SendMessage(telemetry.NewBlockIntervalTM(
			&bestHash,
			best.Number,
			&finalizedHash,
			finalised.Number,
			big.NewInt(int64(s.transactionHandler.TransactionsCount())),
			big.NewInt(0), // TODO: (ed) determine where to get used_state_cache_size (#1501)
		))
		if err != nil {
			logger.Debugf("problem sending system.interval telemetry message: %s", err)
		}
		time.Sleep(s.telemetryInterval)
	}
}

func (s *Service) handleConn(conn libp2pnetwork.Conn) {
	// TODO: currently we only have one set so setID is 0, change this once we have more set in peerSet.
	s.host.cm.peerSetHandler.Incoming(0, conn.RemotePeer())
}

// Stop closes running instances of the host and network services as well as
// the message channel from the network service to the core service (services that
// are dependent on the host instance should be closed first)
func (s *Service) Stop() error {
	s.cancel()

	// close mDNS discovery service
	err := s.mdns.close()
	if err != nil {
		logger.Errorf("Failed to close mDNS discovery service: %s", err)
	}

	// close host and host services
	err = s.host.close()
	if err != nil {
		logger.Errorf("Failed to close host: %s", err)
	}

	s.host.cm.peerSetHandler.Stop()

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

	np := newNotificationsProtocol(protocolID, handshakeGetter, handshakeDecoder, handshakeValidator)
	s.notificationsProtocols[messageID] = np
	decoder := createDecoder(np, handshakeDecoder, messageDecoder)
	handlerWithValidate := s.createNotificationsMessageHandler(np, messageHandler, batchHandler)

	s.host.registerStreamHandler(protocolID, func(stream libp2pnetwork.Stream) {
		logger.Tracef("received stream using sub-protocol %s", protocolID)
		s.readStream(stream, decoder, handlerWithValidate)
	})

	logger.Infof("registered notifications sub-protocol %s", protocolID)
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

	logger.Debugf("gossiping from host %s message of type %d: %s",
		s.host.id(), msg.Type(), msg)

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

	logger.Errorf("message type %d not supported by any notifications protocol", msg.Type())
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

// CollectGauge will be used to collect countable metrics from network service
func (s *Service) CollectGauge() map[string]int64 {
	var isSynced int64
	if !s.syncer.IsSynced() {
		isSynced = 1
	}

	return map[string]int64{
		gssmrIsMajorSyncMetric: isSynced,
	}
}

// HighestBlock returns the highest known block number
func (*Service) HighestBlock() int64 {
	// TODO: refactor this to get the data from the sync service (#1857)
	return 0
}

// StartingBlock return the starting block number that's currently being synced
func (*Service) StartingBlock() int64 {
	// TODO: refactor this to get the data from the sync service (#1857)
	return 0
}

// IsSynced returns whether we are synced (no longer in bootstrap mode) or not
func (s *Service) IsSynced() bool {
	return s.syncer.IsSynced()
}

// ReportPeer reports ReputationChange according to the peer behaviour.
func (s *Service) ReportPeer(change peerset.ReputationChange, p peer.ID) {
	s.host.cm.peerSetHandler.ReportPeer(change, p)
}

func (s *Service) startPeerSetHandler() {
	s.host.cm.peerSetHandler.Start()
	// wait for peerSetHandler to start.
	if !s.noBootstrap {
		s.host.bootstrap()
	}

	go s.startProcessingMsg()
}

func (s *Service) processMessage(msg peerset.Message) {
	peerID := msg.PeerID
	if peerID == "" {
		logger.Errorf("found empty peer id in peerset message")
		return
	}
	switch msg.Status {
	case peerset.Connect:
		addrInfo := s.host.h.Peerstore().PeerInfo(peerID)
		if len(addrInfo.Addrs) == 0 {
			var err error
			addrInfo, err = s.host.discovery.findPeer(peerID)
			if err != nil {
				logger.Debugf("failed to find peer id %s: %s", peerID, err)
				return
			}
		}

		err := s.host.connect(addrInfo)
		if err != nil {
			logger.Debugf("failed to open connection for peer %s: %s", peerID, err)
			return
		}
		logger.Debugf("connection successful with peer %s", peerID)
	case peerset.Drop, peerset.Reject:
		err := s.host.closePeer(peerID)
		if err != nil {
			logger.Debugf("failed to close connection with peer %s: %s", peerID, err)
			return
		}
		logger.Debugf("connection dropped successfully for peer %s", peerID)
	}
}

func (s *Service) startProcessingMsg() {
	msgCh := s.host.cm.peerSetHandler.Messages()
	for {
		select {
		case <-s.ctx.Done():
			return
		case msg := <-msgCh:
			s.processMessage(msg)
		}
	}
}
