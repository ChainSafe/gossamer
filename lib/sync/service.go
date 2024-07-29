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
	"github.com/libp2p/go-libp2p/core/peer"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "new-sync"))

type Network interface {
	AllConnectedPeersIDs() []peer.ID
	ReportPeer(change peerset.ReputationChange, p peer.ID)
	BlockAnnounceHandshake(*types.Header) error
	GetRequestResponseProtocol(subprotocol string, requestTimeout time.Duration,
		maxResponseSize uint64) *network.RequestResponseProtocol
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
	OnBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error
	OnBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error
	NextActions() ([]*syncTask, error)
	IsFinished(results []*syncTaskResult) (done bool, repChanges []Change, blocks []peer.ID, err error)
	ShowMetrics()
}

type BlockOrigin byte

const (
	networkInitialSync BlockOrigin = iota
	networkBroadcast
)

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

	stopCh chan struct{}
}

func NewSyncService(network Network,
	blockState BlockState,
	currentStrategy, defaultStrategy Strategy) *SyncService {
	return &SyncService{
		network:           network,
		blockState:        blockState,
		currentStrategy:   currentStrategy,
		defaultStrategy:   defaultStrategy,
		workerPool:        newSyncWorkerPool(network),
		waitPeersDuration: 2 * time.Second,
		minPeers:          1,
		stopCh:            make(chan struct{}),
	}
}

func (s *SyncService) waitWorkers() {
	waitPeersTimer := time.NewTimer(s.waitPeersDuration)

	bestBlockHeader, err := s.blockState.BestBlockHeader()
	if err != nil {
		panic(fmt.Sprintf("failed to get highest finalised header: %v", err))
	}

	for {
		total := s.workerPool.totalWorkers()
		logger.Info("waiting peers...")
		logger.Infof("total workers: %d, min peers: %d", total, s.minPeers)
		if total >= s.minPeers {
			return
		}

		err := s.network.BlockAnnounceHandshake(bestBlockHeader)
		if err != nil && !errors.Is(err, network.ErrNoPeersConnected) {
			logger.Errorf("retrieving target info from peers: %v", err)
		}

		select {
		case <-waitPeersTimer.C:
			waitPeersTimer.Reset(s.waitPeersDuration)

		case <-s.stopCh:
			return
		}
	}
}

func (s *SyncService) Start() error {
	s.waitWorkers()

	s.wg.Add(1)
	go s.runSyncEngine()
	return nil
}

func (s *SyncService) Stop() error {
	// TODO: implement stop mechanism
	close(s.stopCh)
	s.wg.Wait()
	return nil
}

func (s *SyncService) HandleBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	logger.Infof("receiving a block announce handshake: %s", from.String())
	s.workerPool.fromBlockAnnounceHandshake(from, msg.BestBlockHash, uint(msg.BestBlockNumber))

	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentStrategy.OnBlockAnnounceHandshake(from, msg)
	return nil
}

func (s *SyncService) HandleBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error {
	return s.currentStrategy.OnBlockAnnounce(from, msg)
}

func (s *SyncService) OnConnectionClosed(who peer.ID) {
	logger.Tracef("removing peer worker: %s", who.String())
	s.workerPool.removeWorker(who)
}

func (s *SyncService) CreateBlockResponse(who peer.ID, req *network.BlockRequestMessage) (
	*network.BlockResponseMessage, error) {
	return nil, nil
}

func (s *SyncService) IsSynced() bool {
	return false
}

func (s *SyncService) HighestBlock() uint {
	return 0
}

func (s *SyncService) runSyncEngine() {
	defer s.wg.Done()

	logger.Infof("starting sync engine with strategy: %T", s.currentStrategy)

	// TODO: need to handle stop channel
	for {
		finalisedHeader, err := s.blockState.GetHighestFinalisedHeader()
		if err != nil {
			logger.Criticalf("getting highest finalized header: %w", err)
			return
		}

		logger.Infof(
			"🚣 currently syncing, %d peers connected, last finalised #%d (%s) ",
			len(s.network.AllConnectedPeersIDs()),
			s.workerPool.totalWorkers(),
			finalisedHeader.Number,
			finalisedHeader.Hash().Short(),
		)

		tasks, err := s.currentStrategy.NextActions()
		if err != nil {
			panic(fmt.Sprintf("current sync strategy next actions failed with: %s", err.Error()))
		}

		logger.Infof("sending %d tasks", len(tasks))
		results := s.workerPool.submitRequests(tasks)

		done, repChanges, blocks, err := s.currentStrategy.IsFinished(results)
		if err != nil {
			panic(fmt.Sprintf("current sync strategy failed with: %s", err.Error()))
		}

		for _, change := range repChanges {
			s.network.ReportPeer(change.rep, change.who)
		}

		for _, block := range blocks {
			s.workerPool.ignorePeerAsWorker(block)
		}

		s.currentStrategy.ShowMetrics()

		if done {
			if s.defaultStrategy == nil {
				panic("nil default strategy")
			}

			s.mu.Lock()
			s.currentStrategy = s.defaultStrategy
			s.mu.Unlock()
		}
	}
}
