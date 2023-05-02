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
func (b Bitfield) Generate(rand *rand.Rand, size int) reflect.Value {
	n := rand.Int() % size
	b.bits = make([]uint64, n)
	for i := range b.bits {
		b.bits[i] = rand.Uint64()
	}

	// we need to make sure we don't add empty words at the end of the
	// bitfield otherwise it would break equality on some of the tests
	// below.
	for len(b.bits) > 0 && b.bits[len(b.bits)-1] == 0 {
		b.bits = b.bits[:len(b.bits)-2]
	}
	return reflect.ValueOf(b)
}

// Test if the bit at the specified position is set.
func (b Bitfield) testBit(position uint) bool {
	wordOff := position / 64
	if wordOff >= uint(len(b.bits)) {
		return false
	}
	return testBit(b.bits[wordOff], position%64)
}

func Test_SetBit(t *testing.T) {
	f := func(a Bitfield, idx uint) bool {
		// let's bound the max bitfield index at 2^24. this is needed because when calling
		// `set_bit` we will extend the backing vec to accomodate the given bitfield size, this
		// way we restrict the maximum allocation size to 16MB.
		idx = uint(math.Min(float64(idx), 1<<24))
		a.SetBit(idx)
		return a.testBit(idx)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// translated from bitor test in
// https://github.com/paritytech/finality-grandpa/blob/fbe2404574f74713bccddfe4104d60c2a32d1fe6/src/bitfield.rs#L243
func Test_Merge(t *testing.T) {
	f := func(a, b Bitfield) bool {
		c := NewBitfield()
		copy(a.bits, c.bits)
		cBits := c.iter1s(0, 0)
		for _, bit := range cBits {
			if !(a.testBit(bit.Position) || b.testBit(bit.Position)) {
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
		f := func(a Bitfield) bool {
			b := NewBitfield()
			for _, bit1 := range a.iter1s(0, 0) {
				b.SetBit(bit1.Position)
			}
			return assert.Equal(t, a, b)
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("even odd", func(t *testing.T) {
		f := func(a Bitfield) bool {
			b := NewBitfield()
			for _, bit1 := range a.Iter1sEven() {
				assert.True(t, !b.testBit(bit1.Position))
				assert.True(t, bit1.Position%2 == 0)
				b.SetBit(bit1.Position)
			}
			for _, bit1 := range a.Iter1sOdd() {
				assert.True(t, !b.testBit(bit1.Position))
				assert.True(t, bit1.Position%2 == 1)
				b.SetBit(bit1.Position)
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
		f := func(a, b Bitfield) bool {
			c := NewBitfield()
			for _, bit1 := range a.iter1sMerged(b, 0, 0) {
				c.SetBit(bit1.Position)
			}
			return assert.Equal(t, &c, a.Merge(b))
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("even odd", func(t *testing.T) {
		f := func(a, b Bitfield) bool {
			c := NewBitfield()
			for _, bit1 := range a.Iter1sMergedEven(b) {
				assert.True(t, !c.testBit(bit1.Position))
				assert.True(t, bit1.Position%2 == 0)
				c.SetBit(bit1.Position)
			}
			for _, bit1 := range a.Iter1sMergedOdd(b) {
				assert.True(t, !c.testBit(bit1.Position))
				assert.True(t, bit1.Position%2 == 1)
				c.SetBit(bit1.Position)
			}
			return assert.Equal(t, &c, a.Merge(b))
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
}
