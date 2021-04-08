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
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	log "github.com/ChainSafe/log15"
)

var logger = log.New("pkg", "sync")

// Service deals with chain syncing by sending block request messages and watching for responses.
type Service struct {
	codeHash common.Hash // cached hash of runtime code

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

	handler := log.StreamHandler(os.Stdout, log.TerminalFormat())
	handler = log.CallerFileHandler(handler)
	logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, handler))

	codeHash, err := cfg.StorageState.LoadCodeHash(nil)
	if err != nil {
		return nil, err
	}

	return &Service{
		codeHash:         codeHash,
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
	logger.Debug("received BlockAnnounceMessage")

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
		logger.Debug(
			"saved block header to block state",
			"number", header.Number,
			"hash", header.Hash(),
		)
	}

	return nil
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

			header, err := s.blockState.GetHeader(bd.Hash) //nolint
			if err != nil {
				logger.Debug("failed to get header", "hash", bd.Hash, "error", err)
				return i, err
			}

			err = s.blockState.AddBlockToBlockTree(header)
			if err != nil {
				logger.Debug("failed to add block to blocktree", "hash", bd.Hash, "error", err)
			}

			if bd.Justification != nil && bd.Justification.Exists() {
				logger.Debug("handling Justification...", "number", header.Number, "hash", bd.Hash)
				s.handleJustification(header, bd.Justification.Value())
			}

			continue
		}

		var header *types.Header

		if bd.Header.Exists() && !hasHeader {
			header, err = types.NewHeaderFromOptional(bd.Header)
			if err != nil {
				return i, err
			}

			logger.Trace("processing header", "hash", header.Hash(), "number", header.Number)

			err = s.handleHeader(header)
			if err != nil {
				return i, err
			}

			logger.Trace("header processed", "hash", bd.Hash)
		}

		if bd.Body.Exists() && !hasBody {
			body, err := types.NewBodyFromOptional(bd.Body) //nolint
			if err != nil {
				return i, err
			}

			logger.Trace("processing body", "hash", bd.Hash)

			err = s.handleBody(body)
			if err != nil {
				return i, err
			}

			logger.Trace("body processed", "hash", bd.Hash)
		}

		if bd.Header.Exists() && bd.Body.Exists() {
			header, err = types.NewHeaderFromOptional(bd.Header)
			if err != nil {
				return i, err
			}

			body, err := types.NewBodyFromOptional(bd.Body)
			if err != nil {
				return i, err
			}

			block := &types.Block{
				Header: header,
				Body:   body,
			}

			logger.Debug("processing block", "hash", bd.Hash)

			err = s.handleBlock(block)
			if err != nil {
				logger.Error("failed to handle block", "number", block.Header.Number, "error", err)
				return i, err
			}

			logger.Debug("block processed", "hash", bd.Hash)
		}

		if bd.Justification != nil && bd.Justification.Exists() && header != nil {
			logger.Debug("handling Justification...", "number", bd.Number(), "hash", bd.Hash)
			s.handleJustification(header, bd.Justification.Value())
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
	if block == nil || block.Header == nil || block.Body == nil {
		return errors.New("block, header, or body is nil")
	}

	parent, err := s.blockState.GetHeader(block.Header.ParentHash)
	if err != nil {
		return fmt.Errorf("failed to get parent hash: %w", err)
	}

	logger.Trace("getting parent state", "root", parent.StateRoot)
	ts, err := s.storageState.TrieState(&parent.StateRoot)
	if err != nil {
		return err
	}

	root := ts.MustRoot()
	if !bytes.Equal(parent.StateRoot[:], root[:]) {
		panic("parent state root does not match snapshot state root")
	}

	s.runtime.SetContextStorage(ts)
	logger.Trace("going to execute block", "header", block.Header, "exts", block.Body)

	_, err = s.runtime.ExecuteBlock(block)
	if err != nil {
		return fmt.Errorf("failed to execute block %d: %w", block.Header.Number, err)
	}

	err = s.storageState.StoreTrie(ts)
	if err != nil {
		return err
	}
	logger.Trace("stored resulting state", "state root", ts.MustRoot())

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
		logger.Debug("🔗 imported block", "number", block.Header.Number, "hash", block.Header.Hash())
		telemetry.GetInstance().SendBlockImport(block.Header.Hash().String(), block.Header.Number)
	}

	// handle consensus digest for authority changes
	if s.digestHandler != nil {
		s.handleDigests(block.Header)
	}

	return s.handleRuntimeChanges(ts)
}

func (s *Service) handleJustification(header *types.Header, justification []byte) {
	if len(justification) == 0 || header == nil {
		return
	}

	err := s.blockState.SetFinalizedHash(header.Hash(), 0, 0)
	if err != nil {
		logger.Error("failed to set finalized hash", "error", err)
		return
	}

	err = s.blockState.SetJustification(header.Hash(), justification)
	if err != nil {
		logger.Error("failed tostore justification", "error", err)
		return
	}

	logger.Info("🔨 finalized block", "number", header.Number, "hash", header.Hash())
}

func (s *Service) handleRuntimeChanges(newState *rtstorage.TrieState) error {
	currCodeHash, err := newState.LoadCodeHash()
	if err != nil {
		return err
	}

	if bytes.Equal(s.codeHash[:], currCodeHash[:]) {
		return nil
	}

	logger.Info("🔄 detected runtime code change, upgrading...", "block", s.blockState.BestBlockHash(), "previous code hash", s.codeHash, "new code hash", currCodeHash)
	code := newState.LoadCode()
	if len(code) == 0 {
		return ErrEmptyRuntimeCode
	}

	err = s.runtime.UpdateRuntimeCode(code)
	if err != nil {
		logger.Crit("failed to update runtime code", "error", err)
		return err
	}

	s.codeHash = currCodeHash
	return nil
}

func (s *Service) handleDigests(header *types.Header) {
	for i, d := range header.Digest {
		if d.Type() == types.ConsensusDigestType {
			cd, ok := d.(*types.ConsensusDigest)
			if !ok {
				logger.Error("handleDigests", "index", i, "error", "cannot cast invalid consensus digest item")
				continue
			}

			err := s.digestHandler.HandleConsensusDigest(cd, header)
			if err != nil {
				logger.Error("handleDigests", "index", i, "digest", cd, "error", err)
			}
		}
	}
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
