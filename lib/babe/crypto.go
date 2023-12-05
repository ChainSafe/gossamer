// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/gtank/merlin"
)

// the code in this file is based off
// https://github.com/paritytech/substrate/blob/89275433863532d797318b75bb5321af098fea7c/primitives/consensus/babe/src/lib.rs#L93
var babeVRFPrefix = []byte("substrate-babe-vrf")

func makeTranscript(randomness Randomness, slot, epoch uint64) *merlin.Transcript {
	t := merlin.NewTranscript("BABE") //string(types.BabeEngineID[:])
	crypto.AppendUint64(t, []byte("slot number"), slot)
	crypto.AppendUint64(t, []byte("current epoch"), epoch)
	t.AppendMessage([]byte("chain randomness"), randomness[:])
	return t
}

// claimPrimarySlot checks if a slot can be claimed.
// If it cannot be claimed, the wrapped error
// errOverPrimarySlotThreshold is returned.
// https://github.com/paritytech/substrate/blob/master/client/consensus/babe/src/authorship.rs#L239
func claimPrimarySlot(randomness Randomness,
	slot, epoch uint64,
	threshold *scale.Uint128,
	keypair *sr25519.Keypair,
) (*VrfOutputAndProof, error) {
	transcript := makeTranscript(randomness, slot, epoch)

	out, proof, err := keypair.VrfSign(transcript)
	if err != nil {
		return nil, err
	}

	logger.Tracef("claimPrimarySlot pub=%s slot=%d epoch=%d output=0x%x proof=0x%x",
		keypair.Public().Hex(), slot, epoch, out, proof)

	ok, err := checkPrimaryThreshold(randomness, slot, epoch, out, threshold, keypair.Public().(*sr25519.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to compare with threshold, %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: for slot %d, epoch %d and threshold %s",
			errOverPrimarySlotThreshold, slot, epoch, threshold)
	}

	return &VrfOutputAndProof{
		output: out,
		proof:  proof,
	}, nil
}

// checkPrimaryThreshold returns true if the authority was authorized to produce a block in the given slot and epoch
func checkPrimaryThreshold(randomness Randomness,
	slot, epoch uint64,
	output [sr25519.VRFOutputLength]byte,
	threshold *scale.Uint128,
	pub *sr25519.PublicKey,
) (bool, error) {
	t := makeTranscript(randomness, slot, epoch)
	inout, err := sr25519.AttachInput(output, pub, t)
	if err != nil {
		return false, fmt.Errorf("attaching sr25519 input: %w", err)
	}

	const size = 16
	res, err := inout.MakeBytes(size, babeVRFPrefix)
	if err != nil {
		return false, fmt.Errorf("making sr25519 bytes: %w", err)
	}

	inoutUint, err := scale.NewUint128(res)
	if err != nil {
		return false, fmt.Errorf("failed to convert bytes to Uint128: %w", err)
	}

	logger.Tracef("checkPrimaryThreshold pub=%s randomness=0x%x slot=%d epoch=%d threshold=0x%x output=0x%x inout=0x%x",
		pub.Hex(), randomness, slot, epoch, threshold, output, res)

	return inoutUint.Compare(threshold) < 0, nil
}

func claimSecondarySlotVRF(randomness Randomness,
	slot, epoch uint64,
	authorities []types.AuthorityRaw,
	keypair *sr25519.Keypair,
	authorityIndex uint32,
) (*VrfOutputAndProof, error) {

	secondarySlotAuthor, err := getSecondarySlotAuthor(slot, len(authorities), randomness)
	if err != nil {
		return nil, fmt.Errorf("cannot get secondary slot author: %w", err)
	}

	if authorityIndex != secondarySlotAuthor {
		return nil, errNotOurTurnToPropose
	}

	transcript := makeTranscript(randomness, slot, epoch)

	out, proof, err := keypair.VrfSign(transcript)
	if err != nil {
		return nil, fmt.Errorf("cannot verify transcript: %w", err)
	}

	logger.Debugf("claimed secondary slot, for slot number: %d", slot)

	return &VrfOutputAndProof{
		output: out,
		proof:  proof,
	}, nil
}

func claimSecondarySlotPlain(randomness Randomness, slot uint64, authorities []types.AuthorityRaw, authorityIndex uint32,
) error {
	secondarySlotAuthor, err := getSecondarySlotAuthor(slot, len(authorities), randomness)
	if err != nil {
		return fmt.Errorf("cannot get secondary slot author: %w", err)
	}

	if authorityIndex != secondarySlotAuthor {
		return errNotOurTurnToPropose
	}

	logger.Debugf("claimed secondary slot, for slot number: %d", slot)
	return nil
}

// CalculateThreshold calculates the slot lottery threshold
// equation: threshold = 2^128 * (1 - (1-c)^(1/len(authorities))
// see https://github.com/paritytech/substrate/blob/master/client/consensus/babe/src/authorship.rs#L44
func CalculateThreshold(C1, C2 uint64, numAuths int) (*scale.Uint128, error) {
	if C1 == 0 || C2 == 0 {
		return nil, ErrThresholdOneIsZero
	}
	c := float64(C1) / float64(C2)
	if c > 1 {
		return nil, errors.New("invalid C1/C2: greater than 1")
	}

	// 1 / len(authorities)
	theta := float64(1) / float64(numAuths)

	// (1-c)^(theta)
	pp := 1 - c
	ppExp := math.Pow(pp, theta)

	// 1 - (1-c)^(theta)
	p := 1 - ppExp
	pRat := new(big.Rat).SetFloat64(p)

	// 1 << 128
	shift := new(big.Int).Lsh(big.NewInt(1), 128)
	numer := new(big.Int).Mul(shift, pRat.Num())
	denom := pRat.Denom()

	// (1 << 128) * (1 - (1-c)^(w_k/sum(w_i)))
	thresholdBig := new(big.Int).Div(numer, denom)

	// special case where threshold is maximum
	if thresholdBig.Cmp(shift) == 0 {
		return scale.MaxUint128, nil
	}

	if len(thresholdBig.Bytes()) > 16 {
		return nil, errors.New("threshold must be under or equal to 16 bytes")
	}

	return scale.NewUint128(thresholdBig)
}
