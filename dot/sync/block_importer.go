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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var blockSizeGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "gossamer_sync",
	Name:      "block_size",
	Help:      "represent the size of blocks synced",
})

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
		VerifyBlockJustification(common.Hash, uint, []byte) (round uint64, setID uint64, err error)
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
		blockState:         cfg.BlockState,
		storageState:       cfg.StorageState,
		transactionState:   cfg.TransactionState,
		babeVerifier:       cfg.BabeVerifier,
		finalityGadget:     cfg.FinalityGadget,
		blockImportHandler: cfg.BlockImportHandler,
		telemetry:          cfg.Telemetry,
	}
}

func (b *blockImporter) importBlock(bd *types.BlockData, origin BlockOrigin) (imported bool, err error) {
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
func (b *blockImporter) processBlockData(blockData types.BlockData, origin BlockOrigin) error {
	if blockData.Header != nil {
		header := blockData.Header

		if blockData.Body != nil {
			err := b.processBlockDataWithHeaderAndBody(blockData, origin)
			if err != nil {
				return fmt.Errorf("processing block data with header and body: %w", err)
			}
		}

		if blockData.Justification != nil && len(*blockData.Justification) > 0 {
			round, setID, err := b.finalityGadget.VerifyBlockJustification(
				header.Hash(), header.Number, *blockData.Justification)
			if err != nil {
				return fmt.Errorf("verifying justification for block %s: %w", header.Hash().String(), err)
			}

			err = b.blockState.SetFinalisedHash(header.Hash(), round, setID)
			if err != nil {
				return fmt.Errorf("setting finalised hash: %w", err)
			}

			err = b.blockState.SetJustification(header.Hash(), *blockData.Justification)
			if err != nil {
				return fmt.Errorf("setting justification for block number %d: %w", header.Number, err)
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
	origin BlockOrigin) (err error) {

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

	err = b.handleBlock(block)
	if err != nil {
		return fmt.Errorf("handling block: %w", err)
	}

	return nil
}

// handleBlock executes blocks and writes them to disk
func (b *blockImporter) handleBlock(block *types.Block) error {
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

	root := ts.Trie().MustHash()
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

	announceImportedBlock := false
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
