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
	"context"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
)

// ChainProcessor processes ready blocks.
// it is implemented by *chainProcessor
type ChainProcessor interface {
	start()
	stop()
}

type chainProcessor struct {
	ctx    context.Context
	cancel context.CancelFunc

	// blocks that are ready for processing. ie. their parent is known, or their parent is ahead
	// of them within this channel and thus will be processed first
	readyBlocks *blockQueue

	// set of block not yet ready to be processed.
	// blocks are placed here if they fail to be processed due to missing parent block
	pendingBlocks DisjointBlockSet

	blockState         BlockState
	storageState       StorageState
	transactionState   TransactionState
	babeVerifier       BabeVerifier
	finalityGadget     FinalityGadget
	blockImportHandler BlockImportHandler
}

func newChainProcessor(readyBlocks *blockQueue, pendingBlocks DisjointBlockSet, blockState BlockState, storageState StorageState, transactionState TransactionState, babeVerifier BabeVerifier, finalityGadget FinalityGadget, blockImportHandler BlockImportHandler) *chainProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &chainProcessor{
		ctx:                ctx,
		cancel:             cancel,
		readyBlocks:        readyBlocks,
		pendingBlocks:      pendingBlocks,
		blockState:         blockState,
		storageState:       storageState,
		transactionState:   transactionState,
		babeVerifier:       babeVerifier,
		finalityGadget:     finalityGadget,
		blockImportHandler: blockImportHandler,
	}
}

func (s *chainProcessor) start() {
	go s.processReadyBlocks()
}

func (s *chainProcessor) stop() {
	s.cancel()
}

func (s *chainProcessor) processReadyBlocks() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		bd := s.readyBlocks.pop()
		if bd == nil {
			continue
		}

		if err := s.processBlockData(bd); err != nil {
			logger.Error("ready block failed", "hash", bd.Hash, "error", err)

			// depending on the error, we might want to save this block for later
			if errors.Is(err, errFailedToGetParent) {
				if err := s.pendingBlocks.addBlock(&types.Block{
					Header: *bd.Header,
					Body:   *bd.Body,
				}); err != nil {
					logger.Debug("failed to re-add block to pending blocks", "error", err)
				}
			}
		}
	}
}

// processBlockData processes the BlockData from a BlockResponse and returns the index of the last BlockData it handled on success,
// or the index of the block data that errored on failure.
func (s *chainProcessor) processBlockData(bd *types.BlockData) error {
	if bd == nil {
		return ErrNilBlockData
	}

	err := s.blockState.CompareAndSetBlockData(bd)
	if err != nil {
		return fmt.Errorf("failed to compare and set data: %w", err)
	}

	hasHeader, _ := s.blockState.HasHeader(bd.Hash)
	hasBody, _ := s.blockState.HasBlockBody(bd.Hash)
	if hasHeader && hasBody {
		// TODO: fix this; sometimes when the node shuts down the "best block" isn't stored properly,
		// so when the node restarts it has blocks higher than what it thinks is the best, causing it not to sync
		// if we update the node to only store finalised blocks in the database, this should be fixed and the entire
		// code block can be removed (#1784)
		block, err := s.blockState.GetBlockByHash(bd.Hash) //nolint
		if err != nil {
			logger.Debug("failed to get header", "hash", bd.Hash, "error", err)
			return err
		}

		logger.Debug("skipping block, already have", "hash", bd.Hash, "number", block.Header.Number)

		err = s.blockState.AddBlockToBlockTree(&block.Header)
		if errors.Is(err, blocktree.ErrBlockExists) {
			return nil
		} else if err != nil {
			logger.Warn("failed to add block to blocktree", "hash", bd.Hash, "error", err)
			return err
		}

		if bd.Justification != nil {
			logger.Debug("handling Justification...", "number", block.Header.Number, "hash", bd.Hash)
			s.handleJustification(&block.Header, *bd.Justification)
		}

		// TODO: this is probably unnecessary, since the state is already in the database
		// however, this case shouldn't be hit often, since it's only hit if the node state
		// is rewinded or if the node shuts down unexpectedly (#1784)
		state, err := s.storageState.TrieState(&block.Header.StateRoot)
		if err != nil {
			logger.Warn("failed to load state for block", "block", block.Header.Hash(), "error", err)
			return err
		}

		if err := s.blockImportHandler.HandleBlockImport(block, state); err != nil {
			logger.Warn("failed to handle block import", "error", err)
		}

		return nil
	}

	if bd.Header != nil && bd.Body != nil {
		if err = s.handleHeader(bd.Header); err != nil {
			return err
		}

		s.handleBody(bd.Body)

		block := &types.Block{
			Header: *bd.Header,
			Body:   *bd.Body,
		}

		logger.Debug("processing block", "hash", bd.Hash)

		if err = s.handleBlock(block); err != nil {
			logger.Error("failed to handle block", "number", block.Header.Number, "error", err)
			return err
		}

		logger.Debug("block processed", "hash", bd.Hash)
	}

	if bd.Justification != nil && bd.Header != nil {
		logger.Debug("handling Justification...", "number", bd.Number(), "hash", bd.Hash)
		s.handleJustification(bd.Header, *bd.Justification)
	}

	return nil
}

// handleHeader handles headers included in BlockResponses
func (s *chainProcessor) handleHeader(header *types.Header) error {
	err := s.babeVerifier.VerifyBlock(header)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidBlock, err.Error())
	}

	return nil
}

// handleHeader handles block bodies included in BlockResponses
func (s *chainProcessor) handleBody(body *types.Body) {
	for _, ext := range *body {
		s.transactionState.RemoveExtrinsic(ext)
	}
}

// handleHeader handles blocks (header+body) included in BlockResponses
func (s *chainProcessor) handleBlock(block *types.Block) error {
	if block == nil || block.Body == nil {
		return errors.New("block or body is nil")
	}

	parent, err := s.blockState.GetHeader(block.Header.ParentHash)
	if err != nil {
		return fmt.Errorf("%w: %s", errFailedToGetParent, err)
	}

	s.storageState.Lock()
	defer s.storageState.Unlock()

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

func (s *chainProcessor) handleJustification(header *types.Header, justification []byte) {
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
