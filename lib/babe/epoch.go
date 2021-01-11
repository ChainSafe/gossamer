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
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

// initiateEpoch sets the randomness for the given epoch, runs the lottery for the slots in the epoch,
// and stores updated EpochInfo in the database
func (b *Service) initiateEpoch(epoch, startSlot uint64) error {
	if epoch > 1 {
		has, err := b.epochState.HasEpochData(epoch)
		if err != nil {
			return err
		}

		var data *types.EpochData
		if !has {
			data = &types.EpochData{
				Randomness:  b.epochData.randomness,
				Authorities: b.epochData.authorities,
			}

			err = b.epochState.SetEpochData(epoch, data)
		} else {
			data, err = b.epochState.GetEpochData(epoch)
		}

		if err != nil {
			return err
		}

		idx, err := b.getAuthorityIndex(data.Authorities)
		if err != nil && err != ErrNotAuthority {
			return err
		}

		has, err = b.epochState.HasConfigData(epoch)
		if err != nil {
			return err
		}

		if has {
			cfgData, err := b.epochState.GetConfigData(epoch)
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
	}

	if !b.authority {
		return nil
	}

	var err error
	for i := startSlot; i < startSlot+b.epochLength; i++ {
		b.slotToProof[i], err = b.runLottery(i)
		if err != nil {
			return fmt.Errorf("error running slot lottery at slot %d: error %s", i, err)
		}
	}

	// if we were previously disabled, we are now re-enabled since the epoch changed
	b.isDisabled = false
	return nil
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
func (b *Service) runLottery(slot uint64) (*VrfOutputAndProof, error) {
	slotBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBytes, slot)
	vrfInput := append(slotBytes, b.epochData.randomness[:]...)

	output, proof, err := b.vrfSign(vrfInput)
	if err != nil {
		return nil, err
	}

	outputInt := big.NewInt(0).SetBytes(output[:])

	if outputInt.Cmp(b.epochData.threshold) < 0 {
		outbytes := [sr25519.VrfOutputLength]byte{}
		copy(outbytes[:], output)
		proofbytes := [sr25519.VrfProofLength]byte{}
		copy(proofbytes[:], proof)
		b.logger.Trace("lottery", "won slot", slot)
		return &VrfOutputAndProof{
			output: outbytes,
			proof:  proofbytes,
		}, nil
	}

	return nil, nil
}

func getVRFOutput(header *types.Header) ([sr25519.VrfOutputLength]byte, error) {
	var bh *types.BabeHeader

	for _, d := range header.Digest {
		digest, err := types.DecodeDigestItem(d)
		if err != nil {
			continue
		}

		if digest.Type() == types.PreRuntimeDigestType {
			prd, ok := digest.(*types.PreRuntimeDigest)
			if !ok {
				continue
			}

			tbh := new(types.BabeHeader)
			err = tbh.Decode(prd.Data)
			if err != nil {
				continue
			}

			bh = tbh
			break
		}
	}

	if bh == nil {
		return [sr25519.VrfOutputLength]byte{}, fmt.Errorf("block %d: %w", header.Number, ErrNoBABEHeader)
	}

	return bh.VrfOutput, nil
}
