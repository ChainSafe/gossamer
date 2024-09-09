// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

// Generate is used by testing/quick to genereate
func (bitfield) Generate(rand *rand.Rand, size int) reflect.Value { //skipcq: GO-W1029
	n := rand.Int() % size
	bits := make([]uint64, n)
	for i := range bits {
		bits[i] = rand.Uint64()
	}

	// we need to make sure we don't add empty words at the end of the
	// bitfield otherwise it would break equality on some of the tests
	// below.
	for len(bits) > 0 && bits[len(bits)-1] == 0 {
		bits = bits[:len(bits)-2]
	}
	return reflect.ValueOf(bitfield{
		bits: bits,
	})
}

// Test if the bit at the specified position is set.
func (b *bitfield) testBit(position uint) bool { //skipcq: GO-W1029
	wordOff := position / 64
	if wordOff >= uint(len(b.bits)) {
		return false
	}
	return testBit(b.bits[wordOff], position%64)
}

func TestBitfield_SetBit(t *testing.T) {
	f := func(a bitfield, idx uint) bool {
		// let's bound the max bitfield index at 2^24. this is needed because when calling
		// `SetBit` we will extend the backing vec to accommodate the given bitfield size, this
		// way we restrict the maximum allocation size to 16MB.
		idx = uint(math.Min(float64(idx), 1<<24))
		a.SetBit(idx)
		return a.testBit(idx)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestBitfield_iter1s_bitor(t *testing.T) {
	f := func(a, b bitfield) bool {
		c := newBitfield()
		copy(a.bits, c.bits)
		cBits := c.iter1s(0, 0)
		for _, bit := range cBits {
			if !(a.testBit(bit.position) || b.testBit(bit.position)) {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func Test_iter1s(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		f := func(a bitfield) bool {
			b := newBitfield()
			for _, bit1 := range a.iter1s(0, 0) {
				b.SetBit(bit1.position)
			}
			return assert.Equal(t, a, b)
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("even_odd", func(t *testing.T) {
		f := func(a bitfield) bool {
			b := newBitfield()
			for _, bit1 := range a.Iter1sEven() {
				assert.True(t, !b.testBit(bit1.position))
				assert.True(t, bit1.position%2 == 0)
				b.SetBit(bit1.position)
			}
			for _, bit1 := range a.Iter1sOdd() {
				assert.True(t, !b.testBit(bit1.position))
				assert.True(t, bit1.position%2 == 1)
				b.SetBit(bit1.position)
			}
			return assert.Equal(t, a, b)
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
}

func Test_iter1sMerged(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		f := func(a, b bitfield) bool {
			c := newBitfield()
			for _, bit1 := range a.iter1sMerged(b, 0, 0) {
				c.SetBit(bit1.position)
			}
			return assert.Equal(t, &c, a.Merge(b))
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("even_odd", func(t *testing.T) {
		f := func(a, b bitfield) bool {
			c := newBitfield()
			for _, bit1 := range a.Iter1sMergedEven(b) {
				assert.True(t, !c.testBit(bit1.position))
				assert.True(t, bit1.position%2 == 0)
				c.SetBit(bit1.position)
			}
			for _, bit1 := range a.Iter1sMergedOdd(b) {
				assert.True(t, !c.testBit(bit1.position))
				assert.True(t, bit1.position%2 == 1)
				c.SetBit(bit1.position)
			}
			return assert.Equal(t, &c, a.Merge(b))
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
}
