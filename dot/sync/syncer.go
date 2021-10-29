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
	"os"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"

	log "github.com/ChainSafe/log15"
	"github.com/libp2p/go-libp2p-core/peer"
)

var logger = log.New("pkg", "sync")

// Service deals with chain syncing by sending block request messages and watching for responses.
type Service struct {
	blockState     BlockState
	chainSync      ChainSync
	chainProcessor ChainProcessor
}

// Config is the configuration for the sync Service.
type Config struct {
	LogLvl             log.Lvl
	Network            Network
	BlockState         BlockState
	StorageState       StorageState
	FinalityGadget     FinalityGadget
	TransactionState   TransactionState
	BlockImportHandler BlockImportHandler
	BabeVerifier       BabeVerifier
	MinPeers           int
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

	handler := log.StreamHandler(os.Stdout, log.TerminalFormat())
	handler = log.CallerFileHandler(handler)
	logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, handler))

	readyBlocks := newBlockQueue(maxResponseSize * 30)
	pendingBlocks := newDisjointBlockSet(pendingBlocksLimit)
	chainSync := newChainSync(cfg.BlockState, cfg.Network, readyBlocks, pendingBlocks, cfg.MinPeers, cfg.SlotDuration)
	chainProcessor := newChainProcessor(readyBlocks, pendingBlocks, cfg.BlockState, cfg.StorageState, cfg.TransactionState, cfg.BabeVerifier, cfg.FinalityGadget, cfg.BlockImportHandler)

	return &Service{
		blockState:     cfg.BlockState,
		chainSync:      chainSync,
		chainProcessor: chainProcessor,
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
