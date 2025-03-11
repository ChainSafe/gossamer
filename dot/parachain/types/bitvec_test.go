package parachaintypes

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/stretchr/testify/require"
)

func TestBitVec_Bits(t *testing.T) {
	tests := []struct {
		name     string
		bits     []bool
		expected []bool
	}{
		{
			name:     "empty bitvec",
			bits:     []bool{},
			expected: []bool{},
		},
		{
			name:     "single bit true",
			bits:     []bool{true},
			expected: []bool{true},
		},
		{
			name:     "single bit false",
			bits:     []bool{false},
			expected: []bool{false},
		},
		{
			name:     "multiple bits",
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
			name:     "empty bitvec",
			bits:     []bool{},
			expected: 0,
		},
		{
			name:     "single bit",
			bits:     []bool{true},
			expected: 1,
		},
		{
			name:     "multiple bits",
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
			name:     "push to empty bitvec",
			initial:  []bool{},
			toPush:   []bool{true, false, true},
			expected: []bool{true, false, true},
		},
		{
			name:     "push to non-empty bitvec",
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
		index    int
		value    bool
		expected []bool
		wantErr  bool
	}{
		{
			name:     "set bit within bounds",
			initial:  []bool{true, false, true},
			index:    1,
			value:    true,
			expected: []bool{true, true, true},
			wantErr:  false,
		},
		{
			name:     "set bit out of bounds",
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
			if (err != nil) != tt.wantErr {
				t.Errorf("SetBit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.expected, bv.Bits())
		})
	}
}

func TestBitVec_GetBit(t *testing.T) {
	tests := []struct {
		name     string
		initial  []bool
		index    int
		expected bool
		wantErr  bool
	}{
		{
			name:     "get bit within bounds",
			initial:  []bool{true, false, true},
			index:    1,
			expected: false,
			wantErr:  false,
		},
		{
			name:     "get bit out of bounds",
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
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBit() error = %v, wantErr %v", err, tt.wantErr)
				return
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
			name:     "empty bitvec",
			bits:     []bool{},
			expected: []byte{0},
		},
		{
			name:     "single byte bitvec",
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
			name:     "empty bitvec",
			data:     []byte{0},
			expected: []bool{},
			wantErr:  false,
		},
		{
			name:     "single byte bitvec",
			data:     []byte{32, 0b01001101},
			expected: []bool{true, false, true, true, false, false, true, false},
			wantErr:  false,
		},
		{
			name:     "invalid data",
			data:     []byte{255},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv := BitVec{}
			err := scale.Unmarshal(tt.data, &bv)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalSCALE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				require.Equal(t, tt.expected, bv.Bits())
			}
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
			name:      "extend empty bitvec",
			initial:   []bool{},
			byteToAdd: 0b10101010,
			expected:  []bool{false, true, false, true, false, true, false, true},
		},
		{
			name:      "extend non-empty bitvec",
			initial:   []bool{true, false},
			byteToAdd: 0b11001100,
			expected:  []bool{true, false, false, false, true, true, false, false, true, true},
		},
		{
			name:      "extend with all bits set",
			initial:   []bool{true, false, true},
			byteToAdd: 0b11111111,
			expected:  []bool{true, false, true, true, true, true, true, true, true, true, true},
		},
		{
			name:      "extend with no bits set",
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

func TestYup(t *testing.T) {

	bv := NewBitVec([]bool{})

	number1 := uint32(255)

	// number1Byte := byte(number1)
	if math.MaxUint8 >= number1 {
		bv.ExtendByByte(byte(number1))
	} else {
		println("number1 is too large")
	}

	number2 := uint32(256)
	if math.MaxUint8 >= number2 {
		bv.ExtendByByte(byte(number2))
	} else {
		println("number2 is too large")
	}
}
