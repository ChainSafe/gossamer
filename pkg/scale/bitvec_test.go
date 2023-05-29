// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestBitVec(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		in         string
		wantBitVec BitVec
		wantErr    bool
	}{
		{
			name:       "empty_bitvec",
			in:         "0x00",
			wantBitVec: NewBitVec(nil),
			wantErr:    false,
		},
		{
			name:       "1_byte",
			in:         "0x2055",
			wantBitVec: NewBitVec([]bool{true, false, true, false, true, false, true, false}),
			wantErr:    false,
		},
		{
			name: "4_bytes",
			in:   "0x645536aa01",
			wantBitVec: NewBitVec([]bool{
				true, false, true, false, true, false, true, false,
				false, true, true, false, true, true, false, false,
				false, true, false, true, false, true, false, true,
				true,
			}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resultBytes, err := common.HexToBytes(tt.in)
			require.NoError(t, err)

			bv := NewBitVec(nil)
			err = Unmarshal(resultBytes, &bv)
			require.NoError(t, err)

			require.Equal(t, tt.wantBitVec.Size(), bv.Size())
			require.Equal(t, tt.wantBitVec.Size(), bv.Size())

			b, err := Marshal(bv)
			require.NoError(t, err)
			require.Equal(t, resultBytes, b)
		})
	}
}

func TestBitVecBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      BitVec
		want    []byte
		wantErr bool
	}{
		{
			name:    "empty_bitvec",
			in:      NewBitVec(nil),
			want:    []byte(nil),
			wantErr: false,
		},
		{
			name:    "1_byte",
			in:      NewBitVec([]bool{true, false, true, false, true, false, true, false}),
			want:    []byte{0x55},
			wantErr: false,
		},
		{
			name: "4_bytes",
			in: NewBitVec([]bool{
				true, false, true, false, true, false, true, false,
				false, true, true, false, true, true, false, false,
				false, true, false, true, false, true, false, true,
				true,
			}),
			want:    []byte{0x55, 0x36, 0xaa, 0x1},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.in.Bytes())
		})
	}
}

func TestBitVecBytesToBits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      []byte
		want    []bool
		wantErr bool
	}{
		{
			name:    "empty",
			in:      []byte(nil),
			want:    []bool(nil),
			wantErr: false,
		},
		{
			name:    "1_byte",
			in:      []byte{0x55},
			want:    []bool{true, false, true, false, true, false, true, false},
			wantErr: false,
		},
		{
			name: "4_bytes",
			in:   []byte{0x55, 0x36, 0xaa, 0x1},
			want: []bool{
				true, false, true, false, true, false, true, false,
				false, true, true, false, true, true, false, false,
				false, true, false, true, false, true, false, true,
				true, false, false, false, false, false, false, false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, bytesToBits(tt.in, uint(len(tt.in)*byteSize)))
		})
	}
}

func TestBitVecBitsToBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      []bool
		want    []byte
		wantErr bool
	}{
		{
			name:    "empty",
			in:      []bool(nil),
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "1_byte",
			in:      []bool{true, false, true, false, true, false, true, false},
			want:    []byte{0x55},
			wantErr: false,
		},
		{
			name: "4_bytes",
			in: []bool{
				true, false, true, false, true, false, true, false,
				false, true, true, false, true, true, false, false,
				false, true, false, true, false, true, false, true,
				true,
			},
			want: []byte{0x55, 0x36, 0xaa, 0x1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, bitsToBytes(tt.in))
		})
	}
}
