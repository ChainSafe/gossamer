// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
)

// initiateEpoch sets the epochData for the given epoch, runs the lottery for the slots in the epoch,
// and stores updated EpochInfo in the database
func (b *Service) initiateEpoch(epoch uint64) (*epochData, error) {
	var (
		startSlot uint64
		err       error
		ed        *epochData
	)

	logger.Debugf("initiating epoch %d", epoch)

	// TODO: if epoch == 1, check that first slot is still set correctly
	if epoch == 0 {
		startSlot, err = b.epochState.GetStartSlotForEpoch(epoch)
		if err != nil {
			return nil, err
		}

		ed, err = b.getLatestEpochData()
		if err != nil {
			return nil, err
		}
	} else if epoch > 0 {
		has, err := b.epochState.HasEpochData(epoch)
		if err != nil {
			return nil, err
		}

		if !has {
			logger.Criticalf("%s, for epoch %d", errNoEpochData, epoch)
			return nil, errNoEpochData
		}

		data, err := b.epochState.GetEpochData(epoch)
		if err != nil {
			return nil, err
		}

		idx, err := b.getAuthorityIndex(data.Authorities)
		if err != nil {
			return nil, err
		}

		has, err = b.epochState.HasConfigData(epoch)
		if err != nil {
			return nil, err
		}

		var cfgData *types.ConfigData
		if has {
			cfgData, err = b.epochState.GetConfigData(epoch)
			if err != nil {
				return nil, err
			}
		} else {
			cfgData, err = b.epochState.GetLatestConfigData()
			if err != nil {
				return nil, err
			}
		}

		threshold, err := CalculateThreshold(cfgData.C1, cfgData.C2, len(data.Authorities))
		if err != nil {
			return nil, err
		}

		ed = &epochData{
			randomness:     data.Randomness,
			authorities:    data.Authorities,
			authorityIndex: idx,
			threshold:      threshold,
		}

		startSlot, err = b.epochState.GetStartSlotForEpoch(epoch)
		if err != nil {
			return nil, err
		}
	}

	// if we're at genesis, we need to determine when the first slot of the network will be
	// by checking when we will be able to produce block 1.
	// note that this assumes there will only be one producer of block 1
	if b.blockState.BestBlockHash() == b.blockState.GenesisHash() {
		startSlot, err = b.getFirstSlot(epoch, ed)
		if err != nil {
			return nil, err
		}

		logger.Debugf("estimated first slot as %d based on building block 1", startSlot)

		// we are at genesis, set first slot by checking at which slot we will be able to produce block 1
		if err = b.epochState.SetFirstSlot(startSlot); err != nil {
			return nil, err
		}
	}

	logger.Infof("initiating epoch %d with start slot %d", epoch, startSlot)
	return ed, nil
}

func (b *Service) getLatestEpochData() (*epochData, error) {
	var err error
	resEpochData := &epochData{}

	epochData, err := b.epochState.GetLatestEpochData()
	if err != nil {
		return nil, err
	}

	resEpochData.randomness = epochData.Randomness
	resEpochData.authorities = epochData.Authorities

	configData, err := b.epochState.GetLatestConfigData()
	if err != nil {
		return nil, err
	}

	resEpochData.threshold, err = CalculateThreshold(configData.C1, configData.C2, len(resEpochData.authorities))
	if err != nil {
		return nil, err
	}

	if !b.authority {
		return resEpochData, nil
	}

	resEpochData.authorityIndex, err = b.getAuthorityIndex(resEpochData.authorities)
	return resEpochData, err
}

func (b *Service) getFirstSlot(epoch uint64, epochData *epochData) (uint64, error) {
	startSlot := getCurrentSlot(b.constants.slotDuration)
	for i := startSlot; i < startSlot+b.constants.epochLength; i++ {
		_, err := b.runLottery(i, epoch, epochData)
		if err != nil {
			if errors.Is(err, errOverPrimarySlotThreshold) {
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
func (b *Service) runLottery(slot, epoch uint64, epochData *epochData) (*VrfOutputAndProof, error) {
	return claimPrimarySlot(
		epochData.randomness,
		slot,
		epoch,
		epochData.threshold,
		b.keypair,
	)
}
