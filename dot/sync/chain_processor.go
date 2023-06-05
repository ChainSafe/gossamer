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

	chainSync ChainSync

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
	telemetry          Telemetry
}

type chainProcessorConfig struct {
	readyBlocks        *blockQueue
	pendingBlocks      DisjointBlockSet
	syncer             ChainSync
	blockState         BlockState
	storageState       StorageState
	transactionState   TransactionState
	babeVerifier       BabeVerifier
	finalityGadget     FinalityGadget
	blockImportHandler BlockImportHandler
	telemetry          Telemetry
	badBlocks          []string
}

func newChainProcessor(cfg chainProcessorConfig) *chainProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &chainProcessor{
		ctx:                ctx,
		cancel:             cancel,
		readyBlocks:        cfg.readyBlocks,
		pendingBlocks:      cfg.pendingBlocks,
		chainSync:          cfg.syncer,
		blockState:         cfg.blockState,
		storageState:       cfg.storageState,
		transactionState:   cfg.transactionState,
		babeVerifier:       cfg.babeVerifier,
		finalityGadget:     cfg.finalityGadget,
		blockImportHandler: cfg.blockImportHandler,
		telemetry:          cfg.telemetry,
	}
}

func (s *chainProcessor) stop() {
	s.cancel()
}

func (s *chainProcessor) processReadyBlocks() {
	for {
		bd, err := s.readyBlocks.pop(s.ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			panic(fmt.Sprintf("unhandled error: %s", err))
		}

		if err := s.processBlockData(*bd); err != nil {
			// depending on the error, we might want to save this block for later
			if !errors.Is(err, errFailedToGetParent) && !errors.Is(err, blocktree.ErrParentNotFound) {
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
func (c *chainProcessor) processBlockData(blockData types.BlockData) error {
	logger.Debugf("processing block data with hash %s", blockData.Hash)

	headerInState, err := c.blockState.HasHeader(blockData.Hash)
	if err != nil {
		return fmt.Errorf("checking if block state has header: %w", err)
	}

	bodyInState, err := c.blockState.HasBlockBody(blockData.Hash)
	if err != nil {
		return fmt.Errorf("checking if block state has body: %w", err)
	}

	// while in bootstrap mode we don't need to broadcast block announcements
	announceImportedBlock := c.chainSync.syncState() == tip
	if headerInState && bodyInState {
		err = c.processBlockDataWithStateHeaderAndBody(blockData, announceImportedBlock)
		if err != nil {
			return fmt.Errorf("processing block data with header and "+
				"body in block state: %w", err)
		}
		return nil
	}

	if blockData.Header != nil {
		if blockData.Body != nil {
			err = c.processBlockDataWithHeaderAndBody(blockData, announceImportedBlock)
			if err != nil {
				return fmt.Errorf("processing block data with header and body: %w", err)
			}
			logger.Debugf("block with hash %s processed", blockData.Hash)
		}

		if blockData.Justification != nil && len(*blockData.Justification) > 0 {
			err = c.handleJustification(blockData.Header, *blockData.Justification)
			if err != nil {
				return fmt.Errorf("handling justification: %w", err)
			}
		}
	}

	err = c.blockState.CompareAndSetBlockData(&blockData)
	if err != nil {
		return fmt.Errorf("comparing and setting block data: %w", err)
	}

	return nil
}

func (c *chainProcessor) processBlockDataWithStateHeaderAndBody(blockData types.BlockData,
	announceImportedBlock bool) (err error) {
	// TODO: fix this; sometimes when the node shuts down the "best block" isn't stored properly,
	// so when the node restarts it has blocks higher than what it thinks is the best, causing it not to sync
	// if we update the node to only store finalised blocks in the database, this should be fixed and the entire
	// code block can be removed (#1784)
	block, err := c.blockState.GetBlockByHash(blockData.Hash)
	if err != nil {
		return fmt.Errorf("getting block by hash: %w", err)
	}

	err = c.blockState.AddBlockToBlockTree(block)
	if errors.Is(err, blocktree.ErrBlockExists) {
		logger.Debugf(
			"block number %d with hash %s already exists in block tree, skipping it.",
			block.Header.Number, blockData.Hash)
		return nil
	} else if err != nil {
		return fmt.Errorf("adding block to blocktree: %w", err)
	}

	if blockData.Justification != nil && len(*blockData.Justification) > 0 {
		err = c.handleJustification(&block.Header, *blockData.Justification)
		if err != nil {
			return fmt.Errorf("handling justification: %w", err)
		}
	}

	// TODO: this is probably unnecessary, since the state is already in the database
	// however, this case shouldn't be hit often, since it's only hit if the node state
	// is rewinded or if the node shuts down unexpectedly (#1784)
	state, err := c.storageState.TrieState(&block.Header.StateRoot)
	if err != nil {
		return fmt.Errorf("loading trie state: %w", err)
	}

	err = c.blockImportHandler.HandleBlockImport(block, state, announceImportedBlock)
	if err != nil {
		return fmt.Errorf("handling block import: %w", err)
	}

	return nil
}

func (c *chainProcessor) processBlockDataWithHeaderAndBody(blockData types.BlockData,
	announceImportedBlock bool) (err error) {
	err = c.babeVerifier.VerifyBlock(blockData.Header)
	if err != nil {
		return fmt.Errorf("babe verifying block: %w", err)
	}

	c.handleBody(blockData.Body)

	block := &types.Block{
		Header: *blockData.Header,
		Body:   *blockData.Body,
	}

	err = c.handleBlock(block, announceImportedBlock)
	if err != nil {
		return fmt.Errorf("handling block: %w", err)
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
func (s *chainProcessor) handleBlock(block *types.Block, announceImportedBlock bool) error {
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

	rt, err := s.blockState.GetRuntime(parent.Hash())
	if err != nil {
		return err
	}

	rt.SetContextStorage(ts)

	_, err = rt.ExecuteBlock(block)
	if err != nil {
		return fmt.Errorf("failed to execute block %d: %w", block.Header.Number, err)
	}

	if err = s.blockImportHandler.HandleBlockImport(block, ts, announceImportedBlock); err != nil {
		return err
	}

	logger.Debugf("🔗 imported block number %d with hash %s", block.Header.Number, block.Header.Hash())

	blockHash := block.Header.Hash()
	s.telemetry.SendMessage(telemetry.NewBlockImport(
		&blockHash,
		block.Header.Number,
		"NetworkInitialSync"))

	return nil
}

func (s *chainProcessor) handleJustification(header *types.Header, justification []byte) (err error) {
	logger.Debugf("handling justification for block %d...", header.Number)

	headerHash := header.Hash()
	err = s.finalityGadget.VerifyBlockJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("verifying block number %d justification: %w", header.Number, err)
	}

	err = s.blockState.SetJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("setting justification for block number %d: %w", header.Number, err)
	}

	logger.Infof("🔨 finalised block number %d with hash %s", header.Number, headerHash)
	return nil
}
