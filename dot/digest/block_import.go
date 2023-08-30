// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type BlockImportHandler struct {
	epochState EpochState
}

func NewBlockImportHandler(epochState EpochState) *BlockImportHandler {
	return &BlockImportHandler{
		epochState: epochState,
	}
}

// HandleDigests handles consensus digests for an imported block
func (h *BlockImportHandler) HandleDigests(header *types.Header) error {
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
			return fmt.Errorf("consensus digests: %w", err)
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
			return fmt.Errorf("unmarshaling grandpa consensus digest: %w", err)
		}
	case types.BabeEngineID:
		data := types.NewBabeConsensusDigest()
		err := scale.Unmarshal(d.Data, &data)
		if err != nil {
			return fmt.Errorf("unmarshaling babe consensus digest: %w", err)
		}

		err = h.epochState.HandleBABEDigest(header, data)
		if err != nil {
			return fmt.Errorf("handling babe digest: %w", err)
		}
	default:
		return fmt.Errorf("%w: 0x%x", ErrUnknownConsensusEngineID, d.ConsensusEngineID.ToBytes())
	}

	return nil
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
