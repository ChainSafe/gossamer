package parachaintypes

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestBitVec_Bits(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			bv, err := NewBitVec(tt.bits)
			require.NoError(t, err)

			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}

func TestBitVec_Len(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			bv, err := NewBitVec(tt.bits)
			require.NoError(t, err)

			require.Equal(t, tt.expected, bv.Len())
		})
	}
}

func TestBitVec_PushBits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		initial  []bool
		toPush   []bool
		expected []bool
		wantErr  bool
	}{
		{
			name:     "push_to_empty_bitvec",
			initial:  []bool{},
			toPush:   []bool{true, false, true},
			expected: []bool{true, false, true},
			wantErr:  false,
		},
		{
			name:     "push_to_non_empty_bitvec",
			initial:  []bool{true, false},
			toPush:   []bool{true, true, false},
			expected: []bool{true, false, true, true, false},
			wantErr:  false,
		},
		{
			name:     "length_larger_than_allowed",
			initial:  []bool{},
			toPush:   make([]bool, MaxBitVecLength+1),
			expected: []bool{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bv, err := NewBitVec(tt.initial)
			require.NoError(t, err)

			err = bv.PushBits(tt.toPush)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}

func TestBitVec_SetBit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		initial  []bool
		index    uint
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bv, err := NewBitVec(tt.initial)
			require.NoError(t, err)

			err = bv.Set(tt.index, tt.value)
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
	t.Parallel()

	tests := []struct {
		name     string
		initial  []bool
		index    uint
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bv, err := NewBitVec(tt.initial)
			require.NoError(t, err)

			got, err := bv.Get(tt.index)
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
	t.Parallel()

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
		{
			name:     "medium_bitvec",
			bits:     make([]bool, 1000),
			expected: append([]byte{161, 15}, make([]byte, 125)...),
		},
		{
			name:     "max_length_bitvec",
			bits:     make([]bool, MaxBitVecLength),
			expected: append([]byte{254, 255, 255, 127}, make([]byte, 67108864)...),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bv, err := NewBitVec(tt.bits)
			require.NoError(t, err)

			encoded, err := scale.Marshal(bv)
			require.NoError(t, err)

			require.Equal(t, len(tt.expected), len(encoded))
			require.Equal(t, tt.expected, encoded)
		})
	}
}

func TestBitVec_UnmarshalSCALE(t *testing.T) {
	t.Parallel()

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
		{
			name:     "medium_bitvec",
			data:     append([]byte{161, 15}, make([]byte, 125)...),
			expected: make([]bool, 1000),
			wantErr:  false,
		},
		{
			name:     "medium_bitvec_with_extra_bytes",
			data:     append(append([]byte{161, 15}, make([]byte, 125)...), []byte{1, 1, 1, 1, 1, 1, 1, 1, 1}...),
			expected: make([]bool, 1000),
			wantErr:  false,
		},
		{
			name:     "max_length_bitvec",
			data:     append([]byte{254, 255, 255, 127}, make([]byte, 67108864)...),
			expected: make([]bool, MaxBitVecLength),
			wantErr:  false,
		},
		{
			name:     "length_exceeds_max_length",
			data:     append([]byte{254, 255, 255, 128}, make([]byte, 67108865)...),
			expected: []bool{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bv := BitVec{}
			err := scale.Unmarshal(tt.data, &bv)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, len(tt.expected), bv.Len())
			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}

func TestBitVec_ExtendByByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		initial   []bool
		byteToAdd byte
		expected  []bool
		wantErr   bool
	}{
		{
			name:      "extend_empty_bitvec",
			initial:   []bool{},
			byteToAdd: byte(10),
			expected:  []bool{false, true, false, true, false, false, false, false},
			wantErr:   false,
		},
		{
			name:      "extend_non_empty_bitvec",
			initial:   []bool{true, false},
			byteToAdd: 0b11001100,
			expected:  []bool{true, false, false, false, true, true, false, false, true, true},
			wantErr:   false,
		},
		{
			name:      "extend_with_all_bits_set",
			initial:   []bool{true, false, true},
			byteToAdd: 0b11111111,
			expected:  []bool{true, false, true, true, true, true, true, true, true, true, true},
			wantErr:   false,
		},
		{
			name:      "extend_with_no_bits_set",
			initial:   []bool{true, false, true},
			byteToAdd: 0b00000000,
			expected:  []bool{true, false, true, false, false, false, false, false, false, false, false},
			wantErr:   false,
		},
		{
			name:      "length_exceeds_max_length",
			initial:   make([]bool, MaxBitVecLength),
			byteToAdd: 0b10101010,
			expected:  make([]bool, MaxBitVecLength),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bv, err := NewBitVec(tt.initial)
			require.NoError(t, err)

			err = bv.ExtendByByte(tt.byteToAdd)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, len(tt.expected), bv.Len())
			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}
func TestBitVec_IsEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		bv1Bits        []bool
		bv2Bits        []bool
		expectedResult bool
	}{
		{
			name:           "empty_bitvecs_are_equal",
			bv1Bits:        []bool{},
			bv2Bits:        []bool{},
			expectedResult: true,
		},
		{
			name:           "single_bit_vectors_equal",
			bv1Bits:        []bool{true},
			bv2Bits:        []bool{true},
			expectedResult: true,
		},
		{
			name:           "single_bit_vectors_not_equal",
			bv1Bits:        []bool{true},
			bv2Bits:        []bool{false},
			expectedResult: false,
		},
		{
			name:           "different_lengths_not_equal",
			bv1Bits:        []bool{true, false},
			bv2Bits:        []bool{true},
			expectedResult: false,
		},
		{
			name:           "complete_byte_equal",
			bv1Bits:        []bool{true, false, true, true, false, false, true, false},
			bv2Bits:        []bool{true, false, true, true, false, false, true, false},
			expectedResult: true,
		},
		{
			name:           "complete_byte_not_equal",
			bv1Bits:        []bool{true, false, true, true, false, false, true, false},
			bv2Bits:        []bool{true, false, true, true, false, false, false, false},
			expectedResult: false,
		},
		{
			name:           "partial_byte_equal",
			bv1Bits:        []bool{true, false, true},
			bv2Bits:        []bool{true, false, true},
			expectedResult: true,
		},
		{
			name:           "partial_byte_not_equal",
			bv1Bits:        []bool{true, false, true},
			bv2Bits:        []bool{true, true, true},
			expectedResult: false,
		},
		{
			name: "multiple_complete_bytes_equal",
			bv1Bits: []bool{true, false, true, true, false, false, true, false, true, true, false, false, true, true,
				false, false},
			bv2Bits: []bool{true, false, true, true, false, false, true, false, true, true, false, false, true, true,
				false, false},
			expectedResult: true,
		},
		{
			name: "multiple_complete_bytes_not_equal",
			bv1Bits: []bool{true, false, true, true, false, false, true, false, true, true, false, false, true, true,
				false, false},
			bv2Bits: []bool{true, false, true, true, false, false, true, false, true, true, false, false, true, true,
				false, true},
			expectedResult: false,
		},
		{
			name:           "multiple_bytes_with_partial_byte_equal",
			bv1Bits:        []bool{true, false, true, true, false, false, true, false, true, true, false},
			bv2Bits:        []bool{true, false, true, true, false, false, true, false, true, true, false},
			expectedResult: true,
		},
		{
			name:           "multiple_bytes_with_partial_byte_not_equal",
			bv1Bits:        []bool{true, false, true, true, false, false, true, false, true, true, false},
			bv2Bits:        []bool{true, false, true, true, false, false, true, false, true, true, true},
			expectedResult: false,
		},
		{
			name:           "error_on_invalid_bitvec_creation",
			bv1Bits:        nil,
			bv2Bits:        []bool{true, false, true},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bv1, err := NewBitVec(tt.bv1Bits)
			require.NoError(t, err)

			bv2, err := NewBitVec(tt.bv2Bits)
			require.NoError(t, err)

			require.Equal(t, tt.expectedResult, bv1.IsEqual(&bv2))
			require.Equal(t, tt.expectedResult, bv2.IsEqual(&bv1))
		})
	}
}

func TestBitVec_CountOnes(t *testing.T) {
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
			name:     "single_bit_set",
			bits:     []bool{true},
			expected: 1,
		},
		{
			name:     "no_bits_set",
			bits:     []bool{false, false, false},
			expected: 0,
		},
		{
			name:     "all_bits_set",
			bits:     []bool{true, true, true},
			expected: 3,
		},
		{
			name:     "mixed_bits_within_single_byte",
			bits:     []bool{true, false, true, false, true},
			expected: 3,
		},
		{
			name: "complete_byte_plus_partial_byte",
			bits: []bool{
				true, false, true, true, false, true, true, true, // complete byte
				true, false, true, // partial byte
			},
			expected: 8,
		},
		{
			name: "multiple_complete_bytes",
			bits: []bool{
				true, true, true, true, false, false, false, false, // first byte
				true, true, true, true, true, true, true, true, // second byte
			},
			expected: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv, err := NewBitVec(tt.bits)
			require.NoError(t, err)

			got := bv.CountOnes()
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestBitVec_Mask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bvBits   []bool
		maskBits []bool
		expected []bool
	}{
		{
			name:     "same_length_bitvecs",
			bvBits:   []bool{true, true, false, true, true},
			maskBits: []bool{true, false, true, false, true},
			expected: []bool{false, true, false, true, false},
		},
		{
			name:     "bv_longer_than_mask",
			bvBits:   []bool{true, true, false, true, true},
			maskBits: []bool{true, false, true},
			expected: []bool{false, true, false, true, true},
		},
		{
			name:     "bv_shorter_than_mask",
			bvBits:   []bool{true, true, false},
			maskBits: []bool{true, false, true, false, true},
			expected: []bool{false, true, false},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bv, err := NewBitVec(tt.bvBits)
			require.NoError(t, err)

			mask, err := NewBitVec(tt.maskBits)
			require.NoError(t, err)

			bv.Mask(mask)
			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}
