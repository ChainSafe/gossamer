// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"math"
	"math/big"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
)

func (VoterSet[ID]) Generate(rand *rand.Rand, _ int) reflect.Value {
	for {
		idsValue, ok := quick.Value(reflect.TypeOf(make([]ID, 0)), rand)
		if !ok {
			panic("unable to generate value")
		}
		ids := idsValue.Interface().([]ID)
		weights := make([]IDWeight[ID], len(ids))
		for i, id := range ids {
			u64v, ok := quick.Value(reflect.TypeOf(uint64(0)), rand)
			if !ok {
				panic("unable to generate value")
			}
			weights[i] = IDWeight[ID]{
				id,
				u64v.Interface().(uint64),
			}
		}
		set := NewVoterSet(weights)
		if set == nil {
			continue
		}
		return reflect.ValueOf(*set)
	}
}

func TestVoterSet_Equality(t *testing.T) {
	f := func(v []IDWeight[uint]) bool {
		v1 := NewVoterSet(v)
		if v1 != nil {
			rand := rand.New(rand.NewSource(time.Now().UnixNano())) //skipcq: GSC-G404
			rand.Shuffle(len(v), func(i, j int) { v[i], v[j] = v[j], v[i] })
			v2 := NewVoterSet(v)
			assert.NotNil(t, v1)
			return assert.Equal(t, v1, v2)
		}
		// either no authority has a valid weight
		var noValIDWeight = true
		for _, iw := range v {
			if iw.Weight != 0 {
				noValIDWeight = false
				break
			}
		}
		if noValIDWeight == true {
			return true
		}
		// or the total weight overflows a uint64
		sum := big.NewInt(0)
		for _, iw := range v {
			sum.Add(sum, new(big.Int).SetUint64(iw.Weight))
		}
		return sum.Cmp(new(big.Int).SetUint64(uint64(math.MaxUint64))) > 0

	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVoterSet_TotalWeight(t *testing.T) {
	f := func(v []IDWeight[uint]) bool {
		totalWeight := big.NewInt(0)
		for _, iw := range v {
			totalWeight.Add(totalWeight, new(big.Int).SetUint64(iw.Weight))
		}
		// this validator set is invalid
		if totalWeight.Cmp(new(big.Int).SetUint64(uint64(math.MaxUint64))) > 0 {
			return true
		}

		expected := VoterWeight(totalWeight.Uint64())
		v1 := NewVoterSet(v)
		if v1 != nil {
			return assert.Equal(t, expected, v1.totalWeight)
		}
		return assert.Equal(t, expected, VoterWeight(0))
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVoterSet_MinTreshold(t *testing.T) {
	f := func(v VoterSet[uint]) bool {
		t := v.threshold
		w := v.totalWeight
		return t >= 2*(w/3)+(w%3)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
