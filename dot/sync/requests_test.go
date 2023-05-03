package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/stretchr/testify/require"
)

func TestAscendingBlockRequest(t *testing.T) {
	one := uint32(1)
	three := uint32(3)
	maxResponseSize := uint32(maxResponseSize)
	cases := map[string]struct {
		startNumber, targetNumber   uint
		expectedBlockRequestMessage []*network.BlockRequestMessage
	}{
		"start_greater_than_target": {
			startNumber:                 10,
			targetNumber:                0,
			expectedBlockRequestMessage: []*network.BlockRequestMessage{},
		},

		"no_difference_between_start_and_target": {
			startNumber:  10,
			targetNumber: 10,
			expectedBlockRequestMessage: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(10)),
					Direction:     network.Ascending,
					Max:           &one,
				},
			},
		},

		"requesting_128_blocks": {
			startNumber:  0,
			targetNumber: 128,
			expectedBlockRequestMessage: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(0)),
					Direction:     network.Ascending,
					Max:           &maxResponseSize,
				},
			},
		},

		"requesting_4_chunks_of_128_blocks": {
			startNumber:  0,
			targetNumber: 512, // 128 * 4
			expectedBlockRequestMessage: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(0)),
					Direction:     network.Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(128)),
					Direction:     network.Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(256)),
					Direction:     network.Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(384)),
					Direction:     network.Ascending,
					Max:           &maxResponseSize,
				},
			},
		},

		"requesting_4_chunks_of_128_plus_3_blocks": {
			startNumber:  0,
			targetNumber: 512 + 3, // 128 * 4
			expectedBlockRequestMessage: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(0)),
					Direction:     network.Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(128)),
					Direction:     network.Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(256)),
					Direction:     network.Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(384)),
					Direction:     network.Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(512)),
					Direction:     network.Ascending,
					Max:           &three,
				},
			},
		},
	}

	for tname, tt := range cases {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			requests := ascedingBlockRequests(tt.startNumber, tt.targetNumber, bootstrapRequestData)
			require.Equal(t, requests, tt.expectedBlockRequestMessage)
		})
	}
}
