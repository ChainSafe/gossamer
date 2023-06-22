package parachain

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestEncodeStatementFetchingRequest(t *testing.T) {
	testCases := []struct {
		name           string
		request        StatementFetchingRequest
		expectedEncode []byte
	}{
		{
			// expected encoding is generated by running rust test code:
			// fn statement_request() {
			// 	let hash4 = Hash::repeat_byte(4);
			// 	let statement_fetching_request = StatementFetchingRequest{
			// 		relay_parent: hash4,
			// 		candidate_hash: CandidateHash(hash4)
			// 	};
			// 	println!(
			// 		"statement_fetching_request encode => {:?}\n\n",
			// 		statement_fetching_request.encode()
			// 	);
			// }
			name: "all 4 in common.Hash",
			request: StatementFetchingRequest{
				RelayParent:   getDummyHash(4),
				CandidateHash: CandidateHash{Value: getDummyHash(4)},
			},
			expectedEncode: []byte{
				4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
				4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
				4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
			},
		},
		{
			name: "all 7 in common.Hash",
			request: StatementFetchingRequest{
				RelayParent:   getDummyHash(7),
				CandidateHash: CandidateHash{Value: getDummyHash(7)},
			},
			expectedEncode: []byte{
				7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
				7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
				7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
			},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			actualEncode, err := c.request.Encode()
			require.NoError(t, err)
			require.Equal(t, c.expectedEncode, actualEncode)
		})
	}
}

func TestStatementFetchingResponse(t *testing.T) {
	hash5 := getDummyHash(5)

	var collatorID CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	var collatorSignature CollatorSignature
	tempSignature := common.MustHexToBytes(testSDMHex["collatorSignature"])
	copy(collatorSignature[:], tempSignature)

	missingDataInStatement := MissingDataInStatement{
		Descriptor: CandidateDescriptor{
			ParaID:                      uint32(1),
			RelayParent:                 hash5,
			Collator:                    collatorID,
			PersistedValidationDataHash: hash5,
			PovHash:                     hash5,
			ErasureRoot:                 hash5,
			Signature:                   collatorSignature,
			ParaHead:                    hash5,
			ValidationCodeHash:          ValidationCodeHash(hash5),
		},
		Commitments: CandidateCommitments{
			UpwardMessages:            []UpwardMessage{{1, 2, 3}},
			NewValidationCode:         &ValidationCode{1, 2, 3},
			HeadData:                  headData{1, 2, 3},
			ProcessedDownwardMessages: uint32(5),
			HrmpWatermark:             uint32(0),
		},
	}

	EncodedValue := []byte{0, 1, 0, 0, 0, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 72, 33, 91, 157, 50, 38, 1, 229, 177, 169, 81, 100, 206, 160, 220, 70, 38, 245, 69, 249, 131, 67, 208, 127, 21, 81, 235, 149, 67, 196, 177, 71, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 198, 124, 185, 59, 240, 163, 111, 206, 227, 210, 157, 232, 166, 166, 154, 117, 150, 89, 104, 10, 207, 72, 100, 117, 224, 162, 85, 42, 95, 190, 216, 126, 69, 173, 206, 95, 41, 6, 152, 216, 89, 96, 149, 114, 43, 51, 89, 146, 39, 247, 70, 31, 81, 175, 134, 23, 200, 190, 116, 184, 148, 207, 27, 134, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 4, 12, 1, 2, 3, 0, 1, 12, 1, 2, 3, 12, 1, 2, 3, 5, 0, 0, 0, 0, 0, 0, 0}

	response := NewStatementFetchingResponse()

	err := response.Set(missingDataInStatement)
	require.NoError(t, err)

	t.Run("Encode StatementFetchingResponse", func(t *testing.T) {
		actualEncode, err := response.Encode()
		require.NoError(t, err)

		require.Equal(t, EncodedValue, actualEncode)
	})

	t.Run("Decode StatementFetchingResponse", func(t *testing.T) {
		err := response.Decode(EncodedValue)
		require.NoError(t, err)

		actualData, err := response.Value()
		require.NoError(t, err)

		require.EqualValues(t, missingDataInStatement, actualData)
	})
}
