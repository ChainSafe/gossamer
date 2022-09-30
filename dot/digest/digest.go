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
	ErrUnknownConsensusEngineID = errors.New("unknown consensus engine ID")
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
func (h *Handler) HandleDigests(header *types.Header) error {
	consensusDigests := h.toConsensusDigests(header.Digest.Types)
	consensusDigests, err := checkForGRANDPAForcedChanges(consensusDigests)
	if err != nil {
		return fmt.Errorf("failed while checking GRANDPA digests: %w", err)
	}

	for i := range consensusDigests {
		// avoiding implicit memory aliasing in for loop, since:
		// for _, digest := range consensusDigests { &digest }
		// is using the address of a loop variable
		digest := consensusDigests[i]
		err := h.handleConsensusDigest(&digest, header)
		if err != nil {
			h.logger.Errorf("cannot handle consensus digest: %w", err)
		}
	}

	return nil
}

// toConsensusDigests converts a slice of scale.VaryingDataType to a slice of types.ConsensusDigest.
func (h *Handler) toConsensusDigests(scaleVaryingTypes []scale.VaryingDataType) []types.ConsensusDigest {
	consensusDigests := make([]types.ConsensusDigest, 0, len(scaleVaryingTypes))

	for _, d := range scaleVaryingTypes {
		digestValue, err := d.Value()
		if err != nil {
			h.logger.Error(err.Error())
			continue
		}
		digest, ok := digestValue.(types.ConsensusDigest)
		if !ok {
			continue
		}

		switch digest.ConsensusEngineID {
		case types.GrandpaEngineID, types.BabeEngineID:
			consensusDigests = append(consensusDigests, digest)
		}
	}

	return consensusDigests
}

// checkForGRANDPAForcedChanges removes any GrandpaScheduledChange in the presence of a
// GrandpaForcedChange in the same block digest, returning a new slice of types.ConsensusDigest
func checkForGRANDPAForcedChanges(digests []types.ConsensusDigest) ([]types.ConsensusDigest, error) {
	var hasForcedChange bool
	digestsWithoutScheduled := make([]types.ConsensusDigest, 0, len(digests))
	for i, digest := range digests {
		if digest.ConsensusEngineID != types.GrandpaEngineID {
			digestsWithoutScheduled = append(digestsWithoutScheduled, digest)
			continue
		}

		data := types.NewGrandpaConsensusDigest()
		err := scale.Unmarshal(digest.Data, &data)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal GRANDPA consensus digest: %w", err)
		}

		dataValue, err := data.Value()
		if err != nil {
			return nil, fmt.Errorf("getting value of digest type at index %d: %w", i, err)
		}
		switch dataValue.(type) {
		case types.GrandpaScheduledChange:
		case types.GrandpaForcedChange:
			hasForcedChange = true
			digestsWithoutScheduled = append(digestsWithoutScheduled, digest)
		default:
			digestsWithoutScheduled = append(digestsWithoutScheduled, digest)
		}
	}

	if hasForcedChange {
		return digestsWithoutScheduled, nil
	}

	return digests, nil
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
		return fmt.Errorf("%w: 0x%x", ErrUnknownConsensusEngineID, d.ConsensusEngineID.ToBytes())
	}
}

func (h *Handler) handleBabeConsensusDigest(digest scale.VaryingDataType, header *types.Header) error {
	headerHash := header.Hash()

	digestValue, err := digest.Value()
	if err != nil {
		return fmt.Errorf("getting digest value: %w", err)
	}
	switch val := digestValue.(type) {
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

			err := h.HandleDigests(&block.Header)
			if err != nil {
				h.logger.Errorf("failed to handle digests: %s", err)
			}

			err = h.grandpaState.ApplyForcedChanges(&block.Header)
			if err != nil {
				h.logger.Errorf("failed to apply forced changes: %s", err)
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
				h.logger.Errorf("failed to apply scheduled change: %s", err)
			}

		case <-ctx.Done():
			return
		}
	}
}
