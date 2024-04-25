// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/mdns"
	"github.com/ChainSafe/gossamer/internal/metrics"
	"github.com/ChainSafe/gossamer/lib/common"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// NetworkStateTimeout is the set time interval that we update network state
	NetworkStateTimeout = time.Minute

	// the following are sub-protocols used by the node
	SyncID          = "/sync/2"
	lightID         = "/light/2"
	blockAnnounceID = "/block-announces/1"
	transactionsID  = "/transactions/1"

	maxMessageSize = 1024 * 64 // 64kb for now

	defaultBufferSize = 128
)

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "network"))

	peerCountGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_node",
		Name:      "peer_count_total",
		Help:      "total peer count",
	})
	connectionsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_node",
		Name:      "connections_total",
		Help:      "total number of connections",
	})
	nodeLatencyGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_node",
		Name:      "latency_ms",
		Help:      "average node latency in milliseconds",
	})
	inboundBlockAnnounceStreamsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_streams_block_announce",
		Name:      "inbound_total",
		Help:      "total number of inbound block announce streams",
	})
	outboundBlockAnnounceStreamsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_streams_block_announce",
		Name:      "outbound_total",
		Help:      "total number of outbound block announce streams",
	})
	inboundGrandpaStreamsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_streams_grandpa",
		Name:      "inbound_total",
		Help:      "total number of inbound grandpa streams",
	})
	outboundGrandpaStreamsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_streams_grandpa",
		Name:      "outbound_total",
		Help:      "total number of outbound grandpa streams",
	})
	inboundStreamsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_streams",
		Name:      "inbound_total",
		Help:      "total number of inbound streams",
	})
	outboundStreamsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_streams",
		Name:      "outbound_total",
		Help:      "total number of outbound streams",
	})
	processStartTimeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "substrate", // Note: this is using substrate namespace because that is what zombienet uses
		//  to confirm nodes have started TODO: consider other ways to handle this, see issue #3205
		Name: "process_start_time_seconds",
		Help: "gossamer process start seconds unix timestamp, " +
			"using substrate namespace so zombienet detects node start",
	})
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
	mdns          MDNS
	gossip        *gossip
	bufPool       *sync.Pool
	streamManager *streamManager

	notificationsProtocols map[MessageType]*notificationsProtocol // map of sub-protocol msg ID to protocol info
	notificationsMu        sync.RWMutex

	lightRequest   map[peer.ID]struct{} // set if we have sent a light request message to the given peer
	lightRequestMu sync.RWMutex

	// Service interfaces
	blockState         BlockState
	syncer             Syncer
	transactionHandler TransactionHandler

	// networkEventInfoChannels stores channels used to receive network event information,
	// such as connected and disconnected peers
	networkEventInfoChannels map[chan *NetworkEventInfo]struct{}

	// Configuration options
	noBootstrap bool
	noDiscover  bool
	noMDNS      bool
	noGossip    bool // internal option

	Metrics metrics.IntervalConfig

	// telemetry
	telemetryInterval time.Duration
	closeCh           chan struct{}

	telemetry Telemetry
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
		return nil, fmt.Errorf("failed to create host: %w", err)
	}

	bufPool := &sync.Pool{
		New: func() interface{} {
			b := make([]byte, maxMessageSize)
			return &b
		},
	}

	serviceTag := string(host.protocolID)
	notifee := mdns.NewNotifeeTracker(host.p2pHost.Peerstore(), host.cm.peerSetHandler)
	mdnsLogger := log.NewFromGlobal(log.AddContext("module", "mdns"))
	mdnsLogger.Debugf(
		"Creating mDNS discovery service with host %s and protocol %s...",
		host.id(), host.protocolID)
	mdnsService := mdns.NewService(host.p2pHost, serviceTag, mdnsLogger, notifee)

	network := &Service{
		ctx:                      ctx,
		cancel:                   cancel,
		cfg:                      cfg,
		host:                     host,
		mdns:                     mdnsService,
		gossip:                   newGossip(),
		blockState:               cfg.BlockState,
		transactionHandler:       cfg.TransactionHandler,
		noBootstrap:              cfg.NoBootstrap,
		noMDNS:                   cfg.NoMDNS,
		syncer:                   cfg.Syncer,
		notificationsProtocols:   make(map[MessageType]*notificationsProtocol),
		lightRequest:             make(map[peer.ID]struct{}),
		networkEventInfoChannels: make(map[chan *NetworkEventInfo]struct{}),
		telemetryInterval:        cfg.telemetryInterval,
		closeCh:                  make(chan struct{}),
		bufPool:                  bufPool,
		streamManager:            newStreamManager(ctx),
		telemetry:                cfg.Telemetry,
		Metrics:                  cfg.Metrics,
	}

	return network, nil
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

	s.host.registerStreamHandler(s.host.protocolID+SyncID, s.handleSyncStream)
	s.host.registerStreamHandler(s.host.protocolID+lightID, s.handleLightStream)

	// register block announce protocol
	err := s.RegisterNotificationsProtocol(
		s.host.protocolID+blockAnnounceID,
		blockAnnounceMsgType,
		s.getBlockAnnounceHandshake,
		decodeBlockAnnounceHandshake,
		s.validateBlockAnnounceHandshake,
		decodeBlockAnnounceMessage,
		s.handleBlockAnnounceMessage,
		nil,
		maxBlockAnnounceNotificationSize,
	)
	if err != nil {
		logger.Warnf("failed to register notifications protocol with block announce id %s: %s",
			blockAnnounceID, err)
	}

	txnBatch := make(chan *batchMessage, s.cfg.batchSize)
	txnBatchHandler := s.createBatchMessageHandler(txnBatch)

	// register transactions protocol
	err = s.RegisterNotificationsProtocol(
		s.host.protocolID+transactionsID,
		transactionMsgType,
		s.getTransactionHandshake,
		decodeTransactionHandshake,
		validateTransactionHandshake,
		decodeTransactionMessage,
		s.handleTransactionMessage,
		txnBatchHandler,
		maxTransactionsNotificationSize,
	)
	if err != nil {
		logger.Warnf("failed to register notifications protocol with transaction id %s: %s", transactionsID, err)
	}

	// this handles all new connections (incoming and outgoing)
	// it creates a per-protocol mutex for sending outbound handshakes to the peer
	s.host.cm.connectHandler = func(peerID peer.ID) {
		for _, prtl := range s.notificationsProtocols {
			prtl.peersData.setMutex(peerID)
		}
		// TODO: currently we only have one set so setID is 0, change this once we have more set in peerSet
		const setID = 0
		s.host.cm.peerSetHandler.Incoming(setID, peerID)
	}

	// when a peer gets disconnected, we should clear all handshake data we have for it.
	s.host.cm.disconnectHandler = func(peerID peer.ID) {
		for _, prtl := range s.notificationsProtocols {
			prtl.peersData.deleteMutex(peerID)
			prtl.peersData.deleteInboundHandshakeData(peerID)
			prtl.peersData.deleteOutboundHandshakeData(peerID)
		}
	}

	// log listening addresses to console
	for _, addr := range s.host.multiaddrs() {
		logger.Infof("Started listening on %s", addr)
	}

	s.startPeerSetHandler()

	if !s.noMDNS {
		err = s.mdns.Start()
		if err != nil {
			return fmt.Errorf("starting mDNS service: %w", err)
		}
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

	if s.Metrics.Publish {
		processStartTimeGauge.Set(float64(time.Now().Unix()))
		go s.updateMetrics()
	}

	go s.logPeerCount()
	go s.publishNetworkTelemetry(s.closeCh)
	go s.sentBlockIntervalTelemetry()
	s.streamManager.start()

	return nil
}

func (s *Service) updateMetrics() {
	ticker := time.NewTicker(s.Metrics.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			peerCountGauge.Set(float64(s.host.peerCount()))
			connectionsGauge.Set(float64(len(s.host.p2pHost.Network().Conns())))
			nodeLatencyGauge.Set(float64(
				s.host.p2pHost.Peerstore().LatencyEWMA(s.host.id()).Milliseconds()))
			inboundBlockAnnounceStreamsGauge.Set(float64(
				s.getNumStreams(blockAnnounceMsgType, true)))
			outboundBlockAnnounceStreamsGauge.Set(float64(
				s.getNumStreams(blockAnnounceMsgType, false)))
			inboundGrandpaStreamsGauge.Set(float64(s.getNumStreams(ConsensusMsgType, true)))
			outboundGrandpaStreamsGauge.Set(float64(s.getNumStreams(ConsensusMsgType, false)))
			inboundStreamsGauge.Set(float64(s.getTotalStreams(true)))
			outboundStreamsGauge.Set(float64(s.getTotalStreams(false)))
		}
	}
}

func (s *Service) getTotalStreams(inbound bool) (count int64) {
	for _, conn := range s.host.p2pHost.Network().Conns() {
		for _, stream := range conn.GetStreams() {
			streamIsInbound := isInbound(stream)
			if (streamIsInbound && inbound) || (!streamIsInbound && !inbound) {
				count++
			}
		}
	}
	return count
}

func (s *Service) getNumStreams(protocolID MessageType, inbound bool) (count int64) {
	np, has := s.notificationsProtocols[protocolID]
	if !has {
		return 0
	}

	if inbound {
		return np.peersData.countInboundStreams()
	}
	return np.peersData.countOutboundStreams()
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

	for {
		select {
		case <-done:
			return

		case <-ticker.C:
			o := s.host.bwc.GetBandwidthTotals()
			s.telemetry.SendMessage(telemetry.NewBandwidth(o.RateIn, o.RateOut, s.host.peerCount()))
		}
	}
}

func (s *Service) sentBlockIntervalTelemetry() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

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

		s.telemetry.SendMessage(telemetry.NewBlockInterval(
			&bestHash,
			best.Number,
			&finalizedHash,
			finalised.Number,
			big.NewInt(int64(s.transactionHandler.TransactionsCount())),
			big.NewInt(0), // TODO: (ed) determine where to get used_state_cache_size (#1501)
		))

		time.Sleep(s.telemetryInterval)
	}
}

// Stop closes running instances of the host and network services as well as
// the message channel from the network service to the core service (services that
// are dependent on the host instance should be closed first)
func (s *Service) Stop() error {
	s.cancel()

	// close mDNS discovery service
	err := s.mdns.Stop()
	if err != nil {
		logger.Errorf("Failed to close mDNS discovery service: %s", err)
	}

	// close host and host services
	err = s.host.close()
	if err != nil {
		logger.Errorf("Failed to close host: %s", err)
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
	messageID MessageType,
	handshakeGetter HandshakeGetter,
	handshakeDecoder HandshakeDecoder,
	handshakeValidator HandshakeValidator,
	messageDecoder MessageDecoder,
	messageHandler NotificationsMessageHandler,
	batchHandler NotificationsMessageBatchHandler,
	maxSize uint64,
) error {
	s.notificationsMu.Lock()
	defer s.notificationsMu.Unlock()

	if _, has := s.notificationsProtocols[messageID]; has {
		return errors.New("notifications protocol with message type already exists")
	}

	np := newNotificationsProtocol(protocolID, handshakeGetter, handshakeDecoder, handshakeValidator, maxSize)
	s.notificationsProtocols[messageID] = np
	decoder := createDecoder(np, handshakeDecoder, messageDecoder)
	handlerWithValidate := s.createNotificationsMessageHandler(np, messageHandler, batchHandler)

	s.host.registerStreamHandler(protocolID, func(stream libp2pnetwork.Stream) {
		logger.Tracef("received stream using sub-protocol %s", protocolID)
		s.readStream(stream, decoder, handlerWithValidate, maxSize)
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

		prtl.peersData.setMutex(to)

		s.sendData(to, hs, prtl, msg)
		return nil
	}

	return errors.New("message not supported by any notifications protocol")
}

func (s *Service) GetRequestResponseProtocol(subprotocol string, requestTimeout time.Duration,
	maxResponseSize uint64) *RequestResponseProtocol {

	protocolID := s.host.protocolID + protocol.ID(subprotocol)
	return &RequestResponseProtocol{
		ctx:             s.ctx,
		host:            s.host,
		requestTimeout:  requestTimeout,
		maxResponseSize: maxResponseSize,
		protocolID:      protocolID,
		responseBuf:     make([]byte, maxResponseSize),
		responseBufMu:   sync.Mutex{},
	}
}

func (s *Service) GetNetworkEventsChannel() chan *NetworkEventInfo {
	ch := make(chan *NetworkEventInfo, defaultBufferSize)
	s.networkEventInfoChannels[ch] = struct{}{}
	return ch
}

func (s *Service) FreeNetworkEventsChannel(ch chan *NetworkEventInfo) {
	delete(s.networkEventInfoChannels, ch)
}

type NetworkEvent bool

const (
	Connected    NetworkEvent = true
	Disconnected NetworkEvent = false
)

type NetworkEventInfo struct {
	PeerID         peer.ID
	Event          NetworkEvent
	Role           common.NetworkRole
	MayBeAuthority *types.AuthorityID
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

// AllConnectedPeersIDs returns all the connected to the node instance
func (s *Service) AllConnectedPeersIDs() []peer.ID {
	return s.host.p2pHost.Network().Peers()
}

// Peers returns information about connected peers needed for the rpc server
func (s *Service) Peers() []common.PeerInfo {
	var peers []common.PeerInfo

	s.notificationsMu.RLock()
	np := s.notificationsProtocols[blockAnnounceMsgType]
	s.notificationsMu.RUnlock()

	for _, p := range s.host.peers() {
		data := np.peersData.getInboundHandshakeData(p)
		if data == nil || data.handshake == nil {
			peers = append(peers, common.PeerInfo{
				PeerID: p.String(),
			})

			continue
		}

		peerHandshakeMessage := data.handshake
		peers = append(peers, common.PeerInfo{
			PeerID:     p.String(),
			Role:       peerHandshakeMessage.(*BlockAnnounceHandshake).Roles,
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
func (s *Service) NodeRoles() common.NetworkRole {
	return s.cfg.Roles
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

func (s *Service) DisconnectPeer(setID int, p peer.ID) {
	s.host.cm.peerSetHandler.DisconnectPeer(setID, p)
}

func (s *Service) startPeerSetHandler() {
	s.host.cm.peerSetHandler.Start(s.ctx)
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
		addrInfo := s.host.p2pHost.Peerstore().PeerInfo(peerID)
		if len(addrInfo.Addrs) == 0 {
			var err error
			addrInfo, err = s.host.discovery.findPeer(peerID)
			if err != nil {
				logger.Warnf("failed to find peer id %s: %s", peerID, err)
				return
			}
		}

		err := s.host.connect(addrInfo)
		if err != nil {
			logger.Warnf("failed to open connection for peer %s: %s", peerID, err)
			return
		}
		logger.Debugf("connection successful with peer %s", peerID)

		for ch := range s.networkEventInfoChannels {
			ch <- &NetworkEventInfo{
				PeerID: peerID,
				Event:  Connected,
			}
		}

	case peerset.Drop, peerset.Reject:
		err := s.host.closePeer(peerID)
		if err != nil {
			logger.Warnf("failed to close connection with peer %s: %s", peerID, err)
			return
		}
		logger.Debugf("connection dropped successfully for peer %s", peerID)

		for ch := range s.networkEventInfoChannels {
			ch <- &NetworkEventInfo{
				PeerID: peerID,
				Event:  Disconnected,
			}
		}

	}
}

func (s *Service) startProcessingMsg() {
	msgCh := s.host.cm.peerSetHandler.Messages()
	for {
		select {
		case <-s.ctx.Done():
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}

			s.processMessage(msg)
		}
	}
}

func (s *Service) BlockAnnounceHandshake(header *types.Header) error {
	peers := s.host.peers()
	if len(peers) == 0 {
		return ErrNoPeersConnected
	}

	protocol, ok := s.notificationsProtocols[blockAnnounceMsgType]
	if !ok {
		panic("block announce message type not found")
	}

	handshake, err := protocol.getHandshake()
	if err != nil {
		return fmt.Errorf("getting handshake: %w", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(peers))
	for _, p := range peers {
		protocol.peersData.setMutex(p)

		go func(p peer.ID) {
			defer wg.Done()
			stream, err := s.sendHandshake(p, handshake, protocol)
			if err != nil {
				logger.Tracef("sending block announce handshake: %s", err)
				return
			}

			response := protocol.peersData.getOutboundHandshakeData(p)
			if response.received && response.validated {
				closeOutboundStream(protocol, p, stream)
			}
		}(p)
	}

	wg.Wait()
	return nil
}
