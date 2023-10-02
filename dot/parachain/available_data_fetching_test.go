// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEncodeAvailableDataFetchingRequest(t *testing.T) {
	availableDataFetchingRequest := AvailableDataFetchingRequest{
		CandidateHash: parachaintypes.CandidateHash{
			Value: common.MustHexToHash("0x677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c19"),
		},
	}

	actualEncode, err := availableDataFetchingRequest.Encode()
	require.NoError(t, err)

	expextedEncode := common.MustHexToBytes("0x677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c19")
	require.Equal(t, expextedEncode, actualEncode)
}

func TestAvailableDataFetchingResponse(t *testing.T) {
	t.Parallel()

	testHash := common.MustHexToHash("0x677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c19")
	testBytes := testHash.ToBytes()
	availableData := AvailableData{
		PoV: parachaintypes.PoV{BlockData: testBytes},
		ValidationData: PersistedValidationData{
			ParentHead:             testBytes,
			RelayParentNumber:      parachaintypes.BlockNumber(4),
			RelayParentStorageRoot: testHash,
			MaxPovSize:             6,
		},
	}

	testCases := []struct {
		name        string
		value       scale.VaryingDataTypeValue
		encodeValue []byte
	}{
		{
			name:        "AvailableData",
			value:       availableData,
			encodeValue: common.MustHexToBytes("0x0080677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c1980677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c1904000000677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c1906000000"), //nolint:lll
		},
		{
			name:        "NoSuchData",
			value:       NoSuchData{},
			encodeValue: []byte{1},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			t.Run("encode", func(t *testing.T) {
				t.Parallel()

				availableDataFetchingResponse := NewAvailableDataFetchingResponse()
				err := availableDataFetchingResponse.Set(c.value)
				require.NoError(t, err)

				actualEncode, err := availableDataFetchingResponse.Encode()
				require.NoError(t, err)

				require.Equal(t, c.encodeValue, actualEncode)
			})

			t.Run("decode", func(t *testing.T) {
				t.Parallel()

				availableDataFetchingResponse := NewAvailableDataFetchingResponse()
				err := availableDataFetchingResponse.Decode(c.encodeValue)
				require.NoError(t, err)

				actualData, err := availableDataFetchingResponse.Value()
				require.NoError(t, err)

				require.EqualValues(t, c.value, actualData)
			})

		})
	}
}
