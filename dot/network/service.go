// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"
	"time"

	gssmrmetrics "github.com/ChainSafe/gossamer/dot/metrics"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/services"
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
	logger                    = log.NewFromGlobal(log.AddContext("pkg", "network"))
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
	closeCh           chan struct{}

	blockResponseBuf   []byte
	blockResponseBufMu sync.Mutex

	batchSize int
}

// NewService creates a new network service from the configuration and message channels
func NewService(cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background()) //nolint

	logger.Patch(log.SetLevel(cfg.LogLvl))
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
		closeCh:                make(chan struct{}),
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
		logger.Warnf("failed to register notifications protocol with block announce id %s: %s",
			blockAnnounceID, err)
	}

	txnBatch := make(chan *BatchMessage, s.batchSize)
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
	s.host.h.Network().SetConnHandler(s.handleConn)

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

		time.Sleep(gssmrmetrics.RefreshInterval)
	}
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

			err = telemetry.GetInstance().SendMessage(telemetry.NewNetworkStateTM(s.host.h, s.Peers()))
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
			logger.Tracef(
				"Cleaning up inbound handshake data for peer %s and protocol %s",
				peerID, protocolID)
			np.inboundHandshakeData.Delete(peerID)
		}

		if _, ok := np.getOutboundHandshakeData(peerID); ok {
			logger.Tracef(
				"Cleaning up outbound handshake data for peer %s and protocol %s",
				peerID, protocolID)
			np.outboundHandshakeData.Delete(peerID)
		}
	})

	info := s.notificationsProtocols[messageID]

	decoder := createDecoder(info, handshakeDecoder, messageDecoder)
	handlerWithValidate := s.createNotificationsMessageHandler(info, messageHandler, batchHandler)

	s.host.registerStreamHandler(protocolID, func(stream libp2pnetwork.Stream) {
		logger.Tracef("received stream using sub-protocol %s", protocolID)
		conn := stream.Conn()
		if conn == nil {
			logger.Error("Failed to get connection from stream")
			return
		}

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
			logger.Tracef(
				"failed to read from stream id %s of peer %s using protocol %s: %s",
				stream.ID(), stream.Conn().RemotePeer(), stream.Protocol(), err)
			_ = stream.Close()
			return
		}

		s.streamManager.logMessageReceived(stream.ID())

		// decode message based on message type
		msg, err := decoder(msgBytes[:tot], peer, isInbound(stream))
		if err != nil {
			logger.Tracef("failed to decode message from stream id %s using protocol %s: %s",
				stream.ID(), stream.Protocol(), err)
			continue
		}

		logger.Tracef(
			"host %s received message from peer %s: %s",
			s.host.id(), peer, msg.String())

		err = handler(stream, msg)
		if err != nil {
			logger.Tracef("failed to handle message %s from stream id %s: %s", msg, stream.ID(), err)
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

	resp := NewLightResponse()
	var err error
	switch {
	case lr.RemoteCallRequest != nil:
		resp.RemoteCallResponse, err = remoteCallResp(lr.RemoteCallRequest)
	case lr.RemoteHeaderRequest != nil:
		resp.RemoteHeaderResponse, err = remoteHeaderResp(lr.RemoteHeaderRequest)
	case lr.RemoteChangesRequest != nil:
		resp.RemoteChangesResponse, err = remoteChangeResp(lr.RemoteChangesRequest)
	case lr.RemoteReadRequest != nil:
		resp.RemoteReadResponse, err = remoteReadResp(lr.RemoteReadRequest)
	case lr.RemoteReadChildRequest != nil:
		resp.RemoteReadResponse, err = remoteReadChildResp(lr.RemoteReadChildRequest)
	default:
		logger.Warn("ignoring LightRequest without request data")
		return nil
	}

	if err != nil {
		logger.Errorf("failed to get the response: %s", err)
		return err
	}

	// TODO(arijit): Remove once we implement the internal APIs. Added to increase code coverage. (#1856)
	logger.Debugf("LightResponse message: %s", resp)

	err = s.host.writeToStream(stream, resp)
	if err != nil {
		logger.Warnf("failed to send LightResponse message to peer %s: %s", stream.Conn().RemotePeer(), err)
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
		case m := <-msgCh:
			msg, ok := m.(peerset.Message)
			if !ok {
				logger.Error(fmt.Sprintf("failed to get message from peerSet: type is %T instead of peerset.Message", m))
				continue
			}
			s.processMessage(msg)
		}
	}
}
