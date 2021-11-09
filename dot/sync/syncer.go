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

package sync

import (
	"math/big"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/libp2p/go-libp2p-core/peer"
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
}

// NewService returns a new *sync.Service
func NewService(cfg *Config) (*Service, error) {
	if cfg.Network == nil {
		return nil, errNilNetwork
	}

	if cfg.BlockState == nil {
		return nil, errNilBlockState
	}

	if cfg.StorageState == nil {
		return nil, errNilStorageState
	}

	if cfg.FinalityGadget == nil {
		return nil, errNilFinalityGadget
	}

	if cfg.TransactionState == nil {
		return nil, errNilTransactionState
	}

	if cfg.BabeVerifier == nil {
		return nil, errNilVerifier
	}

	if cfg.BlockImportHandler == nil {
		return nil, errNilBlockImportHandler
	}

	logger.Patch(log.SetLevel(cfg.LogLvl))

	readyBlocks := newBlockQueue(maxResponseSize * 30)
	pendingBlocks := newDisjointBlockSet(pendingBlocksLimit)

	csCfg := &chainSyncConfig{
		bs:            cfg.BlockState,
		net:           cfg.Network,
		readyBlocks:   readyBlocks,
		pendingBlocks: pendingBlocks,
		minPeers:      cfg.MinPeers,
		maxPeers:      cfg.MaxPeers,
		slotDuration:  cfg.SlotDuration,
	}

	chainSync := newChainSync(csCfg)
	chainProcessor := newChainProcessor(readyBlocks, pendingBlocks, cfg.BlockState, cfg.StorageState, cfg.TransactionState, cfg.BabeVerifier, cfg.FinalityGadget, cfg.BlockImportHandler)

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
	go s.chainProcessor.start()
	return nil
}

// Stop stops the chainSync and chainProcessor modules
func (s *Service) Stop() error {
	s.chainSync.stop()
	s.chainProcessor.stop()
	return nil
}

// HandleBlockAnnounceHandshake notifies the `chainSync` module that we have received a BlockAnnounceHandshake from the given peer.
func (s *Service) HandleBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	return s.chainSync.setPeerHead(from, msg.BestBlockHash, big.NewInt(int64(msg.BestBlockNumber)))
}

// HandleBlockAnnounce notifies the `chainSync` module that we have received a block announcement from the given peer.
func (s *Service) HandleBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error {
	logger.Debug("received BlockAnnounceMessage")

	// create header from message
	header, err := types.NewHeader(msg.ParentHash, msg.StateRoot, msg.ExtrinsicsRoot, msg.Number, msg.Digest)
	if err != nil {
		return err
	}

	return s.chainSync.setBlockAnnounce(from, header)
}

// IsSynced exposes the synced state
func (s *Service) IsSynced() bool {
	return s.chainSync.syncState() == tip
}

func reverseBlockData(data []*types.BlockData) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
