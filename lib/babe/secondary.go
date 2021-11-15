// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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
