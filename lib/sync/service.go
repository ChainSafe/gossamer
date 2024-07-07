package sync

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/libp2p/go-libp2p/core/peer"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "new-sync"))

type Network interface {
	AllConnectedPeersIDs() []peer.ID
	BlockAnnounceHandshake(*types.Header) error
}

type BlockState interface {
	BestBlockHeader() (*types.Header, error)
}

type Strategy interface {
	OnBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error
	NextActions() ([]*syncTask, error)
	IsFinished() (bool, error)
}

type SyncService struct {
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

func NewSyncService(network Network, blockState BlockState,
	currentStrategy, defaultStrategy Strategy) *SyncService {
	return &SyncService{
		network:           network,
		blockState:        blockState,
		currentStrategy:   currentStrategy,
		defaultStrategy:   defaultStrategy,
		workerPool:        newSyncWorkerPool(network),
		waitPeersDuration: 2 * time.Second,
		minPeers:          5,
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
		tasks, err := s.currentStrategy.NextActions()
		if err != nil {
			panic(fmt.Sprintf("current sync strategy next actions failed with: %s", err.Error()))
		}

		s.workerPool.submitRequests(tasks)

		done, err := s.currentStrategy.IsFinished()
		if err != nil {
			panic(fmt.Sprintf("current sync strategy failed with: %s", err.Error()))
		}

		if done {
			if s.defaultStrategy == nil {
				panic("nil default strategy")
			}

			s.currentStrategy = s.defaultStrategy
			s.defaultStrategy = nil
		}
	}
}
