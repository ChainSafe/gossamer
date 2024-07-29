package sync

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
)

func (f *FullSyncStrategy) handleReadyBlock(bd *types.BlockData, origin BlockOrigin) error {
	err := f.processBlockData(*bd, origin)
	if err != nil {
		// depending on the error, we might want to save this block for later
		logger.Errorf("processing block #%d (%s) failed: %s", bd.Header.Number, bd.Hash, err)
		return err
	}

	return nil
}

// processBlockData processes the BlockData from a BlockResponse and
// returns the index of the last BlockData it handled on success,
// or the index of the block data that errored on failure.
// TODO: https://github.com/ChainSafe/gossamer/issues/3468
func (f *FullSyncStrategy) processBlockData(blockData types.BlockData, origin BlockOrigin) error {
	// while in bootstrap mode we don't need to broadcast block announcements
	// TODO: set true if not in initial sync setup
	announceImportedBlock := false

	if blockData.Header != nil {
		if blockData.Body != nil {
			err := f.processBlockDataWithHeaderAndBody(blockData, origin, announceImportedBlock)
			if err != nil {
				return fmt.Errorf("processing block data with header and body: %w", err)
			}
		}

		if blockData.Justification != nil && len(*blockData.Justification) > 0 {
			err := f.handleJustification(blockData.Header, *blockData.Justification)
			if err != nil {
				return fmt.Errorf("handling justification: %w", err)
			}
		}
	}

	err := f.blockState.CompareAndSetBlockData(&blockData)
	if err != nil {
		return fmt.Errorf("comparing and setting block data: %w", err)
	}

	return nil
}

func (f *FullSyncStrategy) processBlockDataWithHeaderAndBody(blockData types.BlockData,
	origin BlockOrigin, announceImportedBlock bool) (err error) {

	if origin != networkInitialSync {
		err = f.babeVerifier.VerifyBlock(blockData.Header)
		if err != nil {
			return fmt.Errorf("babe verifying block: %w", err)
		}
	}

	f.handleBody(blockData.Body)

	block := &types.Block{
		Header: *blockData.Header,
		Body:   *blockData.Body,
	}

	err = f.handleBlock(block, announceImportedBlock)
	if err != nil {
		return fmt.Errorf("handling block: %w", err)
	}

	return nil
}

// handleHeader handles block bodies included in BlockResponses
func (f *FullSyncStrategy) handleBody(body *types.Body) {
	acc := 0
	for _, ext := range *body {
		acc += len(ext)
		f.transactionState.RemoveExtrinsic(ext)
	}

	blockSizeGauge.Set(float64(acc))
}

// handleHeader handles blocks (header+body) included in BlockResponses
func (f *FullSyncStrategy) handleBlock(block *types.Block, announceImportedBlock bool) error {
	parent, err := f.blockState.GetHeader(block.Header.ParentHash)
	if err != nil {
		return fmt.Errorf("%w: %s", errFailedToGetParent, err)
	}

	f.storageState.Lock()
	defer f.storageState.Unlock()

	ts, err := f.storageState.TrieState(&parent.StateRoot)
	if err != nil {
		return err
	}

	root := ts.MustRoot()
	if !bytes.Equal(parent.StateRoot[:], root[:]) {
		panic("parent state root does not match snapshot state root")
	}

	rt, err := f.blockState.GetRuntime(parent.Hash())
	if err != nil {
		return err
	}

	rt.SetContextStorage(ts)

	_, err = rt.ExecuteBlock(block)
	if err != nil {
		return fmt.Errorf("failed to execute block %d: %w", block.Header.Number, err)
	}

	if err = f.blockImportHandler.HandleBlockImport(block, ts, announceImportedBlock); err != nil {
		return err
	}

	blockHash := block.Header.Hash()
	f.telemetry.SendMessage(telemetry.NewBlockImport(
		&blockHash,
		block.Header.Number,
		"NetworkInitialSync"))

	return nil
}

func (f *FullSyncStrategy) handleJustification(header *types.Header, justification []byte) (err error) {
	headerHash := header.Hash()
	err = f.finalityGadget.VerifyBlockJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("verifying block number %d justification: %w", header.Number, err)
	}

	err = f.blockState.SetJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("setting justification for block number %d: %w", header.Number, err)
	}

	return nil
}
