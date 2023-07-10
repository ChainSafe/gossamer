package parachain

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestEncodePoVFetchingRequest(t *testing.T) {
	poVFetchingRequest := PoVFetchingRequest{
		CandidateHash: CandidateHash{
			common.MustHexToHash("0x677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c19"),
		},
	}

	actualEncode, err := poVFetchingRequest.Encode()
	require.NoError(t, err)

	expextedEncode := common.MustHexToBytes("0x677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c19")
	require.Equal(t, expextedEncode, actualEncode)

}

func TestPoVFetchingResponse(t *testing.T) {
	t.Parallel()

	testBytes := common.MustHexToBytes("0x677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c19")
	testCases := []struct {
		name        string
		value       any
		encodeValue []byte
	}{
		{
			name:        "PoV",
			value:       PoV{BlockData: testBytes},
			encodeValue: common.MustHexToBytes("0x0080677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c19"),
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
				err := povFetchingResponse.SetValue(c.value)
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
