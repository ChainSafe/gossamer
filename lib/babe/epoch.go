// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package babe

import (
	"errors"
	"fmt"
)

// initiateEpoch sets the epochData for the given epoch, runs the lottery for the slots in the epoch,
// and stores updated EpochInfo in the database
func (b *Service) initiateEpoch(epoch uint64) error {
	var (
		startSlot uint64
		err       error
	)

	logger.Debug("initiating epoch", "epoch", epoch)

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
			logger.Crit("no epoch data for next BABE epoch", "epoch", epoch)
			return errNoEpochData
		}

		data, err := b.epochState.GetEpochData(epoch)
		if err != nil {
			return err
		}

		idx, err := b.getAuthorityIndex(data.Authorities)
		if err != nil && !errors.Is(err, ErrNotAuthority) { // TODO: this should be checked in the upper function
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
				threshold:      b.epochData.threshold,
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

		logger.Debug("estimated first slot based on building block 1", "slot", startSlot)
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

	logger.Info("initiating epoch", "epoch", epoch, "start slot", startSlot)

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
			logger.Trace("claimed slot!", "slot", startSlot, "slots into epoch", i-startSlot)
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
