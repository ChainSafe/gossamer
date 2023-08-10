// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/libp2p/go-libp2p/core/peer"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "sync"))

// Service deals with chain syncing by sending block request messages and watching for responses.
type Service struct {
	blockState     BlockState
	chainSync      ChainSync
	chainProcessor ChainProcessor
	network        Network
}

// Config is the configuration for the sync Service.
type Config struct {
	LogLvl             log.Level
	Network            Network
	BlockState         BlockState
	StorageState       StorageState
	FinalityGadget     FinalityGadget
	TransactionState   TransactionState
	BlockImportHandler BlockImportHandler
	BabeVerifier       BabeVerifier
	MinPeers, MaxPeers int
	SlotDuration       time.Duration
	Telemetry          Telemetry
	BadBlocks          []string
}

// NewService returns a new *sync.Service
func NewService(cfg *Config, blockReqRes network.RequestMaker) (*Service, error) {
	logger.Patch(log.SetLevel(cfg.LogLvl))

	readyBlocks := newBlockQueue(maxResponseSize * 30)
	pendingBlocks := newDisjointBlockSet(pendingBlocksLimit)

	csCfg := chainSyncConfig{
		bs:            cfg.BlockState,
		net:           cfg.Network,
		readyBlocks:   readyBlocks,
		pendingBlocks: pendingBlocks,
		minPeers:      cfg.MinPeers,
		maxPeers:      cfg.MaxPeers,
		slotDuration:  cfg.SlotDuration,
	}
	chainSync := newChainSync(csCfg, blockReqRes)

	cpCfg := chainProcessorConfig{
		readyBlocks:        readyBlocks,
		pendingBlocks:      pendingBlocks,
		syncer:             chainSync,
		blockState:         cfg.BlockState,
		storageState:       cfg.StorageState,
		transactionState:   cfg.TransactionState,
		babeVerifier:       cfg.BabeVerifier,
		finalityGadget:     cfg.FinalityGadget,
		blockImportHandler: cfg.BlockImportHandler,
		telemetry:          cfg.Telemetry,
		badBlocks:          cfg.BadBlocks,
	}
	chainProcessor := newChainProcessor(cpCfg)

	return &Service{
		blockState:     cfg.BlockState,
		chainSync:      chainSync,
		chainProcessor: chainProcessor,
		network:        cfg.Network,
	}, nil
}

// Start begins the chainSync and chainProcessor modules. It begins syncing in bootstrap mode
func (s *Service) Start() error {
	go s.chainSync.start()
	go s.chainProcessor.processReadyBlocks()
	return nil
}

// Stop stops the chainSync and chainProcessor modules
func (s *Service) Stop() error {
	s.chainSync.stop()
	s.chainProcessor.stop()
	return nil
}

// HandleBlockAnnounceHandshake notifies the `chainSync` module that
// we have received a BlockAnnounceHandshake from the given peer.
func (s *Service) HandleBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	return s.chainSync.setPeerHead(from, msg.BestBlockHash, uint(msg.BestBlockNumber))
}

// HandleBlockAnnounce notifies the `chainSync` module that we have received a block announcement from the given peer.
func (s *Service) HandleBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error {
	logger.Debug("received BlockAnnounceMessage")
	header := types.NewHeader(msg.ParentHash, msg.StateRoot, msg.ExtrinsicsRoot, msg.Number, msg.Digest)
	return s.chainSync.setBlockAnnounce(from, header)
}

// IsSynced exposes the synced state
func (s *Service) IsSynced() bool {
	return s.chainSync.syncState() == tip
}

// HighestBlock gets the highest known block number
func (s *Service) HighestBlock() uint {
	highestBlock, err := s.chainSync.getHighestBlock()
	if err != nil {
		logger.Warnf("failed to get the highest block: %s", err)
		return 0
	}
	return highestBlock
}

func reverseBlockData(data []*types.BlockData) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
