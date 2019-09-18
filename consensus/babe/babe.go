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
	"math/big"

	"github.com/ChainSafe/gossamer/runtime"
)

// BabeSession contains the VRF keys for the validator
type BabeSession struct {
	vrfPublicKey  VrfPublicKey
	vrfPrivateKey VrfPrivateKey
	rt            *runtime.Runtime

	config    *BabeConfiguration
	epochData *Epoch

	authorityIndex uint64

	// authorities []VrfPublicKey
	authorityWeights []uint64

	// currentEpoch uint64
	// currentSlot  uint64

	// TODO: TransactionQueue
}

// NewBabeSession returns a new Babe session using the provided VRF keys and runtime
func NewBabeSession(pubkey VrfPublicKey, privkey VrfPrivateKey, rt *runtime.Runtime) *BabeSession {
	return &BabeSession{
		vrfPublicKey:  pubkey,
		vrfPrivateKey: privkey,
		rt:            rt,
	}
}

// runs the slot lottery for a specific slot
// returns true if validator is authorized to produce a block for that slot, false otherwise
func (b *BabeSession) runLottery(slot uint64) (bool, error) {
	if slot < b.epochData.StartSlot {
		return false, errors.New("slot is not in this epoch")
	}

	output, err := b.vrfSign(slot)
	if err != nil {
		return false, err
	}

	output_int := new(big.Int).SetBytes(output)
	threshold := calculateThreshold(b.config.C1, b.config.C2, b.authorityIndex, b.authorityWeights)

	return output_int.Cmp(threshold) > 0, nil
}

func (b *BabeSession) vrfSign(slot uint64) ([]byte, error) {
	// TOOD: return VRF output and proof
	// sign b.epochData.Randomness and slot
	return nil, nil
}

// calculates the slot lottery threshold for the authority at authorityIndex.
// equation: threshold = 2^128 * (1 - (1-c)^(w_k/sum(w_i)))
// where w_k is the weight of the authority at the specified index, and sum(w_i) is the
// sum of all the authority weights
// see: https://github.com/paritytech/substrate/blob/master/core/consensus/babe/src/lib.rs#L1022
func calculateThreshold(C1, C2, authorityIndex uint64, authorityWeights []uint64) *big.Int {
	var sum uint64 = 0
	for _, weight := range authorityWeights {
		sum += weight
	}

	theta := float64(authorityWeights[authorityIndex]) / float64(sum)
	c := new(big.Float).SetFloat64(float64(C1) / float64(C2))
	// c_int, _ := c.Int(new(big.Int))
	// return c_int

	pp := big.NewFloat(1).Sub(big.NewFloat(1), c) // 1-c
	pp_exp := pp.MantExp(pp) // get exponent of 1-c
	pp_exp_theta := int(float64(pp_exp) * theta) // multiply exponent by theta
	pp_theta := new(big.Float).SetMantExp(pp, pp_exp_theta) // turn it back into a float
	// pp_theta_int, _ := pp_theta.Int(new(big.Int))
	// return pp_theta_int

	p := new(big.Float).Sub(big.NewFloat(1), pp_theta)
	p_f64, _ := p.Float64()
	p_rat := new(big.Rat).SetFloat64(p_f64)
	q := new(big.Int).Lsh(big.NewInt(1), 128) // 1 << 128

	return q.Mul(q, p_rat.Num()).Div(q, p_rat.Denom())

}
