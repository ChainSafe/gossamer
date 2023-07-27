// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/libp2p/go-libp2p/core/peer"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "sync"))

// Service deals with chain syncing by sending block request messages and watching for responses.
type Service struct {
	blockState BlockState
	chainSync  ChainSync
	network    Network
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
	RequestMaker       network.RequestMaker
}

// NewService returns a new *sync.Service
func NewService(cfg *Config) (*Service, error) {
	logger.Patch(log.SetLevel(cfg.LogLvl))

	pendingBlocks := newDisjointBlockSet(pendingBlocksLimit)

	csCfg := chainSyncConfig{
		bs:                 cfg.BlockState,
		net:                cfg.Network,
		pendingBlocks:      pendingBlocks,
		minPeers:           cfg.MinPeers,
		maxPeers:           cfg.MaxPeers,
		slotDuration:       cfg.SlotDuration,
		storageState:       cfg.StorageState,
		transactionState:   cfg.TransactionState,
		babeVerifier:       cfg.BabeVerifier,
		finalityGadget:     cfg.FinalityGadget,
		blockImportHandler: cfg.BlockImportHandler,
		telemetry:          cfg.Telemetry,
		badBlocks:          cfg.BadBlocks,
		requestMaker:       cfg.RequestMaker,
	}
	chainSync := newChainSync(csCfg)

	return &Service{
		blockState: cfg.BlockState,
		chainSync:  chainSync,
		network:    cfg.Network,
	}, nil
}

// Start begins the chainSync and chainProcessor modules. It begins syncing in bootstrap mode
func (s *Service) Start() error {
	go s.chainSync.start()
	return nil
}

// Stop stops the chainSync and chainProcessor modules
func (s *Service) Stop() error {
	s.chainSync.stop()
	return nil
}

// HandleBlockAnnounceHandshake notifies the `chainSync` module that
// we have received a BlockAnnounceHandshake from the given peer.
func (s *Service) HandleBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	return s.chainSync.onBlockAnnounceHandshake(from, msg.BestBlockHash, uint(msg.BestBlockNumber))
}

// HandleBlockAnnounce notifies the `chainSync` module that we have received a block announcement from the given peer.
func (s *Service) HandleBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error {
	logger.Debug("received BlockAnnounceMessage")
	blockAnnounceHeader := types.NewHeader(msg.ParentHash, msg.StateRoot, msg.ExtrinsicsRoot, msg.Number, msg.Digest)
	blockAnnounceHeaderHash := blockAnnounceHeader.Hash()

	// if the peer reports a lower or equal best block number than us,
	// check if they are on a fork or not
	bestBlockHeader, err := s.blockState.BestBlockHeader()
	if err != nil {
		return fmt.Errorf("best block header: %w", err)
	}

	if blockAnnounceHeader.Number <= bestBlockHeader.Number {
		// check if our block hash for that number is the same, if so, do nothing
		// as we already have that block
		ourHash, err := s.blockState.GetHashByNumber(blockAnnounceHeader.Number)
		if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
			return fmt.Errorf("get block hash by number: %w", err)
		}

		if ourHash == blockAnnounceHeaderHash {
			return nil
		}

		// check if their best block is on an invalid chain, if it is,
		// potentially downscore them
		// for now, we can remove them from the syncing peers set
		fin, err := s.blockState.GetHighestFinalisedHeader()
		if err != nil {
			return fmt.Errorf("get highest finalised header: %w", err)
		}

		// their block hash doesn't match ours for that number (ie. they are on a different
		// chain), and also the highest finalised block is higher than that number.
		// thus the peer is on an invalid chain
		if fin.Number >= blockAnnounceHeader.Number && msg.BestBlock {
			// TODO: downscore this peer, or temporarily don't sync from them? (#1399)
			// perhaps we need another field in `peerState` to mark whether the state is valid or not
			s.network.ReportPeer(peerset.ReputationChange{
				Value:  peerset.BadBlockAnnouncementValue,
				Reason: peerset.BadBlockAnnouncementReason,
			}, from)
			return fmt.Errorf("%w: for peer %s and block number %d",
				errPeerOnInvalidFork, from, blockAnnounceHeader.Number)
		}

		// peer is on a fork, check if we have processed the fork already or not
		// ie. is their block written to our db?
		has, err := s.blockState.HasHeader(blockAnnounceHeaderHash)
		if err != nil {
			return fmt.Errorf("while checking if header exists: %w", err)
		}

		// if so, do nothing, as we already have their fork
		if has {
			return nil
		}
	}

	// we assume that if a peer sends us a block announce for a certain block,
	// that is also has the chain up until and including that block.
	// this may not be a valid assumption, but perhaps we can assume that
	// it is likely they will receive this block and its ancestors before us.
	return s.chainSync.onBlockAnnounce(announcedBlock{
		who:    from,
		header: blockAnnounceHeader,
	})
}

// IsSynced exposes the synced state
func (s *Service) IsSynced() bool {
	return s.chainSync.getSyncMode() == tip
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
