package parachain

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEncodePoVFetchingRequest(t *testing.T) {
	poVFetchingRequest := PoVFetchingRequest{
		CandidateHash: CandidateHash{Value: getDummyHash(6)},
	}

	actualEncode, err := poVFetchingRequest.Encode()
	require.NoError(t, err)

	expextedEncode := common.MustHexToBytes("0x0606060606060606060606060606060606060606060606060606060606060606")
	require.Equal(t, expextedEncode, actualEncode)

}

func TestPoVFetchingResponse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		value       scale.VaryingDataTypeValue
		encodeValue []byte
	}{
		{
			name:        "PoV",
			value:       PoV(getDummyHash(6).ToBytes()),
			encodeValue: common.MustHexToBytes("0x00800606060606060606060606060606060606060606060606060606060606060606"),
		},
		{
			name:        "NoSuchPoV",
			value:       NoSuchPoV{},
			encodeValue: []byte{1},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			t.Run("encode", func(t *testing.T) {
				t.Parallel()

				povFetchingResponse := NewPoVFetchingResponse()
				err := povFetchingResponse.Set(c.value)
				require.NoError(t, err)

				actualEncode, err := povFetchingResponse.Encode()
				require.NoError(t, err)

				require.Equal(t, c.encodeValue, actualEncode)
			})

			t.Run("decode", func(t *testing.T) {
				t.Parallel()

				povFetchingResponse := NewPoVFetchingResponse()
				err := povFetchingResponse.Decode(c.encodeValue)
				require.NoError(t, err)

				actualData, err := povFetchingResponse.Value()
				require.NoError(t, err)

				require.EqualValues(t, c.value, actualData)
			})

		})
	}
}
