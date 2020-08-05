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
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
)

func (b *Service) initiateEpoch(epoch, startSlot uint64) error {
	if epoch > 1 {
		b.randomness = b.epochRandomness(epoch)
	}

	if epoch > 0 {
		first, err := b.blockState.BestBlockNumber()
		if err != nil {
			return err
		}

		// Duration may only change when the runtime is updated. This call happens in SetRuntime()
		// FirstBlock is used to calculate the randomness (blocks in an epoch used to calculate epoch randomness for 2 epochs ahead)
		// Randomness changes every epoch, as calculated by epochRandomness()
		info := &types.EpochInfo{
			Duration:   b.config.EpochLength,
			FirstBlock: first.Uint64(),
			Randomness: b.randomness,
		}

		err = b.epochState.SetEpochInfo(epoch, info)
		if err != nil {
			return err
		}
	}

	var err error
	i := startSlot
	for ; i < b.startSlot+b.config.EpochLength; i++ {
		b.slotToProof[i], err = b.runLottery(i)
		if err != nil {
			return fmt.Errorf("error running slot lottery at slot %d: error %s", i, err)
		}
	}

	return nil
}

func (b *Service) epochRandomness(epoch uint64) [types.RandomnessLength]byte {
	return b.randomness //[types.RandomnessLength]byte{}
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
