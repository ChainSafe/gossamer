// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	_ services.Service = &Handler{}
)

var (
	ErrUnknownConsensusID = errors.New("unknown consensus engine ID")
)

// Handler is used to handle consensus messages and relevant authority updates to BABE and GRANDPA
type Handler struct {
	ctx    context.Context
	cancel context.CancelFunc

	// interfaces
	blockState   BlockState
	epochState   EpochState
	grandpaState GrandpaState

	// block notification channels
	imported  chan *types.Block
	finalised chan *types.FinalisationInfo

	logger log.LeveledLogger
}

// NewHandler returns a new Handler
func NewHandler(lvl log.Level, blockState BlockState, epochState EpochState,
	grandpaState GrandpaState) (*Handler, error) {
	imported := blockState.GetImportedBlockNotifierChannel()
	finalised := blockState.GetFinalisedNotifierChannel()

	logger := log.NewFromGlobal(log.AddContext("pkg", "digest"))
	logger.Patch(log.SetLevel(lvl))

	ctx, cancel := context.WithCancel(context.Background())
	return &Handler{
		ctx:          ctx,
		cancel:       cancel,
		blockState:   blockState,
		epochState:   epochState,
		grandpaState: grandpaState,
		imported:     imported,
		finalised:    finalised,
		logger:       logger,
	}, nil
}

// Start starts the Handler
func (h *Handler) Start() error {
	go h.handleBlockImport(h.ctx)
	go h.handleBlockFinalisation(h.ctx)
	return nil
}

// Stop stops the Handler
func (h *Handler) Stop() error {
	h.cancel()
	h.blockState.FreeImportedBlockNotifierChannel(h.imported)
	h.blockState.FreeFinalisedNotifierChannel(h.finalised)
	return nil
}

// HandleDigests handles consensus digests for an imported block
func (h *Handler) HandleDigests(header *types.Header) {
	for i, d := range header.Digest.Types {
		val, ok := d.Value().(types.ConsensusDigest)
		if !ok {
			continue
		}

		err := h.handleConsensusDigest(&val, header)
		if err != nil {
			h.logger.Errorf("cannot handle digest for block number %d, index %d, digest %s: %s",
				header.Number, i, d.Value(), err)
		}
	}
}

func (h *Handler) handleConsensusDigest(d *types.ConsensusDigest, header *types.Header) error {
	switch d.ConsensusEngineID {
	case types.GrandpaEngineID:
		data := types.NewGrandpaConsensusDigest()
		err := scale.Unmarshal(d.Data, &data)
		if err != nil {
			return err
		}

		return h.grandpaState.HandleGRANDPADigest(header, data)
	case types.BabeEngineID:
		data := types.NewBabeConsensusDigest()
		err := scale.Unmarshal(d.Data, &data)
		if err != nil {
			return err
		}

		return h.handleBabeConsensusDigest(data, header)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownConsensusID, d.ConsensusEngineID.ToBytes())
	}
}

func (h *Handler) handleBabeConsensusDigest(digest scale.VaryingDataType, header *types.Header) error {
	headerHash := header.Hash()

	switch val := digest.Value().(type) {
	case types.NextEpochData:
		currEpoch, err := h.epochState.GetEpochForBlock(header)
		if err != nil {
			return fmt.Errorf("cannot get epoch for block %d (%s): %w",
				header.Number, headerHash, err)
		}

		nextEpoch := currEpoch + 1
		h.epochState.StoreBABENextEpochData(nextEpoch, headerHash, val)
		h.logger.Debugf("stored BABENextEpochData data: %v for hash: %s to epoch: %d", digest, headerHash, nextEpoch)
		return nil

	case types.BABEOnDisabled:
		h.logger.Debug("handling BABEOnDisabled")
		return nil

	case types.NextConfigData:
		currEpoch, err := h.epochState.GetEpochForBlock(header)
		if err != nil {
			return fmt.Errorf("cannot get epoch for block %d (%s): %w",
				header.Number, headerHash, err)
		}

		nextEpoch := currEpoch + 1
		h.epochState.StoreBABENextConfigData(nextEpoch, headerHash, val)
		h.logger.Debugf("stored BABENextConfigData data: %v for hash: %s to epoch: %d", digest, headerHash, nextEpoch)
		return nil
	}

	return errors.New("invalid consensus digest data")
}

func (h *Handler) handleBlockImport(ctx context.Context) {
	for {
		select {
		case block := <-h.imported:
			if block == nil {
				continue
			}

			h.HandleDigests(&block.Header)

			h.grandpaState.ApplyForcedChanges(&block.Header)
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) handleBlockFinalisation(ctx context.Context) {
	for {
		select {
		case info := <-h.finalised:
			if info == nil {
				continue
			}

			err := h.epochState.FinalizeBABENextEpochData(&info.Header)
			if err != nil {
				h.logger.Errorf("failed to persist babe next epoch data: %s", err)
			}

			err = h.epochState.FinalizeBABENextConfigData(&info.Header)
			if err != nil {
				h.logger.Errorf("failed to persist babe next epoch config: %s", err)
			}

			err = h.grandpaState.ApplyScheduledChanges(&info.Header)
			if err != nil {
				h.logger.Errorf("failed to apply standard scheduled changes on block finalization: %w", err)
			}

		case <-ctx.Done():
			return
		}
	}
}
