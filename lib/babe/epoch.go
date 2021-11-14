// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"fmt"
)

// initiateEpoch sets the epochData for the given epoch, runs the lottery for the slots in the epoch,
// and stores updated EpochInfo in the database
func (b *Service) initiateEpoch(epoch uint64) error {
	var (
		startSlot uint64
		err       error
	)

	logger.Debugf("initiating epoch %d", epoch)

	if epoch == 0 {
		startSlot, err = b.epochState.GetStartSlotForEpoch(epoch)
		if err != nil {
			return err
		}
	} else if epoch > 0 {
		has, err := b.epochState.HasEpochData(epoch) //nolint
		if err != nil {
			return err
		}

		if !has {
			logger.Criticalf("%s, for epoch %d", errNoEpochData, epoch)
			return errNoEpochData
		}

		data, err := b.epochState.GetEpochData(epoch)
		if err != nil {
			return err
		}

		idx, err := b.getAuthorityIndex(data.Authorities)
		if err != nil {
			return err
		}

		has, err = b.epochState.HasConfigData(epoch)
		if err != nil {
			return err
		}

		if has {
			cfgData, err := b.epochState.GetConfigData(epoch) //nolint
			if err != nil {
				return err
			}

			threshold, err := CalculateThreshold(cfgData.C1, cfgData.C2, len(data.Authorities))
			if err != nil {
				return err
			}

			b.epochData = &epochData{
				randomness:     data.Randomness,
				authorities:    data.Authorities,
				authorityIndex: idx,
				threshold:      threshold,
			}
		} else {
			b.epochData = &epochData{
				randomness:     data.Randomness,
				authorities:    data.Authorities,
				authorityIndex: idx,
				threshold:      b.epochData.threshold, // TODO: threshold might change if authority count changes
			}
		}

		startSlot, err = b.epochState.GetStartSlotForEpoch(epoch)
		if err != nil {
			return err
		}
	}

	// if we're at genesis, we need to determine when the first slot of the network will be
	// by checking when we will be able to produce block 1.
	// note that this assumes there will only be one producer of block 1
	if b.blockState.BestBlockHash() == b.blockState.GenesisHash() {
		startSlot, err = b.getFirstSlot(epoch)
		if err != nil {
			return err
		}

		logger.Debugf("estimated first slot as %d based on building block 1", startSlot)
		for i := startSlot; i < startSlot+b.epochLength; i++ {
			proof, err := b.runLottery(i, epoch) //nolint
			if err != nil {
				return fmt.Errorf("error running slot lottery at slot %d: error %w", i, err)
			}

			if proof != nil {
				startSlot = i
				break
			}
		}

		// we are at genesis, set first slot by checking at which slot we will be able to produce block 1
		err = b.epochState.SetFirstSlot(startSlot)
		if err != nil {
			return err
		}
	}

	logger.Infof("initiating epoch %d with start slot %d", epoch, startSlot)

	for i := startSlot; i < startSlot+b.epochLength; i++ {
		if epoch > 0 {
			delete(b.slotToProof, i-b.epochLength) // clear data from previous epoch
		}

		proof, err := b.runLottery(i, epoch)
		if err != nil {
			return fmt.Errorf("error running slot lottery at slot %d: error %w", i, err)
		}

		if proof != nil {
			b.slotToProof[i] = proof
			logger.Tracef("claimed slot %d, there are now %d slots into epoch", startSlot, i-startSlot)
		}
	}

	return nil
}

func (b *Service) getFirstSlot(epoch uint64) (uint64, error) {
	startSlot := getCurrentSlot(b.slotDuration)
	for i := startSlot; i < startSlot+b.epochLength; i++ {
		proof, err := b.runLottery(i, epoch)
		if err != nil {
			return 0, fmt.Errorf("error running slot lottery at slot %d: error %w", i, err)
		}

		if proof != nil {
			startSlot = i
			break
		}
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

// runLottery runs the lottery for a specific slot number
// returns an encoded VrfOutput and VrfProof if validator is authorized to produce a block for that slot, nil otherwise
// output = return[0:32]; proof = return[32:96]
func (b *Service) runLottery(slot, epoch uint64) (*VrfOutputAndProof, error) {
	return claimPrimarySlot(
		b.epochData.randomness,
		slot,
		epoch,
		b.epochData.threshold,
		b.keypair,
	)
}
