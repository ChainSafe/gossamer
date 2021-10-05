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
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/runtime"

	log "github.com/ChainSafe/log15"
)

var logger = log.New("pkg", "sync")

// Service deals with chain syncing by sending block request messages and watching for responses.
type Service struct {
	// State interfaces
	blockState         BlockState // retrieve our current head of chain from BlockState
	storageState       StorageState
	transactionState   TransactionState
	finalityGadget     FinalityGadget
	blockImportHandler BlockImportHandler

	// Synchronisation variables
	synced           bool
	highestSeenBlock *big.Int // highest block number we have seen

	// BABE verification
	verifier Verifier
}

// Config is the configuration for the sync Service.
type Config struct {
	LogLvl             log.Lvl
	BlockState         BlockState
	StorageState       StorageState
	FinalityGadget     FinalityGadget
	TransactionState   TransactionState
	BlockImportHandler BlockImportHandler
	Runtime            runtime.Instance
	Verifier           Verifier
}

// NewService returns a new *sync.Service
func NewService(cfg *Config) (*Service, error) {
	if cfg.BlockState == nil {
		return nil, errNilBlockState
	}

	if cfg.StorageState == nil {
		return nil, errNilStorageState
	}

	if cfg.Verifier == nil {
		return nil, errNilVerifier
	}

	if cfg.BlockImportHandler == nil {
		return nil, errNilBlockImportHandler
	}

	handler := log.StreamHandler(os.Stdout, log.TerminalFormat())
	handler = log.CallerFileHandler(handler)
	logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, handler))

	return &Service{
		blockState:         cfg.BlockState,
		storageState:       cfg.StorageState,
		finalityGadget:     cfg.FinalityGadget,
		blockImportHandler: cfg.BlockImportHandler,
		synced:             true,
		highestSeenBlock:   big.NewInt(0),
		transactionState:   cfg.TransactionState,
		verifier:           cfg.Verifier,
	}, nil
}

// HandleBlockAnnounce creates a block request message from the block
// announce messages (block announce messages include the header but the full
// block is required to execute `core_execute_block`).
func (s *Service) HandleBlockAnnounce(msg *network.BlockAnnounceMessage) error {
	logger.Debug("received BlockAnnounceMessage")

	// create header from message
	header, err := types.NewHeader(msg.ParentHash, msg.StateRoot, msg.ExtrinsicsRoot, msg.Number, msg.Digest)
	if err != nil {
		return err
	}

	// check if block header is stored in block state
	has, err := s.blockState.HasHeader(header.Hash())
	if err != nil {
		return err
	}

	// save block header if we don't have it already
	if has {
		return nil
	}

	err = s.blockState.SetHeader(header)
	if err != nil {
		return err
	}
	logger.Debug(
		"saved block header to block state",
		"number", header.Number,
		"hash", header.Hash(),
	)
	return nil
}

// ProcessJustification processes block data containing justifications
func (s *Service) ProcessJustification(data []*types.BlockData) (int, error) {
	if len(data) == 0 {
		return 0, ErrNilBlockData
	}

	for i, bd := range data {
		header, err := s.blockState.GetHeader(bd.Hash)
		if err != nil {
			return i, err
		}

		if bd.Justification != nil {
			logger.Debug("handling Justification...", "number", header.Number, "hash", bd.Hash)
			s.handleJustification(header, *bd.Justification)
		}
	}

	return 0, nil
}

// ProcessBlockData processes the BlockData from a BlockResponse and returns the index of the last BlockData it handled on success,
// or the index of the block data that errored on failure.
func (s *Service) ProcessBlockData(data []*types.BlockData) (int, error) {
	if len(data) == 0 {
		return 0, ErrNilBlockData
	}

	for i, bd := range data {
		logger.Debug("starting processing of block", "hash", bd.Hash)

		err := s.blockState.CompareAndSetBlockData(bd)
		if err != nil {
			return i, fmt.Errorf("failed to compare and set data: %w", err)
		}

		hasHeader, _ := s.blockState.HasHeader(bd.Hash)
		hasBody, _ := s.blockState.HasBlockBody(bd.Hash)

		if hasHeader && hasBody {
			// TODO: fix this; sometimes when the node shuts down the "best block" isn't stored properly,
			// so when the node restarts it has blocks higher than what it thinks is the best, causing it not to sync
			logger.Debug("skipping block, already have", "hash", bd.Hash)

			block, err := s.blockState.GetBlockByHash(bd.Hash) //nolint
			if err != nil {
				logger.Debug("failed to get header", "hash", bd.Hash, "error", err)
				return i, err
			}

			err = s.blockState.AddBlockToBlockTree(&block.Header)
			if err != nil && !errors.Is(err, blocktree.ErrBlockExists) {
				logger.Warn("failed to add block to blocktree", "hash", bd.Hash, "error", err)
				return i, err
			}

			if bd.Justification != nil {
				logger.Debug("handling Justification...", "number", block.Header.Number, "hash", bd.Hash)
				s.handleJustification(&block.Header, *bd.Justification)
			}

			// TODO: this is probably unnecessary, since the state is already in the database
			// however, this case shouldn't be hit often, since it's only hit if the node state
			// is rewinded or if the node shuts down unexpectedly
			state, err := s.storageState.TrieState(&block.Header.StateRoot)
			if err != nil {
				logger.Warn("failed to load state for block", "block", block.Header.Hash(), "error", err)
				return i, err
			}

			if err := s.blockImportHandler.HandleBlockImport(block, state); err != nil {
				logger.Warn("failed to handle block import", "error", err)
			}

			continue
		}

		var header *types.Header

		if bd.Header != nil && !hasHeader {
			header = bd.Header

			logger.Trace("processing header", "hash", header.Hash(), "number", header.Number)

			err = s.handleHeader(header)
			if err != nil {
				return i, err
			}

			logger.Trace("header processed", "hash", bd.Hash)
		}

		if bd.Body != nil && !hasBody {
			body := bd.Body //nolint

			logger.Trace("processing body", "hash", bd.Hash)

			err = s.handleBody(body)
			if err != nil {
				return i, err
			}

			logger.Trace("body processed", "hash", bd.Hash)
		}

		if bd.Header != nil && bd.Body != nil {
			header = bd.Header
			body := bd.Body

			block := &types.Block{
				Header: *header,
				Body:   *body,
			}

			logger.Debug("processing block", "hash", bd.Hash)

			err = s.handleBlock(block)
			if err != nil {
				logger.Error("failed to handle block", "number", block.Header.Number, "error", err)
				return i, err
			}

			logger.Debug("block processed", "hash", bd.Hash)
		}

		if bd.Justification != nil && header != nil {
			logger.Debug("handling Justification...", "number", bd.Number(), "hash", bd.Hash)
			s.handleJustification(header, *bd.Justification)
		}
	}

	return len(data) - 1, nil
}

// handleHeader handles headers included in BlockResponses
func (s *Service) handleHeader(header *types.Header) error {
	// TODO: update BABE pre-runtime digest types
	err := s.verifier.VerifyBlock(header)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidBlock, err.Error())
	}

	return nil
}

// handleHeader handles block bodies included in BlockResponses
func (s *Service) handleBody(body *types.Body) error {
	exts, err := body.AsExtrinsics()
	if err != nil {
		logger.Error("cannot parse body as extrinsics", "error", err)
		return err
	}

	for _, ext := range exts {
		s.transactionState.RemoveExtrinsic(ext)
	}

	return err
}

// handleHeader handles blocks (header+body) included in BlockResponses
func (s *Service) handleBlock(block *types.Block) error {
	if block == nil || block.Empty() || block.Header.Empty() {
		return errors.New("block, header, or body is nil")
	}

	parent, err := s.blockState.GetHeader(block.Header.ParentHash)
	if err != nil {
		return fmt.Errorf("failed to get parent hash: %w", err)
	}

	s.storageState.Lock()
	defer s.storageState.Unlock()

	logger.Trace("getting parent state", "root", parent.StateRoot)
	ts, err := s.storageState.TrieState(&parent.StateRoot)
	if err != nil {
		return err
	}

	root := ts.MustRoot()
	if !bytes.Equal(parent.StateRoot[:], root[:]) {
		panic("parent state root does not match snapshot state root")
	}

	hash := parent.Hash()
	rt, err := s.blockState.GetRuntime(&hash)
	if err != nil {
		return err
	}

	rt.SetContextStorage(ts)
	logger.Trace("going to execute block", "header", block.Header, "exts", block.Body)

	_, err = rt.ExecuteBlock(block)
	if err != nil {
		return fmt.Errorf("failed to execute block %d: %w", block.Header.Number, err)
	}

	if err = s.blockImportHandler.HandleBlockImport(block, ts); err != nil {
		return err
	}

	logger.Debug("🔗 imported block", "number", block.Header.Number, "hash", block.Header.Hash())

	blockHash := block.Header.Hash()
	err = telemetry.GetInstance().SendMessage(telemetry.NewBlockImportTM(
		&blockHash,
		block.Header.Number,
		"NetworkInitialSync"))
	if err != nil {
		logger.Debug("problem sending block.import telemetry message", "error", err)
	}

	return nil
}

func (s *Service) handleJustification(header *types.Header, justification []byte) {
	if len(justification) == 0 || header == nil {
		return
	}

	err := s.finalityGadget.VerifyBlockJustification(header.Hash(), justification)
	if err != nil {
		logger.Warn("failed to verify block justification", "hash", header.Hash(), "number", header.Number, "error", err)
		return
	}

	err = s.blockState.SetJustification(header.Hash(), justification)
	if err != nil {
		logger.Error("failed tostore justification", "error", err)
		return
	}

	logger.Info("🔨 finalised block", "number", header.Number, "hash", header.Hash())
}

// IsSynced exposes the synced state
func (s *Service) IsSynced() bool {
	return s.synced
}

// SetSyncing sets whether the node is currently syncing or not
func (s *Service) SetSyncing(syncing bool) {
	s.synced = !syncing
	s.storageState.SetSyncing(syncing)
}
