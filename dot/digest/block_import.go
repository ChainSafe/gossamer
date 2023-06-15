// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type BlockImportHandler struct {
	epochState   EpochState
	grandpaState GrandpaState
}

func NewBlockImportHandler(epochState EpochState, grandpaState GrandpaState) *BlockImportHandler {
	return &BlockImportHandler{
		epochState:   epochState,
		grandpaState: grandpaState,
	}
}

func (h *BlockImportHandler) Handle(importedBlockHeader *types.Header) error {
	err := h.handleDigests(importedBlockHeader)
	if err != nil {
		return fmt.Errorf("while handling digests: %w", err)
	}

	// TODO: move to core handleBlock
	// https://github.com/ChainSafe/gossamer/issues/3330
	err = h.grandpaState.ApplyForcedChanges(importedBlockHeader)
	if err != nil {
		return fmt.Errorf("while apply forced changes: %s", err)
	}

	return nil
}

// HandleDigests handles consensus digests for an imported block
func (h *BlockImportHandler) handleDigests(header *types.Header) error {
	consensusDigests := toConsensusDigests(header.Digest.Types)
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
			logger.Errorf("cannot handle consensus digest: %w", err)
		}
	}

	return nil
}

func (h *BlockImportHandler) handleConsensusDigest(d *types.ConsensusDigest, header *types.Header) error {
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

		return h.epochState.HandleBABEDigest(header, data)
	default:
		return fmt.Errorf("%w: 0x%x", ErrUnknownConsensusEngineID, d.ConsensusEngineID.ToBytes())
	}
}

// toConsensusDigests converts a slice of scale.VaryingDataType to a slice of types.ConsensusDigest.
func toConsensusDigests(scaleVaryingTypes []scale.VaryingDataType) []types.ConsensusDigest {
	consensusDigests := make([]types.ConsensusDigest, 0, len(scaleVaryingTypes))

	for _, d := range scaleVaryingTypes {
		digestValue, err := d.Value()
		if err != nil {
			logger.Error(err.Error())
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
