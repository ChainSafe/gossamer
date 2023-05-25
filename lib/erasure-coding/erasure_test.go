package erasure_coding

import (
	"github.com/klauspost/reedsolomon"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testData = []byte("this is a test of the erasure coding")
var expectedData = [][]byte{{116, 104, 105, 115}, {32, 105, 115, 32}, {97, 32, 116, 101}, {115, 116, 32, 111},
	{102, 32, 116, 104}, {101, 32, 101, 114}, {97, 115, 117, 114}, {101, 32, 99, 111}, {100, 105, 110, 103},
	{0, 0, 0, 0}, {133, 189, 154, 178}, {88, 245, 245, 220}, {59, 208, 165, 70}, {127, 213, 208, 179}}

// erasure data missing chunks
var missing2Chunks = [][]byte{{116, 104, 105, 115}, {32, 105, 115, 32}, {}, {115, 116, 32, 111},
	{102, 32, 116, 104}, {101, 32, 101, 114}, {}, {101, 32, 99, 111}, {100, 105, 110, 103},
	{0, 0, 0, 0}, {133, 189, 154, 178}, {88, 245, 245, 220}, {59, 208, 165, 70}, {127, 213, 208, 179}}
var missing3Chunks = [][]byte{{116, 104, 105, 115}, {32, 105, 115, 32}, {}, {115, 116, 32, 111},
	{}, {101, 32, 101, 114}, {}, {101, 32, 99, 111}, {100, 105, 110, 103}, {0, 0, 0, 0}, {133, 189, 154, 178},
	{88, 245, 245, 220}, {59, 208, 165, 70}, {127, 213, 208, 179}}
var missing5Chunks = [][]byte{{}, {}, {}, {115, 116, 32, 111},
	{}, {101, 32, 101, 114}, {}, {101, 32, 99, 111}, {100, 105, 110, 103}, {0, 0, 0, 0}, {133, 189, 154, 178},
	{88, 245, 245, 220}, {59, 208, 165, 70}, {127, 213, 208, 179}}

func TestObtainChunks(t *testing.T) {
	type args struct {
		validatorsQty int
		data          []byte
	}
	tests := map[string]struct {
		args          args
		expectedValue [][]byte
		expectedError error
	}{
		"happy path": {
			args: args{
				validatorsQty: 10,
				data:          testData,
			},
			expectedValue: expectedData,
		},
		"nil data": {
			args: args{
				validatorsQty: 10,
				data:          nil,
			},
			expectedError: reedsolomon.ErrShortData,
		},
		"not enough validators": {
			args: args{
				validatorsQty: 1,
				data:          testData,
			},
			expectedError: ErrNotEnoughValidators,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ObtainChunks(tt.args.validatorsQty, tt.args.data)
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedValue, got)
		})
	}
}

func TestReconstruct(t *testing.T) {
	type args struct {
		validatorsQty int
		chunks        [][]byte
	}
	tests := map[string]struct {
		args          args
		expected      [][]byte
		expectedError error
	}{
		"missing 2 chunks": {
			args: args{
				validatorsQty: 10,
				chunks:        missing2Chunks,
			},
			expected: expectedData,
		},
		"missing 2 chunks, validator qty 3": {
			args: args{
				validatorsQty: 3,
				chunks:        missing2Chunks,
			},
			expectedError: reedsolomon.ErrTooFewShards,
			expected:      expectedData,
		},
		"missing 3 chunks": {
			args: args{
				validatorsQty: 10,
				chunks:        missing3Chunks,
			},
			expected: expectedData,
		},
		"missing 5 chunks": {
			args: args{
				validatorsQty: 10,
				chunks:        missing5Chunks,
			},
			expected:      missing5Chunks,
			expectedError: reedsolomon.ErrTooFewShards,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := Reconstruct(tt.args.validatorsQty, tt.args.chunks)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, tt.args.chunks)
		})
	}
}
