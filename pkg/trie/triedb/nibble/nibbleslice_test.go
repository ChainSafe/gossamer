// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibble

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var testData = []byte{0x01, 0x23, 0x45}

func Test_Basic(t *testing.T) {
	t.Run("Nibble slice with offset 0", func(t *testing.T) {
		n := NewNibbleSlice(testData)

		require.Equal(t, 6, n.Len())
		require.False(t, n.IsEmpty())

		for i := 0; i < n.Len(); i++ {
			require.Equal(t, byte(i), n.At(i))
		}
	})

	t.Run("Nibble slice with offset 6", func(t *testing.T) {
		n := NewNibbleSliceWithOffset(testData, 6)
		require.True(t, n.IsEmpty())
	})

	t.Run("Nibble slice with offset 3", func(t *testing.T) {
		n := NewNibbleSliceWithOffset(testData, 3)
		require.Equal(t, 3, n.Len())
		for i := 0; i < n.Len(); i++ {
			require.Equal(t, byte(i)+3, n.At(i))
		}
	})
}

func Test_Mid(t *testing.T) {
	n := NewNibbleSlice(testData)
	t.Run("Mid 2", func(t *testing.T) {
		m := n.Mid(2)
		for i := 0; i < m.Len(); i++ {
			require.Equal(t, byte(i)+2, m.At(i))
		}
	})
	t.Run("Mid 3", func(t *testing.T) {
		m := n.Mid(3)
		for i := 0; i < m.Len(); i++ {
			require.Equal(t, byte(i)+3, m.At(i))
		}
	})
}

func Test_EncodedPre(t *testing.T) {
	n := NewNibbleSlice(testData)

	t.Run("Mid 0", func(t *testing.T) {
		m := n.Mid(0)
		expected := NibbleSlice{
			data:   []byte{0x01, 0x23, 0x45},
			offset: 0,
		}

		require.Equal(t, expected, m.ToStored())
	})

	t.Run("Mid 1", func(t *testing.T) {
		m := n.Mid(1)
		expected := NibbleSlice{
			data:   []byte{0x01, 0x23, 0x45},
			offset: 1,
		}

		require.Equal(t, expected, m.ToStored())
	})

	t.Run("Mid 2", func(t *testing.T) {
		m := n.Mid(2)
		expected := NibbleSlice{
			data:   []byte{0x23, 0x45},
			offset: 0,
		}

		require.Equal(t, expected, m.ToStored())
	})

	t.Run("Mid 3", func(t *testing.T) {
		m := n.Mid(3)
		expected := NibbleSlice{
			data:   []byte{0x23, 0x45},
			offset: 1,
		}

		require.Equal(t, expected, m.ToStored())
	})
}

func Test_Shared(t *testing.T) {
	n := NewNibbleSlice(testData)

	other := []byte{0x01, 0x23, 0x01, 0x23, 0x45, 0x67}
	m := NewNibbleSlice(other)

	require.Equal(t, 4, n.CommonPrefix(*m))
	require.Equal(t, 4, m.CommonPrefix(*n))
	require.Equal(t, 3, n.Mid(1).CommonPrefix(*m.Mid(1)))
	require.Equal(t, 0, n.Mid(1).CommonPrefix(*m.Mid(2)))
	require.Equal(t, 6, n.CommonPrefix(*m.Mid(4)))
	require.True(t, m.Mid(4).StartsWith(*n))
}

func Test_Compare(t *testing.T) {
	other := []byte{0x01, 0x23, 0x01, 0x23, 0x45}
	n := NewNibbleSlice(testData)
	m := NewNibbleSlice(other)

	require.False(t, n.Eq(*m))
	require.True(t, n.Eq(*m.Mid(4)))
}
