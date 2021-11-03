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
	"math"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/gtank/merlin"
)

// the code in this file is based off https://github.com/paritytech/substrate/blob/89275433863532d797318b75bb5321af098fea7c/primitives/consensus/babe/src/lib.rs#L93
var babe_vrf_prefix = []byte("substrate-babe-vrf")

func makeTranscript(randomness Randomness, slot, epoch uint64) *merlin.Transcript {
	t := merlin.NewTranscript("BABE") //string(types.BabeEngineID[:])
	crypto.AppendUint64(t, []byte("slot number"), slot)
	crypto.AppendUint64(t, []byte("current epoch"), epoch)
	t.AppendMessage([]byte("chain randomness"), randomness[:])
	return t
}

// claimPrimarySlot checks if a slot can be claimed. if it can be, then a *VrfOutputAndProof is returned, otherwise nil.
// https://github.com/paritytech/substrate/blob/master/client/consensus/babe/src/authorship.rs#L239
func claimPrimarySlot(randomness Randomness,
	slot, epoch uint64,
	threshold *common.Uint128,
	keypair *sr25519.Keypair,
) (*VrfOutputAndProof, error) {
	transcript := makeTranscript(randomness, slot, epoch)

	out, proof, err := keypair.VrfSign(transcript)
	if err != nil {
		return nil, err
	}

	logger.Tracef("claimPrimarySlot pub=%s slot=%d epoch=%d output=0x%x proof=0x%x",
		keypair.Public().Hex(), slot, epoch, out, proof)

	ok := checkPrimaryThreshold(randomness, slot, epoch, out, threshold, keypair.Public().(*sr25519.PublicKey))
	if !ok {
		return nil, nil
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
	threshold *common.Uint128,
	pub *sr25519.PublicKey,
) bool {
	t := makeTranscript(randomness, slot, epoch)
	inout := sr25519.AttachInput(output, pub, t)
	res := sr25519.MakeBytes(inout, 16, babe_vrf_prefix)

	inoutUint := common.Uint128FromLEBytes(res)

	logger.Tracef("checkPrimaryThreshold pub=%s randomness=0x%x slot=%d epoch=%d threshold=0x%x output=0x%x inout=0x%x",
		pub.Hex(), randomness, slot, epoch, threshold, output, res)

	return inoutUint.Cmp(threshold) < 0
}

// CalculateThreshold calculates the slot lottery threshold
// equation: threshold = 2^128 * (1 - (1-c)^(1/len(authorities))
// see https://github.com/paritytech/substrate/blob/master/client/consensus/babe/src/authorship.rs#L44
func CalculateThreshold(C1, C2 uint64, numAuths int) (*common.Uint128, error) {
	c := float64(C1) / float64(C2)
	if c > 1 {
		return nil, errors.New("invalid C1/C2: greater than 1")
	}

	// 1 / len(authorities)
	theta := float64(1) / float64(numAuths)

	// (1-c)^(theta)
	pp := 1 - c
	pp_exp := math.Pow(pp, theta)

	// 1 - (1-c)^(theta)
	p := 1 - pp_exp
	p_rat := new(big.Rat).SetFloat64(p)

	// 1 << 128
	shift := new(big.Int).Lsh(big.NewInt(1), 128)
	numer := new(big.Int).Mul(shift, p_rat.Num())
	denom := p_rat.Denom()

	// (1 << 128) * (1 - (1-c)^(w_k/sum(w_i)))
	threshold_big := new(big.Int).Div(numer, denom)

	// special case where threshold is maximum
	if threshold_big.Cmp(shift) == 0 {
		return common.MaxUint128, nil
	}

	if len(threshold_big.Bytes()) > 16 {
		return nil, errors.New("threshold must be under or equal to 16 bytes")
	}

	return common.Uint128FromBigInt(threshold_big), nil
}
