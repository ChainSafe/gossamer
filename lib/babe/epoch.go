// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

type EpochDescriptor struct {
	data      *epochData
	epoch     uint64
	startSlot uint64
	endSlot   uint64
}

// initiateEpoch sets the epochData for the given epoch, runs the lottery for the slots in the epoch,
// and stores updated EpochInfo in the database
func (b *Service) initiateEpoch(epoch uint64) (*EpochDescriptor, error) {
	logger.Debugf("initiating epoch %d", epoch, b.constants.epochLength)

	bestBlockHeader, err := b.blockState.BestBlockHeader()
	if err != nil {
		return nil, fmt.Errorf("cannot get the best block header: %w", err)
	}

	skipped, diff, err := b.checkIfEpochSkipped(epoch, bestBlockHeader)
	if err != nil {
		return nil, fmt.Errorf("checking if epoch skipped: %w", err)
	}

	epochToFindData := epoch
	if skipped {
		logger.Warnf("⏩ A total of %d epochs were skipped, from %d to %d",
			diff, epoch-diff, epoch)
		epochToFindData = epoch - diff
	}

	epochData, err := b.getEpochData(epochToFindData, bestBlockHeader)
	if err != nil {
		return nil, fmt.Errorf("cannot get epoch data and start slot: %w", err)
	}

	// if we're at genesis or epoch was skipped then we can estimate when the start
	// slot of the epoch will be, the estimation is used to calculate the epoch end
	// TODO: check how substrate deals with these estimation
	if bestBlockHeader.Hash() == b.blockState.GenesisHash() || skipped {
		startSlot, err := b.getFirstAuthoringSlot(epoch, epochData)
		if err != nil {
			return nil, fmt.Errorf("cannot get first authoring slot: %w", err)
		}

		logger.Debugf("estimated first slot as %d for epoch %d", startSlot, epoch)
		return &EpochDescriptor{
			data:      epochData,
			epoch:     epoch,
			startSlot: startSlot,
			endSlot:   startSlot + b.constants.epochLength,
		}, nil
	}

	startSlot, err := b.epochState.GetStartSlotForEpoch(epoch, bestBlockHeader.Hash())
	if err != nil {
		return nil, fmt.Errorf("cannot get start slot for epoch %d: %w", epoch, err)
	}

	logger.Infof("initiating epoch %d with start slot %d", epoch, startSlot)
	return &EpochDescriptor{
		data:      epochData,
		epoch:     epoch,
		startSlot: startSlot,
		endSlot:   startSlot + b.constants.epochLength,
	}, nil
}

var ErrEpochLowerThanExpected = errors.New("epoch lower than expected")

func (b *Service) checkIfEpochSkipped(epochBeingInitialized uint64, bestBlock *types.Header) (
	skipped bool, diff uint64, err error) {
	if epochBeingInitialized == 0 {
		return false, 0, nil
	}

	epochFromBestBlock, err := b.epochState.GetEpochForBlock(bestBlock)
	if err != nil {
		return false, 0, fmt.Errorf("getting epoch for block: %w", err)
	}

	if epochBeingInitialized < epochFromBestBlock {
		return false, 0, fmt.Errorf("%w: expected %d, got: %d",
			ErrEpochLowerThanExpected, epochBeingInitialized, epochFromBestBlock)
	}

	if epochFromBestBlock+1 == epochBeingInitialized {
		return false, 0, nil
	}

	return epochBeingInitialized > epochFromBestBlock, epochBeingInitialized - epochFromBestBlock, nil
}

func (b *Service) getEpochData(epoch uint64, bestBlock *types.Header) (*epochData, error) {
	currEpochData, err := b.epochState.GetEpochDataRaw(epoch, bestBlock)
	if err != nil {
		return nil, fmt.Errorf("getting epoch data for epoch %d: %w", epoch, err)
	}

	currConfigData, err := b.epochState.GetConfigData(epoch, bestBlock)
	if err != nil {
		return nil, fmt.Errorf("getting config data for epoch %d: %w", epoch, err)
	}

	threshold, err := CalculateThreshold(currConfigData.C1, currConfigData.C2, len(currEpochData.Authorities))
	if err != nil {
		return nil, fmt.Errorf("calculating threshold: %w", err)
	}

	idx, err := b.getAuthorityIndex(currEpochData.Authorities)
	if err != nil {
		return nil, fmt.Errorf("getting authority index: %w", err)
	}

	return &epochData{
		randomness:     currEpochData.Randomness,
		authorities:    currEpochData.Authorities,
		authorityIndex: idx,
		threshold:      threshold,
		allowedSlots:   types.AllowedSlots(currConfigData.SecondarySlots),
	}, nil
}

func (b *Service) getFirstAuthoringSlot(epoch uint64, epochData *epochData) (uint64, error) {
	startSlot := getCurrentSlot(b.constants.slotDuration)
	for i := startSlot; i < startSlot+b.constants.epochLength; i++ {
		_, err := claimSlot(epoch, i, epochData, b.keypair)
		if errors.Is(err, errOverPrimarySlotThreshold) || errors.Is(err, errNotOurTurnToPropose) {
			continue
		} else if err != nil {
			return 0, fmt.Errorf("error running slot lottery at slot %d: error %w", i, err)
		}

		startSlot = i
		break
	}

	return startSlot, nil
}

// incrementEpoch increments the current epoch stored in the db and returns the new epoch number
func (b *Service) incrementEpoch() (uint64, error) {
	epoch, err := b.epochState.GetCurrentEpoch()
	if err != nil {
		return 0, err
	}

	next := epoch + 1
	err = b.epochState.StoreCurrentEpoch(next)
	if err != nil {
		return 0, err
	}

	return next, nil
}

// claimSlot attempts to claim a slot for a specific slot number.
// It returns an encoded VrfOutputAndProof if the validator is authorised
// to produce a block for that slot.
// It returns the wrapped error errOverPrimarySlotThreshold
// if it is not authorised.
// output = return[0:32]; proof = return[32:96]
func claimSlot(epochNumber uint64, slotNumber uint64, epochData *epochData, keypair *sr25519.Keypair,
) (*types.PreRuntimeDigest, error) {
	proof, err := claimPrimarySlot(
		epochData.randomness,
		slotNumber,
		epochNumber,
		epochData.threshold,
		keypair,
	)

	if err == nil {
		babePrimaryPreDigest := types.NewBabePrimaryPreDigest(epochData.authorityIndex, slotNumber, proof.output, proof.proof)
		preRuntimeDigest, err := babePrimaryPreDigest.ToPreRuntimeDigest()
		if err != nil {
			return nil, fmt.Errorf("error converting babe primary pre-digest to pre-runtime digest: %w", err)
		}
		logger.Debugf("epoch %d: claimed primary slot %d", epochNumber, slotNumber)
		return preRuntimeDigest, nil
	} else if !errors.Is(err, errOverPrimarySlotThreshold) {
		return nil, fmt.Errorf("error running slot lottery at slot %d: %w", slotNumber, err)
	}

	switch epochData.allowedSlots {
	case types.PrimarySlots:
		return nil, errNotOurTurnToPropose
	case types.PrimaryAndSecondaryVRFSlots:
		proof, err := claimSecondarySlotVRF(
			epochData.randomness, slotNumber, epochNumber, epochData.authorities, keypair, epochData.authorityIndex)
		if err != nil {
			return nil, fmt.Errorf("cannot claim secondary vrf slot at %d: %w", slotNumber, err)
		}
		babeSecondaryVRFPreDigest := types.NewBabeSecondaryVRFPreDigest(
			epochData.authorityIndex, slotNumber, proof.output, proof.proof)
		preRuntimeDigest, err := babeSecondaryVRFPreDigest.ToPreRuntimeDigest()
		if err != nil {
			return nil, fmt.Errorf("error converting babe secondary vrf pre-digest to pre-runtime digest: %w", err)
		}

		logger.Debugf("epoch %d: claimed secondary vrf slot %d", epochNumber, slotNumber)
		return preRuntimeDigest, nil
	case types.PrimaryAndSecondaryPlainSlots:
		err = claimSecondarySlotPlain(
			epochData.randomness, slotNumber, epochData.authorities, epochData.authorityIndex)
		if err != nil {
			return nil, fmt.Errorf("cannot claim secondary plain slot at %d: %w", slotNumber, err)
		}

		preRuntimeDigest, err := types.NewBabeSecondaryPlainPreDigest(
			epochData.authorityIndex, slotNumber).ToPreRuntimeDigest()
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get preruntime digest from babe secondary plain predigest for slot %d: %w", slotNumber, err)
		}

		logger.Debugf("epoch %d: claimed secondary plain slot %d", epochNumber, slotNumber)
		return preRuntimeDigest, nil
	default:
		// this should never occur
		return nil, errInvalidSlotTechnique
	}
}
