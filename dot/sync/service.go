// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	waitPeersDefaultTimeout = 10 * time.Second
	minPeersDefault         = 1
)

var (
	isSyncedGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_syncer",
		Name:      "is_synced",
		Help:      "bool representing whether the node is synced to the head of the chain",
	})

	logger = log.NewFromGlobal(log.AddContext("pkg", "sync"))
)

type BlockOrigin byte

const (
	networkInitialSync BlockOrigin = iota
	networkBroadcast
)

type Network interface {
	AllConnectedPeersIDs() []peer.ID
	ReportPeer(change peerset.ReputationChange, p peer.ID)
	BlockAnnounceHandshake(*types.Header) error
	GetRequestResponseProtocol(subprotocol string, requestTimeout time.Duration,
		maxResponseSize uint64) *network.RequestResponseProtocol
	GossipMessageExcluding(network.NotificationsMessage, peer.ID)
}

type BlockState interface {
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (number uint, err error)
	CompareAndSetBlockData(bd *types.BlockData) error
	GetBlockBody(common.Hash) (*types.Body, error)
	GetHeader(common.Hash) (*types.Header, error)
	HasHeader(hash common.Hash) (bool, error)
	Range(startHash, endHash common.Hash) (hashes []common.Hash, err error)
	RangeInMemory(start, end common.Hash) ([]common.Hash, error)
	GetReceipt(common.Hash) ([]byte, error)
	GetMessageQueue(common.Hash) ([]byte, error)
	GetJustification(common.Hash) ([]byte, error)
	SetFinalisedHash(hash common.Hash, round uint64, setID uint64) error
	SetJustification(hash common.Hash, data []byte) error
	GetHashByNumber(blockNumber uint) (common.Hash, error)
	GetBlockByHash(common.Hash) (*types.Block, error)
	GetRuntime(blockHash common.Hash) (runtime runtime.Instance, err error)
	StoreRuntime(blockHash common.Hash, runtime runtime.Instance)
	GetHighestFinalisedHeader() (*types.Header, error)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo
	GetHeaderByNumber(num uint) (*types.Header, error)
	GetAllBlocksAtNumber(num uint) ([]common.Hash, error)
	IsDescendantOf(parent, child common.Hash) (bool, error)

	IsPaused() bool
	Pause() error
}

type Change struct {
	who peer.ID
	rep peerset.ReputationChange
}

type Strategy interface {
	OnBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) (repChange *Change, err error)
	OnBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error
	NextActions() ([]*SyncTask, error)
	Process(results []*SyncTaskResult) (done bool, repChanges []Change, blocks []peer.ID, err error)
	ShowMetrics()
	IsSynced() bool
}

type SyncService struct {
	mu         sync.Mutex
	wg         sync.WaitGroup
	network    Network
	blockState BlockState

	currentStrategy Strategy
	defaultStrategy Strategy

	workerPool        *syncWorkerPool
	waitPeersDuration time.Duration
	minPeers          int
	slotDuration      time.Duration

	seenBlockSyncRequests *lrucache.LRUCache[common.Hash, uint]

	stopCh chan struct{}
}

func NewSyncService(cfgs ...ServiceConfig) *SyncService {
	svc := &SyncService{
		minPeers:              minPeersDefault,
		waitPeersDuration:     waitPeersDefaultTimeout,
		stopCh:                make(chan struct{}),
		seenBlockSyncRequests: lrucache.NewLRUCache[common.Hash, uint](100),
	}

	for _, cfg := range cfgs {
		cfg(svc)
	}

	return svc
}

func (s *SyncService) waitWorkers() {
	bestBlockHeader, err := s.blockState.BestBlockHeader()
	if err != nil {
		panic(fmt.Sprintf("failed to get highest finalised header: %v", err))
	}

	for {
		total := s.workerPool.totalWorkers()
		if total >= s.minPeers {
			return
		}

		err = s.network.BlockAnnounceHandshake(bestBlockHeader)
		if err != nil && !errors.Is(err, network.ErrNoPeersConnected) {
			logger.Criticalf("waiting workers: %s", err.Error())
			break
		}

		waitPeersTimer := time.NewTimer(s.waitPeersDuration)
		select {
		case <-waitPeersTimer.C:
			waitPeersTimer.Reset(s.waitPeersDuration)

		case <-s.stopCh:
			return
		}
	}
}

func (s *SyncService) Start() error {
	s.wg.Add(1)
	go s.runSyncEngine()
	return nil
}

func (s *SyncService) Stop() error {
	close(s.stopCh)
	s.wg.Wait()
	return nil
}

func (s *SyncService) HandleBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	logger.Infof("receiving a block announce handshake from %s", from.String())
	if err := s.workerPool.fromBlockAnnounceHandshake(from); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentStrategy.OnBlockAnnounceHandshake(from, msg)
}

func (s *SyncService) HandleBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repChange, err := s.currentStrategy.OnBlockAnnounce(from, msg)

	if repChange != nil {
		s.network.ReportPeer(repChange.rep, repChange.who)
	}

	if err != nil {
		return fmt.Errorf("while handling block announce: %w", err)
	}
	return nil
}

func (s *SyncService) OnConnectionClosed(who peer.ID) {
	logger.Tracef("removing peer worker: %s", who.String())
	s.workerPool.removeWorker(who)
}

func (s *SyncService) IsSynced() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentStrategy.IsSynced()
}

func (s *SyncService) HighestBlock() uint {
	highestBlock, err := s.blockState.BestBlockNumber()
	if err != nil {
		logger.Warnf("failed to get the highest block: %s", err)
		return 0
	}
	return highestBlock
}

func (s *SyncService) runSyncEngine() {
	defer s.wg.Done()
	s.waitWorkers()

	logger.Infof("starting sync engine with strategy: %T", s.currentStrategy)

	for {
		select {
		case <-s.stopCh:
			return
		case <-time.After(s.slotDuration):
		}

		s.runStrategy()

		if s.IsSynced() {
			isSyncedGauge.Set(1)
		} else {
			isSyncedGauge.Set(0)
		}
	}
}

func (s *SyncService) runStrategy() {
	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Tracef("running strategy: %T", s.currentStrategy)

	finalisedHeader, err := s.blockState.GetHighestFinalisedHeader()
	if err != nil {
		logger.Criticalf("getting highest finalized header: %w", err)
		return
	}

	bestBlockHeader, err := s.blockState.BestBlockHeader()
	if err != nil {
		logger.Criticalf("getting best block header: %w", err)
		return
	}

	logger.Infof(
		"🚣 currently syncing, %d peers connected, finalized #%d (%s), best #%d (%s)",
		len(s.network.AllConnectedPeersIDs()),
		finalisedHeader.Number,
		finalisedHeader.Hash().Short(),
		bestBlockHeader.Number,
		bestBlockHeader.Hash().Short(),
	)

	tasks, err := s.currentStrategy.NextActions()
	if err != nil {
		logger.Criticalf("current sync strategy next actions failed with: %s", err.Error())
		return
	}

	logger.Tracef("amount of tasks to process: %d", len(tasks))
	if len(tasks) == 0 {
		return
	}

	results := s.workerPool.submitRequests(tasks)
	done, repChanges, peersToIgnore, err := s.currentStrategy.Process(results)
	if err != nil {
		logger.Criticalf("current sync strategy failed with: %s", err.Error())
		return
	}

	for _, change := range repChanges {
		s.network.ReportPeer(change.rep, change.who)
	}

	for _, block := range peersToIgnore {
		s.workerPool.ignorePeerAsWorker(block)
	}

	s.currentStrategy.ShowMetrics()
	logger.Trace("finish process to acquire more blocks")

	if done {
		s.currentStrategy = s.defaultStrategy
	}
}
