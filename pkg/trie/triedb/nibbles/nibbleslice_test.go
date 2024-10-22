// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibbles

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNibbleSlice_Push(t *testing.T) {
	v := NewNibbleSlice()
	for i := uint(0); i < NibblesPerByte*3; i++ {
		iu8 := uint8(i % NibblesPerByte)
		v.Push(iu8)
		assert.Equal(t, i, v.len-1)
		assert.Equal(t, v.At(i), iu8)
	}

	for i := int(NibblesPerByte*3) - 1; i >= 0; i-- {
		iu8 := uint8(uint(i) % NibblesPerByte)
		a := v.Pop()
		assert.NotNil(t, a)
		assert.Equal(t, iu8, *a)
		assert.Equal(t, v.len, uint(i))
	}
}

func TestNibbleSlice_AppendPartial(t *testing.T) {
	t.Run("", func(t *testing.T) {
		appendPartial(t, []byte{1, 2, 3}, []byte{}, Partial{First: 1, PaddedNibble: 1, Data: []byte{0x23}})
	})
	t.Run("", func(t *testing.T) {
		appendPartial(t, []byte{1, 2, 3}, []byte{1}, Partial{First: 0, PaddedNibble: 0, Data: []byte{0x23}})
	})
	t.Run("", func(t *testing.T) {
		appendPartial(t, []byte{0, 1, 2, 3}, []byte{0}, Partial{First: 1, PaddedNibble: 1, Data: []byte{0x23}})
	})
}

func appendPartial(t *testing.T, res []uint8, init []uint8, partial Partial) {
	t.Helper()
	resv := NewNibbleSlice()
	for _, r := range res {
		resv.Push(r)
	}
	initv := NewNibbleSlice()
	for _, r := range init {
		initv.Push(r)
	}
	initv.AppendPartial(partial)
	assert.Equal(t, resv, initv)
}
