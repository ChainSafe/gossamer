package parachain

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEncodeChunkFetchingRequest(t *testing.T) {
	chunkFetchingRequest := ChunkFetchingRequest{
		CandidateHash: CandidateHash{getDummyHash(4)},
		Index:         ValidatorIndex(8),
	}

	actualEncode, err := chunkFetchingRequest.Encode()
	require.NoError(t, err)

	expextedEncode := common.MustHexToBytes("0x040404040404040404040404040404040404040404040404040404040404040408000000")
	require.Equal(t, expextedEncode, actualEncode)
}

func TestChunkFetchingResponse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		value       scale.VaryingDataTypeValue
		encodeValue []byte
	}{
		{
			name: "chunkResponse",
			value: ChunkResponse{
				Chunk: getDummyHash(9).ToBytes(),
				Proof: [][]byte{getDummyHash(3).ToBytes()},
			},
			encodeValue: common.MustHexToBytes("0x0080090909090909090909090909090909090909090909090909090909090909090904800303030303030303030303030303030303030303030303030303030303030303"), //nolint:lll
		},
		{
			name:        "NoSuchChunk",
			value:       NoSuchChunk{},
			encodeValue: []byte{1},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			chunkFetchingResponse := NewChunkFetchingResponse()

			t.Run("encode", func(t *testing.T) {
				t.Parallel()

				err := chunkFetchingResponse.Set(c.value)
				require.NoError(t, err)

				actualEncode, err := chunkFetchingResponse.Encode()
				require.NoError(t, err)

				require.Equal(t, c.encodeValue, actualEncode)
			})

			t.Run("decode", func(t *testing.T) {
				t.Parallel()

				err := chunkFetchingResponse.Decode(c.encodeValue)
				require.NoError(t, err)

				actualData, err := chunkFetchingResponse.Value()
				require.NoError(t, err)

				require.EqualValues(t, c.value, actualData)
			})

		})
	}
}
