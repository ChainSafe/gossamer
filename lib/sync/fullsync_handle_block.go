// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
)

type (
	// Telemetry is the telemetry client to send telemetry messages.
	Telemetry interface {
		SendMessage(msg json.Marshaler)
	}

	// StorageState is the interface for the storage state
	StorageState interface {
		TrieState(root *common.Hash) (*rtstorage.TrieState, error)
		sync.Locker
	}

	// TransactionState is the interface for transaction queue methods
	TransactionState interface {
		RemoveExtrinsic(ext types.Extrinsic)
	}

	// BabeVerifier deals with BABE block verification
	BabeVerifier interface {
		VerifyBlock(header *types.Header) error
	}

	// FinalityGadget implements justification verification functionality
	FinalityGadget interface {
		VerifyBlockJustification(common.Hash, []byte) error
	}

	// BlockImportHandler is the interface for the handler of newly imported blocks
	BlockImportHandler interface {
		HandleBlockImport(block *types.Block, state *rtstorage.TrieState, announce bool) error
	}
)

type blockImporter struct {
	blockState         BlockState
	storageState       StorageState
	transactionState   TransactionState
	babeVerifier       BabeVerifier
	finalityGadget     FinalityGadget
	blockImportHandler BlockImportHandler
	telemetry          Telemetry
}

func newBlockImporter(cfg *FullSyncConfig) *blockImporter {
	return &blockImporter{
		storageState:       cfg.StorageState,
		transactionState:   cfg.TransactionState,
		babeVerifier:       cfg.BabeVerifier,
		finalityGadget:     cfg.FinalityGadget,
		blockImportHandler: cfg.BlockImportHandler,
		telemetry:          cfg.Telemetry,
	}
}

func (b *blockImporter) handle(bd *types.BlockData, origin BlockOrigin) (imported bool, err error) {
	blockAlreadyExists, err := b.blockState.HasHeader(bd.Hash)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return false, err
	}

	if blockAlreadyExists {
		return false, nil
	}

	err = b.processBlockData(*bd, origin)
	if err != nil {
		logger.Errorf("processing block #%d (%s) failed: %s", bd.Header.Number, bd.Hash, err)
		return false, err
	}

	return true, nil
}

// processBlockData processes the BlockData from a BlockResponse and
// returns the index of the last BlockData it handled on success,
// or the index of the block data that errored on failure.
// TODO: https://github.com/ChainSafe/gossamer/issues/3468
func (b *blockImporter) processBlockData(blockData types.BlockData, origin BlockOrigin) error {
	// while in bootstrap mode we don't need to broadcast block announcements
	// TODO: set true if not in initial sync setup
	announceImportedBlock := false

	if blockData.Header != nil {
		if blockData.Body != nil {
			err := b.processBlockDataWithHeaderAndBody(blockData, origin, announceImportedBlock)
			if err != nil {
				return fmt.Errorf("processing block data with header and body: %w", err)
			}
		}

		if blockData.Justification != nil && len(*blockData.Justification) > 0 {
			err := b.handleJustification(blockData.Header, *blockData.Justification)
			if err != nil {
				return fmt.Errorf("handling justification: %w", err)
			}
		}
	}

	err := b.blockState.CompareAndSetBlockData(&blockData)
	if err != nil {
		return fmt.Errorf("comparing and setting block data: %w", err)
	}

	return nil
}

func (b *blockImporter) processBlockDataWithHeaderAndBody(blockData types.BlockData,
	origin BlockOrigin, announceImportedBlock bool) (err error) {

	if origin != networkInitialSync {
		err = b.babeVerifier.VerifyBlock(blockData.Header)
		if err != nil {
			return fmt.Errorf("babe verifying block: %w", err)
		}
	}

	accBlockSize := 0
	for _, ext := range *blockData.Body {
		accBlockSize += len(ext)
		b.transactionState.RemoveExtrinsic(ext)
	}

	blockSizeGauge.Set(float64(accBlockSize))

	block := &types.Block{
		Header: *blockData.Header,
		Body:   *blockData.Body,
	}

	err = b.handleBlock(block, announceImportedBlock)
	if err != nil {
		return fmt.Errorf("handling block: %w", err)
	}

	return nil
}

// handleHeader handles blocks (header+body) included in BlockResponses
func (b *blockImporter) handleBlock(block *types.Block, announceImportedBlock bool) error {
	parent, err := b.blockState.GetHeader(block.Header.ParentHash)
	if err != nil {
		return fmt.Errorf("%w: %s", errFailedToGetParent, err)
	}

	b.storageState.Lock()
	defer b.storageState.Unlock()

	ts, err := b.storageState.TrieState(&parent.StateRoot)
	if err != nil {
		return err
	}

	root := ts.MustRoot()
	if !bytes.Equal(parent.StateRoot[:], root[:]) {
		panic("parent state root does not match snapshot state root")
	}

	rt, err := b.blockState.GetRuntime(parent.Hash())
	if err != nil {
		return err
	}

	rt.SetContextStorage(ts)

	_, err = rt.ExecuteBlock(block)
	if err != nil {
		return fmt.Errorf("failed to execute block %d: %w", block.Header.Number, err)
	}

	if err = b.blockImportHandler.HandleBlockImport(block, ts, announceImportedBlock); err != nil {
		return err
	}

	blockHash := block.Header.Hash()
	b.telemetry.SendMessage(telemetry.NewBlockImport(
		&blockHash,
		block.Header.Number,
		"NetworkInitialSync"))

	return nil
}

func (b *blockImporter) handleJustification(header *types.Header, justification []byte) (err error) {
	headerHash := header.Hash()
	err = b.finalityGadget.VerifyBlockJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("verifying block number %d justification: %w", header.Number, err)
	}

	err = b.blockState.SetJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("setting justification for block number %d: %w", header.Number, err)
	}

	return nil
}
