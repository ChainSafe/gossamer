// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resultBytes, err := common.HexToBytes(tt.in)
			require.NoError(t, err)

			b, err := scale.Marshal(tt.wantBitVec)
			require.NoError(t, err)
			require.Equal(t, resultBytes, b)

			bv := NewBitVec(nil)
			err = scale.Unmarshal(resultBytes, &bv)
			require.NoError(t, err)
			require.Equal(t, tt.wantBitVec.bits, bv.bits)

			b, err = scale.Marshal(bv)
			require.NoError(t, err)
			require.Equal(t, resultBytes, b)
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bv := BitVec{tt.in}
			bytes := bv.bitsToBytes()
			require.Equal(t, tt.want, bytes)
		})
	}
}
