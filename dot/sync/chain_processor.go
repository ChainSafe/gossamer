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

type ChainProcessor interface {
	start()
	stop()
}

type chainProcessor struct {
	ctx    context.Context
	cancel context.CancelFunc

	// blocks that are ready for processing. ie. their parent is known, or their parent is ahead
	// of them within this channel and thus will be processed first
	readyBlocks <-chan *types.BlockData

	blockState         BlockState
	storageState       StorageState
	transactionState   TransactionState
	babeVerifier       BabeVerifier
	finalityGadget     FinalityGadget
	blockImportHandler BlockImportHandler
}

func newChainProcessor(readyBlocks <-chan *types.BlockData, blockState BlockState, storageState StorageState, transactionState TransactionState, babeVerifier BabeVerifier, finalityGadget FinalityGadget, blockImportHandler BlockImportHandler) *chainProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &chainProcessor{
		ctx:                ctx,
		cancel:             cancel,
		readyBlocks:        readyBlocks,
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
		case bd := <-s.readyBlocks:
			err := s.processBlockData(bd)
			if err != nil {
				logger.Crit("ready block failed", "hash", bd.Hash)
				// TODO: we probably want to relay this error to the chainSync module so the block can be retried
			}
		case <-s.ctx.Done():
			return
		}
	}
}

// processBlockData processes the BlockData from a BlockResponse and returns the index of the last BlockData it handled on success,
// or the index of the block data that errored on failure.
func (s *chainProcessor) processBlockData(bd *types.BlockData) error {
	if bd == nil {
		return ErrNilBlockData
	}

	//logger.Debug("starting processing of block", "hash", bd.Hash)

	err := s.blockState.CompareAndSetBlockData(bd)
	if err != nil {
		return fmt.Errorf("failed to compare and set data: %w", err)
	}

	hasHeader, _ := s.blockState.HasHeader(bd.Hash)
	hasBody, _ := s.blockState.HasBlockBody(bd.Hash)
	if hasHeader && hasBody {
		// TODO: fix this; sometimes when the node shuts down the "best block" isn't stored properly,
		// so when the node restarts it has blocks higher than what it thinks is the best, causing it not to sync
		//logger.Debug("skipping block, already have", "hash", bd.Hash)

		block, err := s.blockState.GetBlockByHash(bd.Hash) //nolint
		if err != nil {
			logger.Debug("failed to get header", "hash", bd.Hash, "error", err)
			return err
		}

		logger.Debug("skipping block, already have", "hash", bd.Hash, "number", block.Header.Number)

		err = s.blockState.AddBlockToBlockTree(block.Header)
		if err != nil && !errors.Is(err, blocktree.ErrBlockExists) {
			logger.Warn("failed to add block to blocktree", "hash", bd.Hash, "error", err)
			return err
		}

		if errors.Is(err, blocktree.ErrBlockExists) {
			return nil
		}

		// if bd.Justification != nil && bd.Justification.Exists() {
		// 	logger.Debug("handling Justification...", "number", block.Header.Number, "hash", bd.Hash)
		// 	s.handleJustification(block.Header, bd.Justification.Value())
		// }

		// // TODO: this is probably unnecessary, since the state is already in the database
		// // however, this case shouldn't be hit often, since it's only hit if the node state
		// // is rewinded or if the node shuts down unexpectedly
		// state, err := s.storageState.TrieState(&block.Header.StateRoot)
		// if err != nil {
		// 	logger.Warn("failed to load state for block", "block", block.Header.Hash(), "error", err)
		// 	return err
		// }

		// if err := s.blockImportHandler.HandleBlockImport(block, state); err != nil {
		// 	logger.Warn("failed to handle block import", "error", err)
		// }

		return nil
	}

	var header *types.Header

	if bd.Header.Exists() && !hasHeader {
		header, err = types.NewHeaderFromOptional(bd.Header)
		if err != nil {
			return err
		}

		//logger.Trace("processing header", "hash", header.Hash(), "number", header.Number)

		err = s.handleHeader(header)
		if err != nil {
			return err
		}

		//logger.Trace("header processed", "hash", bd.Hash)
	}

	if bd.Body.Exists() && !hasBody {
		body, err := types.NewBodyFromOptional(bd.Body) //nolint
		if err != nil {
			return err
		}

		//logger.Trace("processing body", "hash", bd.Hash)

		err = s.handleBody(body)
		if err != nil {
			return err
		}

		//logger.Trace("body processed", "hash", bd.Hash)
	}

	if bd.Header.Exists() && bd.Body.Exists() {
		header, err = types.NewHeaderFromOptional(bd.Header)
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

		logger.Debug("processing block", "hash", bd.Hash)

		err = s.handleBlock(block)
		if err != nil {
			logger.Error("failed to handle block", "number", block.Header.Number, "error", err)
			return err
		}

		logger.Debug("block processed", "hash", bd.Hash)
	}

	if bd.Justification != nil && bd.Justification.Exists() && header != nil {
		logger.Debug("handling Justification...", "number", bd.Number(), "hash", bd.Hash)
		s.handleJustification(header, bd.Justification.Value())
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
func (s *chainProcessor) handleBody(body *types.Body) error {
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
func (s *chainProcessor) handleBlock(block *types.Block) error {
	if block == nil || block.Header == nil || block.Body == nil {
		return errors.New("block, header, or body is nil")
	}

	parent, err := s.blockState.GetHeader(block.Header.ParentHash)
	if err != nil {
		return fmt.Errorf("failed to get parent hash: %w", err)
	}

	s.storageState.Lock()
	defer s.storageState.Unlock()

	//logger.Trace("getting parent state", "root", parent.StateRoot)
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
	//logger.Trace("going to execute block", "header", block.Header, "exts", block.Body)

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
