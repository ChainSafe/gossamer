// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	_ services.Service = &Handler{}
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

	// GRANDPA changes
	grandpaScheduledChange *grandpaChange
	grandpaForcedChange    *grandpaChange
	grandpaPause           *pause
	grandpaResume          *resume

	logger log.LeveledLogger
}

type grandpaChange struct {
	auths   []types.Authority
	atBlock uint
}

type pause struct {
	atBlock uint
}

type resume struct {
	atBlock uint
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

		return h.grandpaState.AddPendingChange(header, data)
	case types.BabeEngineID:
		data := types.NewBabeConsensusDigest()
		err := scale.Unmarshal(d.Data, &data)
		if err != nil {
			return err
		}

		return h.handleBabeConsensusDigest(data, header)
	}

	return errors.New("unknown consensus engine ID")
}

func (h *Handler) handleGrandpaConsensusDigest(digest scale.VaryingDataType, header *types.Header) error {
	return h.grandpaState.AddPendingChange(header, digest)
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

			err := h.handleGrandpaChangesOnImport(block.Header.Number)
			if err != nil {
				h.logger.Errorf("failed to handle grandpa changes on block import: %s", err)
			}
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

			err := h.persistBABEDigestsForNextEpoch(&info.Header)
			if err != nil {
				h.logger.Errorf("failed to store babe next epoch digest: %s", err)
			}

			err = h.handleGrandpaChangesOnFinalization(info.Header.Number)
			if err != nil {
				h.logger.Errorf("failed to handle grandpa changes on block finalisation: %s", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// persistBABEDigestsForNextEpoch is called only when a block is finalised
// and defines the correct next epoch data and next config data.
func (h *Handler) persistBABEDigestsForNextEpoch(finalizedHeader *types.Header) error {
	currEpoch, err := h.epochState.GetEpochForBlock(finalizedHeader)
	if err != nil {
		return fmt.Errorf("cannot get epoch for block %d (%s): %w",
			finalizedHeader.Number, finalizedHeader.Hash(), err)
	}

	nextEpoch := currEpoch + 1

	appliedEpochData, appliedConfigData, err := h.epochState.AlreadyDefined(nextEpoch)
	if err != nil {
		return fmt.Errorf("cannot check if next epoch is already defined: %w", err)
	}

	if !appliedEpochData {
		err = h.epochState.FinalizeBABENextEpochData(nextEpoch)
		if err != nil && !errors.Is(err, state.ErrEpochNotInMemory) {
			return fmt.Errorf("cannot finalize babe next epoch data for block number %d (%s): %w",
				finalizedHeader.Number, finalizedHeader.Hash(), err)
		}
	}

	if !appliedConfigData {
		err = h.epochState.FinalizeBABENextConfigData(nextEpoch)
		if err != nil && !errors.Is(err, state.ErrEpochNotInMemory) {
			return fmt.Errorf("cannot finalize babe next config data for block number %d (%s): %w",
				finalizedHeader.Number, finalizedHeader.Hash(), err)
		}
	}

	return nil
}

func (h *Handler) handleGrandpaChangesOnImport(num uint) error {
	resume := h.grandpaResume
	if resume != nil && num >= resume.atBlock {
		h.grandpaResume = nil
	}

	fc := h.grandpaForcedChange
	if fc != nil && num >= fc.atBlock {
		curr, err := h.grandpaState.IncrementSetID()
		if err != nil {
			return err
		}

		h.grandpaForcedChange = nil
		h.logger.Debugf("incremented grandpa set id %d", curr)
	}

	return nil
}

func (h *Handler) handleGrandpaChangesOnFinalization(num uint) error {
	pause := h.grandpaPause
	if pause != nil && num >= pause.atBlock {
		h.grandpaPause = nil
	}

	sc := h.grandpaScheduledChange
	if sc != nil && num >= sc.atBlock {
		curr, err := h.grandpaState.IncrementSetID()
		if err != nil {
			return err
		}

		h.grandpaScheduledChange = nil
		h.logger.Debugf("incremented grandpa set id %d", curr)
	}

	// if blocks get finalised before forced change takes place, disregard it
	h.grandpaForcedChange = nil
	return nil
}

func (h *Handler) handlePause(p types.GrandpaPause) error {
	curr, err := h.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	h.grandpaPause = &pause{
		atBlock: curr.Number + uint(p.Delay),
	}

	return h.grandpaState.SetNextPause(h.grandpaPause.atBlock)
}

func (h *Handler) handleResume(r types.GrandpaResume) error {
	curr, err := h.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	h.grandpaResume = &resume{
		atBlock: curr.Number + uint(r.Delay),
	}

	return h.grandpaState.SetNextResume(h.grandpaResume.atBlock)
}
