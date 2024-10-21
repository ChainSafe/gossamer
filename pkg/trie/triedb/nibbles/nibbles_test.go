// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibbles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNibbles(t *testing.T) {
	data := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	nibbles := NewNibbles(data)
	for i := 0; i < (2 * len(data)); i++ {
		assert.Equal(t, uint8(i), nibbles.At(uint(i)))
	}
}

func TestNibbles_MidAndLeft(t *testing.T) {
	n := NewNibbles([]byte{0x01, 0x23, 0x45})
	m := n.Mid(2)
	for i := 0; i < 4; i++ {
		assert.Equal(t, m.At(uint(i)), uint8(i+2))
	}
	assert.Equal(t, Prefix{Key: []byte{0x01}}, m.Left())
	m = n.Mid(3)
	for i := 0; i < 3; i++ {
		assert.Equal(t, m.At(uint(i)), uint8(i+3))
	}
	padded := uint8(0x23 &^ 0x0F)
	assert.Equal(t, Prefix{Key: []byte{0x01}, Padded: &padded}, m.Left())
}

func TestNibbles_Right(t *testing.T) {
	data := []uint8{1, 2, 3, 4, 5, 234, 78, 99}
	nibbles := NewNibbles(data)
	assert.Equal(t, data, nibbles.Right())

	nibbles = NewNibbles(data, 3)
	assert.Equal(t, data[1:], nibbles.Right())
}

func TestNibbles_CommonPrefix(t *testing.T) {
	n := NewNibbles([]byte{0x01, 0x23, 0x45})

	other := []byte{0x01, 0x23, 0x01, 0x23, 0x45, 0x67}
	m := NewNibbles(other)

	assert.Equal(t, uint(4), n.CommonPrefix(m))
	assert.Equal(t, uint(4), m.CommonPrefix(n))
	assert.Equal(t, uint(3), n.Mid(1).CommonPrefix(m.Mid(1)))
	assert.Equal(t, uint(0), n.Mid(1).CommonPrefix(m.Mid(2)))
	assert.Equal(t, uint(6), n.CommonPrefix(m.Mid(4)))
	assert.False(t, n.StartsWith(m.Mid(4)))
	assert.True(t, m.Mid(4).StartsWith(n))
}

func TestNibbles_Compare(t *testing.T) {
	n := NewNibbles([]byte{1, 35})
	m := NewNibbles([]byte{1})

	assert.Equal(t, -1, m.Compare(n))
	assert.Equal(t, 1, n.Compare(m))

	n = NewNibbles([]byte{1, 35})
	m = NewNibbles([]byte{1, 35})

	assert.Equal(t, 0, m.Compare(n))

	n = NewNibbles([]byte{1, 35})
	m = NewNibbles([]byte{3, 35})
	assert.Equal(t, -1, n.Compare(m))
	assert.Equal(t, 1, m.Compare(n))
}

func TestNibbles_Advance(t *testing.T) {
	n := NewNibbles([]byte{1, 35})
	n.Advance(1)
	n.Advance(1)
	n.Advance(1)
	n.Advance(1)
	require.Panics(t, func() { n.Advance(1) })

	n = NewNibbles([]byte{1, 35})
	require.Panics(t, func() { n.Advance(5) })

	n = NewNibbles([]byte{1, 35})
	n.Advance(4)
	require.Panics(t, func() { n.Advance(1) })
}
