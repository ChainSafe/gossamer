// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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
	processReadyBlocks()
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
	telemetry          telemetry.Client
}

func newChainProcessor(readyBlocks *blockQueue, pendingBlocks DisjointBlockSet,
	blockState BlockState, storageState StorageState,
	transactionState TransactionState, babeVerifier BabeVerifier,
	finalityGadget FinalityGadget, blockImportHandler BlockImportHandler, telemetry telemetry.Client) *chainProcessor {
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
		telemetry:          telemetry,
	}
}

func (s *chainProcessor) stop() {
	s.cancel()
}

func (s *chainProcessor) processReadyBlocks() {
	for {
		bd := s.readyBlocks.pop(s.ctx)
		if s.ctx.Err() != nil {
			return
		}

		if err := s.processBlockData(bd); err != nil {
			// depending on the error, we might want to save this block for later
			if !errors.Is(err, errFailedToGetParent) {
				logger.Errorf("block data processing for block with hash %s failed: %s", bd.Hash, err)
				continue
			}

			logger.Tracef("block data processing for block with hash %s failed: %s", bd.Hash, err)
			if err := s.pendingBlocks.addBlock(&types.Block{
				Header: *bd.Header,
				Body:   *bd.Body,
			}); err != nil {
				logger.Debugf("failed to re-add block to pending blocks: %s", err)
			}
		}
	}
}

// processBlockData processes the BlockData from a BlockResponse and
// returns the index of the last BlockData it handled on success,
// or the index of the block data that errored on failure.
func (s *chainProcessor) processBlockData(bd *types.BlockData) error {
	if bd == nil {
		return ErrNilBlockData
	}

	hasHeader, err := s.blockState.HasHeader(bd.Hash)
	if err != nil {
		return fmt.Errorf("failed to check if block state has header for hash %s: %w", bd.Hash, err)
	}
	hasBody, err := s.blockState.HasBlockBody(bd.Hash)
	if err != nil {
		return fmt.Errorf("failed to check block state has body for hash %s: %w", bd.Hash, err)
	}

	if hasHeader && hasBody {
		// TODO: fix this; sometimes when the node shuts down the "best block" isn't stored properly,
		// so when the node restarts it has blocks higher than what it thinks is the best, causing it not to sync
		// if we update the node to only store finalised blocks in the database, this should be fixed and the entire
		// code block can be removed (#1784)
		block, err := s.blockState.GetBlockByHash(bd.Hash)
		if err != nil {
			logger.Debugf("failed to get block header for hash %s: %s", bd.Hash, err)
			return err
		}

		logger.Debugf(
			"skipping block number %d with hash %s, already have",
			block.Header.Number, bd.Hash) // TODO is this valid?

		err = s.blockState.AddBlockToBlockTree(block)
		if errors.Is(err, blocktree.ErrBlockExists) {
			return nil
		} else if err != nil {
			logger.Warnf("failed to add block with hash %s to blocktree: %s", bd.Hash, err)
			return err
		}

		if bd.Justification != nil {
			logger.Debugf("handling Justification for block number %d with hash %s...", block.Header.Number, bd.Hash)
			err = s.handleJustification(&block.Header, *bd.Justification)
			if err != nil {
				return fmt.Errorf("handling justification: %w", err)
			}
		}

		// TODO: this is probably unnecessary, since the state is already in the database
		// however, this case shouldn't be hit often, since it's only hit if the node state
		// is rewinded or if the node shuts down unexpectedly (#1784)
		state, err := s.storageState.TrieState(&block.Header.StateRoot)
		if err != nil {
			logger.Warnf("failed to load state for block with hash %s: %s", block.Header.Hash(), err)
			return err
		}

		if err := s.blockImportHandler.HandleBlockImport(block, state); err != nil {
			logger.Warnf("failed to handle block import: %s", err)
		}

		return nil
	}

	logger.Debugf("processing block data with hash %s", bd.Hash)

	if bd.Header != nil && bd.Body != nil {
		if err := s.babeVerifier.VerifyBlock(bd.Header); err != nil {
			return err
		}

		s.handleBody(bd.Body)

		block := &types.Block{
			Header: *bd.Header,
			Body:   *bd.Body,
		}

		if err := s.handleBlock(block); err != nil {
			logger.Debugf("failed to handle block number %d: %s", block.Header.Number, err)
			return err
		}

		logger.Debugf("block with hash %s processed", bd.Hash)
	}

	if bd.Justification != nil && bd.Header != nil {
		logger.Debugf("handling Justification for block number %d with hash %s...", bd.Number(), bd.Hash)
		err = s.handleJustification(bd.Header, *bd.Justification)
		if err != nil {
			return fmt.Errorf("handling justification: %w", err)
		}
	}

	if err := s.blockState.CompareAndSetBlockData(bd); err != nil {
		return fmt.Errorf("failed to compare and set data: %w", err)
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

	logger.Debugf("🔗 imported block number %d with hash %s", block.Header.Number, block.Header.Hash())
	// should we announce the block here?

	blockHash := block.Header.Hash()
	s.telemetry.SendMessage(telemetry.NewBlockImport(
		&blockHash,
		block.Header.Number,
		"NetworkInitialSync"))

	return nil
}

func (s *chainProcessor) handleJustification(header *types.Header, justification []byte) (err error) {
	if len(justification) == 0 {
		return nil
	}

	headerHash := header.Hash()
	returnedJustification, err := s.finalityGadget.VerifyBlockJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("verifying block number %d justification: %w", header.Number, err)
	}

	err = s.blockState.SetJustification(headerHash, returnedJustification)
	if err != nil {
		return fmt.Errorf("setting justification for block number %d: %w", header.Number, err)
	}

	logger.Infof("🔨 finalised block number %d with hash %s", header.Number, headerHash)
	return nil
}
