package parachain

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEncodeAvailableDataFetchingRequest(t *testing.T) {
	availableDataFetchingRequest := AvailableDataFetchingRequest{
		CandidateHash: CandidateHash{Value: getDummyHash(4)},
	}

	actualEncode, err := availableDataFetchingRequest.Encode()
	require.NoError(t, err)

	expextedEncode := common.MustHexToBytes("0x0404040404040404040404040404040404040404040404040404040404040404")
	require.Equal(t, expextedEncode, actualEncode)
}

func TestAvailableDataFetchingResponse(t *testing.T) {
	t.Parallel()

	availableData := AvailableData{
		PoV: getDummyHash(2).ToBytes(),
		ValidationData: PersistedValidationData{
			ParentHead:             getDummyHash(3).ToBytes(),
			RelayParentNumber:      BlockNumber(4),
			RelayParentStorageRoot: getDummyHash(5),
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
			encodeValue: common.MustHexToBytes("0x0080020202020202020202020202020202020202020202020202020202020202020280030303030303030303030303030303030303030303030303030303030303030304000000050505050505050505050505050505050505050505050505050505050505050506000000"), //nolint:lll
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
