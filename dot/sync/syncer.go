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
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/runtime"

	log "github.com/ChainSafe/log15"
)

// Service deals with chain syncing by sending block request messages and watching for responses.
type Service struct {
	logger log.Logger

	// State interfaces
	blockState       BlockState // retrieve our current head of chain from BlockState
	storageState     StorageState
	transactionState TransactionState
	blockProducer    BlockProducer

	// Synchronization variables
	synced           bool
	highestSeenBlock *big.Int // highest block number we have seen
	runtime          runtime.Instance

	// BABE verification
	verifier Verifier

	// Consensus digest handling
	digestHandler DigestHandler
}

// Config is the configuration for the sync Service.
type Config struct {
	LogLvl           log.Lvl
	BlockState       BlockState
	StorageState     StorageState
	BlockProducer    BlockProducer
	TransactionState TransactionState
	Runtime          runtime.Instance
	Verifier         Verifier
	DigestHandler    DigestHandler
}

// NewService returns a new *sync.Service
func NewService(cfg *Config) (*Service, error) {
	if cfg.BlockState == nil {
		return nil, ErrNilBlockState
	}

	if cfg.StorageState == nil {
		return nil, ErrNilStorageState
	}

	if cfg.Verifier == nil {
		return nil, ErrNilVerifier
	}

	if cfg.Runtime == nil {
		return nil, ErrNilRuntime
	}

	if cfg.BlockProducer == nil {
		cfg.BlockProducer = newMockBlockProducer()
	}

	logger := log.New("pkg", "sync")
	handler := log.StreamHandler(os.Stdout, log.TerminalFormat())
	handler = log.CallerFileHandler(handler)
	logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, handler))

	return &Service{
		logger:           logger,
		blockState:       cfg.BlockState,
		storageState:     cfg.StorageState,
		blockProducer:    cfg.BlockProducer,
		synced:           true,
		highestSeenBlock: big.NewInt(0),
		transactionState: cfg.TransactionState,
		runtime:          cfg.Runtime,
		verifier:         cfg.Verifier,
		digestHandler:    cfg.DigestHandler,
	}, nil
}

// HandleBlockAnnounce creates a block request message from the block
// announce messages (block announce messages include the header but the full
// block is required to execute `core_execute_block`).
func (s *Service) HandleBlockAnnounce(msg *network.BlockAnnounceMessage) error {
	s.logger.Debug("received BlockAnnounceMessage")

	// create header from message
	header, err := types.NewHeader(
		msg.ParentHash,
		msg.Number,
		msg.StateRoot,
		msg.ExtrinsicsRoot,
		msg.Digest,
	)
	if err != nil {
		return err
	}

	// check if block header is stored in block state
	has, err := s.blockState.HasHeader(header.Hash())
	if err != nil {
		return err
	}

	// save block header if we don't have it already
	if !has {
		err = s.blockState.SetHeader(header)
		if err != nil {
			return err
		}
		s.logger.Debug(
			"saved block header to block state",
			"number", header.Number,
			"hash", header.Hash(),
		)
	}

	return nil
}

// ProcessBlockData processes the BlockData from a BlockResponse and returns the index of the last BlockData it successfully handled.
func (s *Service) ProcessBlockData(data []*types.BlockData) error {
	if len(data) == 0 {
		return ErrNilBlockData
	}

	// TODO: return number of last successful block that was processed
	for _, bd := range data {
		s.logger.Debug("starting processing of block", "hash", bd.Hash)

		err := s.blockState.CompareAndSetBlockData(bd)
		if err != nil {
			return err
		}

		hasHeader, _ := s.blockState.HasHeader(bd.Hash)
		hasBody, _ := s.blockState.HasBlockBody(bd.Hash)
		if hasHeader && hasBody {
			s.logger.Debug("skipping block, already have", "hash", bd.Hash)
			continue
		}

		if bd.Header.Exists() && !hasHeader {
			header, err := types.NewHeaderFromOptional(bd.Header)
			if err != nil {
				return err
			}

			s.logger.Trace("processing header", "hash", header.Hash(), "number", header.Number)

			err = s.handleHeader(header)
			if err != nil {
				return err
			}

			s.logger.Trace("header processed", "hash", bd.Hash)
		}

		if bd.Body.Exists() && !hasBody {
			body, err := types.NewBodyFromOptional(bd.Body)
			if err != nil {
				return err
			}

			s.logger.Trace("processing body", "hash", bd.Hash)

			err = s.handleBody(body)
			if err != nil {
				return err
			}

			s.logger.Trace("body processed", "hash", bd.Hash)
		}

		if bd.Header.Exists() && bd.Body.Exists() {
			header, err := types.NewHeaderFromOptional(bd.Header)
			if err != nil {
				return err
			}

			body, err := types.NewBodyFromOptional(bd.Body)
			if err != nil {
				return err
			}

			block := &types.Block{
				Header: header,
				Body:   body,
			}

			s.logger.Debug("processing block", "hash", bd.Hash)

			err = s.handleBlock(block)
			if err != nil {
				return err
			}

			s.logger.Debug("block processed", "hash", bd.Hash)
		}
	}

	return nil
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
		s.logger.Error("cannot parse body as extrinsics", "error", err)
		return err
	}

	s.logger.Trace("block extrinsics", "extrinsics", exts)

	for _, ext := range exts {
		s.transactionState.RemoveExtrinsic(ext)
	}

	return err
}

// handleHeader handles blocks (header+body) included in BlockResponses
func (s *Service) handleBlock(block *types.Block) error {
	if block == nil || block.Header == nil || block.Body == nil {
		return errors.New("block, header, or body is nil")
	}

	parent, err := s.blockState.GetHeader(block.Header.ParentHash)
	if err != nil {
		return err
	}

	s.logger.Trace("getting parent state", "root", parent.StateRoot)
	parentState, err := s.storageState.TrieState(&parent.StateRoot)
	if err != nil {
		return err
	}

	ts, err := parentState.Copy()
	if err != nil {
		return err
	}

	s.logger.Trace("copied parent state", "parent state root", parentState.MustRoot(), "copy state root", ts.MustRoot())
	// sanity check
	if parentState.MustRoot() != ts.MustRoot() {
		panic("parent state root does not match copy's state root")
	}
	s.runtime.SetContextStorage(ts)
	s.logger.Trace("going to execute block", "block", block, "exts", block.Body)

	_, err = s.runtime.ExecuteBlock(block)
	if err != nil {
		return fmt.Errorf("failed to execute block %d: %w", block.Header.Number, err)
	}

	err = s.storageState.StoreTrie(ts)
	if err != nil {
		return err
	}
	s.logger.Trace("stored resulting state", "state root", ts.MustRoot())

	// TODO: batch writes in AddBlock
	err = s.blockState.AddBlock(block)
	if err != nil {
		if err == blocktree.ErrParentNotFound && block.Header.Number.Cmp(big.NewInt(0)) != 0 {
			return err
		} else if err == blocktree.ErrBlockExists || block.Header.Number.Cmp(big.NewInt(0)) == 0 {
			// this is fine
		} else {
			return err
		}
	} else {
		s.logger.Info("imported block", "number", block.Header.Number, "hash", block.Header.Hash())
		s.logger.Debug("imported block", "header", block.Header, "body", block.Body)
	}

	// handle consensus digest for authority changes
	if s.digestHandler != nil {
		go func() {
			err = s.handleDigests(block.Header)
			if err != nil {
				s.logger.Error("failed to handle block digest", "error", err)
			}
		}()
	}

	return nil
}

func (s *Service) handleDigests(header *types.Header) error {
	for _, d := range header.Digest {
		if d.Type() == types.ConsensusDigestType {
			cd, ok := d.(*types.ConsensusDigest)
			if !ok {
				return errors.New("cannot cast invalid consensus digest item")
			}

			err := s.digestHandler.HandleConsensusDigest(cd, header)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// IsSynced exposes the synced state
func (s *Service) IsSynced() bool {
	return s.synced
}
