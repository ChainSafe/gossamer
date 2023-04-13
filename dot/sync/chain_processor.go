// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
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
	// for {
	// 	bd, err := s.readyBlocks.pop(s.ctx)
	// 	if err != nil {
	// 		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
	// 			return
	// 		}
	// 		panic(fmt.Sprintf("unhandled error: %s", err))
	// 	}

	// 	if err := s.processBlockData(*bd); err != nil {
	// 		// depending on the error, we might want to save this block for later
	// 		logger.Errorf("block data processing for block with hash %s failed: %s", bd.Hash, err)

	// 		if !errors.Is(err, errFailedToGetParent) && !errors.Is(err, blocktree.ErrParentNotFound) {
	// 			continue
	// 		}

	// 		logger.Tracef("block data processing for block with hash %s failed: %s", bd.Hash, err)
	// 		if err := s.pendingBlocks.addBlock(&types.Block{
	// 			Header: *bd.Header,
	// 			Body:   *bd.Body,
	// 		}); err != nil {
	// 			logger.Debugf("failed to re-add block to pending blocks: %s", err)
	// 		}
	// 	}
	// }
}
