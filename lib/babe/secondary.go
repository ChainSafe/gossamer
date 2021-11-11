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
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

func getSecondarySlotAuthor(slot uint64, numAuths int, randomness Randomness) (uint32, error) {
	s := make([]byte, 8)
	binary.LittleEndian.PutUint64(s, slot)
	rand, err := common.Blake2bHash(append(randomness[:], s...))
	if err != nil {
		return 0, err
	}

	randBig := new(big.Int).SetBytes(rand[:])
	num := big.NewInt(int64(numAuths))

	idx := new(big.Int).Mod(randBig, num)
	return uint32(idx.Uint64()), nil
}

// see https://github.com/paritytech/substrate/blob/master/client/consensus/babe/src/authorship.rs#L108
func verifySecondarySlotPlain(authorityIndex uint32, slot uint64, numAuths int, randomness Randomness) error {
	expected, err := getSecondarySlotAuthor(slot, numAuths, randomness)
	if err != nil {
		return err
	}

	logger.Tracef("verifySecondarySlotPlain authority index %d, %d authorities, slot number %d, randomness 0x%x and expected index %d",
		authorityIndex, numAuths, slot, randomness, expected)

	if authorityIndex != expected {
		return ErrBadSecondarySlotClaim
	}

	return nil
}

// see https://github.com/paritytech/substrate/blob/master/client/consensus/babe/src/authorship.rs#L132
func verifySecondarySlotVRF(digest *types.BabeSecondaryVRFPreDigest,
	pk *sr25519.PublicKey,
	epoch uint64,
	numAuths int,
	randomness Randomness,
) (bool, error) {
	expected, err := getSecondarySlotAuthor(digest.SlotNumber, numAuths, randomness)
	if err != nil {
		return false, err
	}

	logger.Tracef("verifySecondarySlotVRF authority index %d, public key %s, %d authorities, slot number %d, epoch %d, randomness 0x%x and expected index %d",
		digest.AuthorityIndex, pk.Hex(), numAuths, digest.SlotNumber, epoch, randomness, expected)

	if digest.AuthorityIndex != expected {
		return false, ErrBadSecondarySlotClaim
	}

	t := makeTranscript(randomness, digest.SlotNumber, epoch)
	return pk.VrfVerify(t, digest.VrfOutput, digest.VrfProof)
}
