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
	"os"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"

	log "github.com/ChainSafe/log15"
	"github.com/libp2p/go-libp2p-core/peer"
)

var logger = log.New("pkg", "sync")

// Service deals with chain syncing by sending block request messages and watching for responses.
type Service struct {
	// State interfaces
	blockState BlockState // retrieve our current head of chain from BlockState

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

	readyBlocks := make(chan *types.BlockData, 2048)
	chainSync := newChainSync(cfg.BlockState, cfg.Network, readyBlocks)

	return &Service{
		blockState: cfg.BlockState,
		chainSync:  chainSync,
	}, nil
}

// HandleBlockAnnounce creates a block request message from the block
// announce messages (block announce messages include the header but the full
// block is required to execute `core_execute_block`).
func (s *Service) HandleBlockAnnounce( /*from peer.ID, */ msg *network.BlockAnnounceMessage) error {
	logger.Debug("received BlockAnnounceMessage")

	// create header from message
	header, err := types.NewHeader(msg.ParentHash, msg.StateRoot, msg.ExtrinsicsRoot, msg.Number, msg.Digest)
	if err != nil {
		return err
	}

	s.chainSync.setBlockAnnounce(peer.ID(""), header)

	// // check if block header is stored in block state
	// has, err := s.blockState.HasHeader(header.Hash())
	// if err != nil {
	// 	return err
	// }

	// // save block header if we don't have it already
	// if has {
	// 	return nil
	// }

	// err = s.blockState.SetHeader(header)
	// if err != nil {
	// 	return err
	// }
	// logger.Debug(
	// 	"saved block header to block state",
	// 	"number", header.Number,
	// 	"hash", header.Hash(),
	// )
	return nil
}

// IsSynced exposes the synced state
func (s *Service) IsSynced() bool {
	return s.chainSync.syncState() == idle
}

func (s *Service) ProcessBlockData(data []*types.BlockData) (int, error) {
	return 0, nil
}

func (s *Service) ProcessJustification(data []*types.BlockData) (int, error) {
	return 0, nil
}

func (s *Service) SetSyncing(_ bool) {}
