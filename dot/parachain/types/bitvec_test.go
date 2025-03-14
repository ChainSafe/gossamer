package parachaintypes

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestBitVec_Bits(t *testing.T) {
	tests := []struct {
		name     string
		bits     []bool
		expected []bool
	}{
		{
			name:     "empty_bitvec",
			bits:     []bool{},
			expected: []bool{},
		},
		{
			name:     "single_bit_true",
			bits:     []bool{true},
			expected: []bool{true},
		},
		{
			name:     "single_bit_false",
			bits:     []bool{false},
			expected: []bool{false},
		},
		{
			name:     "multiple_bits",
			bits:     []bool{true, false, true, true, false, false, true, false},
			expected: []bool{true, false, true, true, false, false, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv := NewBitVec(tt.bits)
			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}

func TestBitVec_Len(t *testing.T) {
	tests := []struct {
		name     string
		bits     []bool
		expected int
	}{
		{
			name:     "empty_bitvec",
			bits:     []bool{},
			expected: 0,
		},
		{
			name:     "single_bit",
			bits:     []bool{true},
			expected: 1,
		},
		{
			name:     "multiple_bits",
			bits:     []bool{true, false, true, true, false, false, true, false},
			expected: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv := NewBitVec(tt.bits)
			require.Equal(t, tt.expected, bv.Len())
		})
	}
}

func TestBitVec_PushBits(t *testing.T) {
	tests := []struct {
		name     string
		initial  []bool
		toPush   []bool
		expected []bool
	}{
		{
			name:     "push_to_empty_bitvec",
			initial:  []bool{},
			toPush:   []bool{true, false, true},
			expected: []bool{true, false, true},
		},
		{
			name:     "push_to_non_empty_bitvec",
			initial:  []bool{true, false},
			toPush:   []bool{true, true, false},
			expected: []bool{true, false, true, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv := NewBitVec(tt.initial)
			bv.PushBits(tt.toPush)
			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}

func TestBitVec_SetBit(t *testing.T) {
	tests := []struct {
		name     string
		initial  []bool
		index    uint32
		value    bool
		expected []bool
		wantErr  bool
	}{
		{
			name:     "set_bit_within_bounds",
			initial:  []bool{true, false, true},
			index:    1,
			value:    true,
			expected: []bool{true, true, true},
			wantErr:  false,
		},
		{
			name:     "set_bit_out_of_bounds",
			initial:  []bool{true, false, true},
			index:    3,
			value:    true,
			expected: []bool{true, false, true},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv := NewBitVec(tt.initial)
			err := bv.SetBit(tt.index, tt.value)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}

func TestBitVec_GetBit(t *testing.T) {
	tests := []struct {
		name     string
		initial  []bool
		index    uint32
		expected bool
		wantErr  bool
	}{
		{
			name:     "get_bit_within_bounds",
			initial:  []bool{true, false, true},
			index:    1,
			expected: false,
			wantErr:  false,
		},
		{
			name:     "get_bit_out_of_bounds",
			initial:  []bool{true, false, true},
			index:    3,
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv := NewBitVec(tt.initial)
			got, err := bv.GetBit(tt.index)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expected, got)
		})
	}
}

func TestBitVec_MarshalSCALE(t *testing.T) {
	tests := []struct {
		name     string
		bits     []bool
		expected []byte
	}{
		{
			name:     "empty_bitvec",
			bits:     []bool{},
			expected: []byte{0},
		},
		{
			name:     "single_byte_bitvec",
			bits:     []bool{true, false, true, true, false, false, true, false},
			expected: []byte{32, 0b01001101},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv := NewBitVec(tt.bits)
			encoded, err := scale.Marshal(bv)
			require.NoError(t, err)
			require.Equal(t, tt.expected, encoded)
		})
	}
}

func TestBitVec_UnmarshalSCALE(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []bool
		wantErr  bool
	}{
		{
			name:     "empty_bitvec",
			data:     []byte{0},
			expected: []bool{},
			wantErr:  false,
		},
		{
			name:     "single_byte_bitvec",
			data:     []byte{32, 0b01001101},
			expected: []bool{true, false, true, true, false, false, true, false},
			wantErr:  false,
		},
		{
			name:     "invalid_data",
			data:     []byte{255},
			expected: []bool{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv := BitVec{}
			err := scale.Unmarshal(tt.data, &bv)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}

func TestBitVec_ExtendByByte(t *testing.T) {
	tests := []struct {
		name      string
		initial   []bool
		byteToAdd byte
		expected  []bool
	}{
		{
			name:      "extend_empty_bitvec",
			initial:   []bool{},
			byteToAdd: 0b10101010,
			expected:  []bool{false, true, false, true, false, true, false, true},
		},
		{
			name:      "extend_non_empty_bitvec",
			initial:   []bool{true, false},
			byteToAdd: 0b11001100,
			expected:  []bool{true, false, false, false, true, true, false, false, true, true},
		},
		{
			name:      "extend_with_all_bits_set",
			initial:   []bool{true, false, true},
			byteToAdd: 0b11111111,
			expected:  []bool{true, false, true, true, true, true, true, true, true, true, true},
		},
		{
			name:      "extend_with_no_bits_set",
			initial:   []bool{true, false, true},
			byteToAdd: 0b00000000,
			expected:  []bool{true, false, true, false, false, false, false, false, false, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv := NewBitVec(tt.initial)
			bv.ExtendByByte(tt.byteToAdd)
			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}

func TestBitVec_IsEqual(t *testing.T) {
	tests := []struct {
		name     string
		bv1Bits  []bool
		bv2Bits  []bool
		expected bool
	}{
		{
			name:     "empty_bitvecs_are_equal",
			bv1Bits:  []bool{},
			bv2Bits:  []bool{},
			expected: true,
		},
		{
			name:     "single_bit_vectors_equal",
			bv1Bits:  []bool{true},
			bv2Bits:  []bool{true},
			expected: true,
		},
		{
			name:     "single_bit_vectors_not_equal",
			bv1Bits:  []bool{true},
			bv2Bits:  []bool{false},
			expected: false,
		},
		{
			name:     "different_lengths_not_equal",
			bv1Bits:  []bool{true, false},
			bv2Bits:  []bool{true},
			expected: false,
		},
		{
			name:     "complete_byte_equal",
			bv1Bits:  []bool{true, false, true, true, false, false, true, false},
			bv2Bits:  []bool{true, false, true, true, false, false, true, false},
			expected: true,
		},
		{
			name:     "complete_byte_not_equal",
			bv1Bits:  []bool{true, false, true, true, false, false, true, false},
			bv2Bits:  []bool{true, false, true, true, false, false, false, false},
			expected: false,
		},
		{
			name:     "partial_byte_equal",
			bv1Bits:  []bool{true, false, true},
			bv2Bits:  []bool{true, false, true},
			expected: true,
		},
		{
			name:     "partial_byte_not_equal",
			bv1Bits:  []bool{true, false, true},
			bv2Bits:  []bool{true, true, true},
			expected: false,
		},
		{
			name:     "multiple_complete_bytes_equal",
			bv1Bits:  []bool{true, false, true, true, false, false, true, false, true, true, false, false, true, true, false, false},
			bv2Bits:  []bool{true, false, true, true, false, false, true, false, true, true, false, false, true, true, false, false},
			expected: true,
		},
		{
			name:     "multiple_complete_bytes_not_equal",
			bv1Bits:  []bool{true, false, true, true, false, false, true, false, true, true, false, false, true, true, false, false},
			bv2Bits:  []bool{true, false, true, true, false, false, true, false, true, true, false, false, true, true, false, true},
			expected: false,
		},
		{
			name:     "multiple_bytes_with_partial_byte_equal",
			bv1Bits:  []bool{true, false, true, true, false, false, true, false, true, true, false},
			bv2Bits:  []bool{true, false, true, true, false, false, true, false, true, true, false},
			expected: true,
		},
		{
			name:     "multiple_bytes_with_partial_byte_not_equal",
			bv1Bits:  []bool{true, false, true, true, false, false, true, false, true, true, false},
			bv2Bits:  []bool{true, false, true, true, false, false, true, false, true, true, true},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			bv1 := NewBitVec(tt.bv1Bits)
			bv2 := NewBitVec(tt.bv2Bits)

			if result := bv1.IsEqual(&bv2); result != tt.expected {
				t.Errorf("IsEqual() = %v, expected %v", result, tt.expected)
				t.Errorf("bv1: %v", bv1.Bits())
				t.Errorf("bv2: %v", bv2.Bits())
			}

			// Test symmetry: a.IsEqual(b) should be the same as b.IsEqual(a)
			if result := bv2.IsEqual(&bv1); result != tt.expected {
				t.Errorf("Symmetry test failed: bv2.IsEqual(bv1) = %v, expected %v", result, tt.expected)
				t.Errorf("bv1: %v", bv1.Bits())
				t.Errorf("bv2: %v", bv2.Bits())
			}
		})
	}
}
