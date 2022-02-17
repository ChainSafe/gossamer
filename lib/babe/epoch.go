// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
)

// initiateEpoch sets the epochData for the given epoch, runs the lottery for the slots in the epoch,
// and stores updated EpochInfo in the database
func (b *Service) initiateEpoch(epoch uint64) (*epochData, error) {
	logger.Debugf("initiating epoch %d", epoch)

	// if epoch == 1, check that first slot is still set correctly
	// ie. that the start slot of the network is the same as the slot number of block 1
	if epoch == 1 {
		if err := b.checkAndSetFirstSlot(); err != nil {
			return nil, fmt.Errorf("cannot check and set first slot: %w", err)
		}
	}

	epochData, startSlot, err := b.getEpochDataAndStartSlot(epoch)
	if err != nil {
		return nil, fmt.Errorf("cannot get epoch data and start slot: %w", err)
	}

	// if we're at genesis, we need to determine when the first slot of the network will be
	// by checking when we will be able to produce block 1.
	// note that this assumes there will only be one producer of block 1
	if b.blockState.BestBlockHash() == b.blockState.GenesisHash() {
		startSlot, err = b.getFirstAuthoringSlot(epoch, epochData)
		if err != nil {
			return nil, fmt.Errorf("cannot get first authoring slot: %w", err)
		}

		logger.Debugf("estimated first slot as %d based on building block 1", startSlot)

		// we are at genesis, set first slot by checking at which slot we will be able to produce block 1
		if err = b.epochState.SetFirstSlot(startSlot); err != nil {
			return nil, fmt.Errorf("cannot set first slot: %w", err)
		}
	}

	logger.Infof("initiating epoch %d with start slot %d", epoch, startSlot)
	return epochData, nil
}

func (b *Service) checkAndSetFirstSlot() error {
	firstSlot, err := b.epochState.GetStartSlotForEpoch(0)
	if err != nil {
		return fmt.Errorf("cannot set first slot: %w", err)
	}

	block, err := b.blockState.GetBlockByNumber(big.NewInt(1))
	if err != nil {
		return fmt.Errorf("cannot get block with number 1: %w", err)
	}

	slot, err := types.GetSlotFromHeader(&block.Header)
	if err != nil {
		return fmt.Errorf("cannot get slot from header of block 1: %w", err)
	}

	if slot != firstSlot {
		if err := b.epochState.SetFirstSlot(slot); err != nil {
			return fmt.Errorf("cannot set first slot for block 1: %w", err)
		}
	}

	return nil
}

func (b *Service) getEpochDataAndStartSlot(epoch uint64) (*epochData, uint64, error) {
	if epoch == 0 {
		startSlot, err := b.epochState.GetStartSlotForEpoch(epoch)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot get start slot for epoch %d: %w", epoch, err)
		}

		epochData, err := b.getLatestEpochData()
		if err != nil {
			return nil, 0, fmt.Errorf("cannot get latest epoch data: %w", err)
		}

		return epochData, startSlot, nil
	}

	has, err := b.epochState.HasEpochData(epoch)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot check for epoch data for epoch %d: %w", epoch, err)
	}

	if !has {
		logger.Criticalf("%s number=%d", errNoEpochData, epoch)
		return nil, 0, fmt.Errorf("%w: for epoch %d", errNoEpochData, epoch)
	}

	data, err := b.epochState.GetEpochData(epoch)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot get epoch data for epoch %d: %w", epoch, err)
	}

	idx, err := b.getAuthorityIndex(data.Authorities)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot get authority index: %w", err)
	}

	has, err = b.epochState.HasConfigData(epoch)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot check for config data for epoch %d: %w", epoch, err)
	}

	var cfgData *types.ConfigData
	if has {
		cfgData, err = b.epochState.GetConfigData(epoch)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot get config data for epoch %d: %w", epoch, err)
		}
	} else {
		cfgData, err = b.epochState.GetLatestConfigData()
		if err != nil {
			return nil, 0, fmt.Errorf("cannot get latest config data from epoch state: %w", err)
		}
	}

	threshold, err := CalculateThreshold(cfgData.C1, cfgData.C2, len(data.Authorities))
	if err != nil {
		return nil, 0, fmt.Errorf("cannot calculate threshold: %w", err)
	}

	ed := &epochData{
		randomness:     data.Randomness,
		authorities:    data.Authorities,
		authorityIndex: idx,
		threshold:      threshold,
		secondary:      cfgData.SecondarySlots,
	}

	startSlot, err := b.epochState.GetStartSlotForEpoch(epoch)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot get start slot for epoch %d: %w", epoch, err)
	}

	return ed, startSlot, nil
}

func (b *Service) getLatestEpochData() (resEpochData *epochData, error error) {
	resEpochData = &epochData{}

	epochData, err := b.epochState.GetLatestEpochData()
	if err != nil {
		return nil, fmt.Errorf("cannot get latest epoch data: %w", err)
	}

	resEpochData.randomness = epochData.Randomness
	resEpochData.authorities = epochData.Authorities

	configData, err := b.epochState.GetLatestConfigData()
	if err != nil {
		return nil, fmt.Errorf("cannot get epoch state latest config data: %w", err)
	}

	resEpochData.secondary = configData.SecondarySlots

	resEpochData.threshold, err = CalculateThreshold(configData.C1, configData.C2, len(resEpochData.authorities))
	if err != nil {
		return nil, fmt.Errorf("cannot calculate threshold: %w", err)
	}

	if !b.authority {
		return resEpochData, nil
	}

	resEpochData.authorityIndex, err = b.getAuthorityIndex(resEpochData.authorities)
	if err != nil {
		return nil, fmt.Errorf("cannot get authority index: %w", err)
	}

	return resEpochData, nil
}

func (b *Service) getFirstAuthoringSlot(epoch uint64, epochData *epochData) (uint64, error) {
	startSlot := getCurrentSlot(b.constants.slotDuration)
	for i := startSlot; i < startSlot+b.constants.epochLength; i++ {
		_, err := b.runLottery(i, epoch, epochData)
		if err != nil {
			if errors.Is(err, errOverPrimarySlotThreshold) {
				continue
			}
			if errors.Is(err, errNotOurTurnToPropose) {
				continue
			}
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
	err = b.epochState.SetCurrentEpoch(next)
	if err != nil {
		return 0, err
	}

	return next, nil
}

// runLottery runs the lottery for a specific slot number.
// It returns an encoded VrfOutputAndProof if the validator is authorised
// to produce a block for that slot.
// It returns the wrapped error errOverPrimarySlotThreshold
// if it is not authorised.
// output = return[0:32]; proof = return[32:96]
func (b *Service) runLottery(slot, epoch uint64, epochData *epochData) (*types.PreRuntimeDigest, error) {
	proof, err := claimPrimarySlot(
		epochData.randomness,
		slot,
		epoch,
		epochData.threshold,
		b.keypair,
	)
	if err == nil {
		preRuntimeDigest := types.NewBabePrimaryPreDigest(epochData.authorityIndex, slot, proof.output, proof.proof)

		return preRuntimeDigest.ToPreRuntimeDigest()
	}

	if errors.Is(err, errOverPrimarySlotThreshold) {
		if epochData.secondary == 0 {
			return nil, errSecondarySlotProductionDisabled
		}

		err = claimSecondarySlotPlain(epochData.randomness, slot,
			epochData.authorities, epochData.authorityIndex)
		if err != nil {
			return nil, fmt.Errorf("cannot claim secondary slot: %w", err)
		}
		preRuntimeDigest := types.NewBabeSecondaryPlainPreDigest(epochData.authorityIndex, slot)
		return preRuntimeDigest.ToPreRuntimeDigest()

	}

	return nil, err
}
