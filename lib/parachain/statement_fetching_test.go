package parachain

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestEncodeStatementFetchingRequest(t *testing.T) {
	testHash := common.MustHexToHash("0x677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c19")
	statementFetchingRequest := StatementFetchingRequest{
		RelayParent:   testHash,
		CandidateHash: CandidateHash{Value: testHash},
	}

	actualEncode, err := statementFetchingRequest.Encode()
	require.NoError(t, err)

	// rust code to find expectedEncode.
	// fn statement_request() {
	// 	let test_hash: H256 = H256::from_str(
	// 	"0x677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c19"
	// 	).unwrap();
	// 	let statement_fetching_request = StatementFetchingRequest{
	// 		relay_parent: test_hash,
	// 		candidate_hash: CandidateHash(test_hash)
	// 	};
	// 	println!(
	// 		"statement_fetching_request encode => {:?}\n\n",
	// 		statement_fetching_request.encode()
	// 	);
	// }
	expextedEncode := common.MustHexToBytes(testDataStatement["hexOfStatementFetchingRequest"])
	require.Equal(t, expextedEncode, actualEncode)
}

func TestStatementFetchingResponse(t *testing.T) {
	t.Parallel()

	testHash := common.MustHexToHash("0x677811d2f3ded2489685468dbdb2e4fa280a249fba9356acceb2e823820e2c19")

	var collatorID CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	var collatorSignature CollatorSignature
	tempSignature := common.MustHexToBytes(testDataStatement["collatorSignature"])
	copy(collatorSignature[:], tempSignature)

	missingDataInStatement := MissingDataInStatement{
		Descriptor: CandidateDescriptor{
			ParaID:                      uint32(1),
			RelayParent:                 testHash,
			Collator:                    collatorID,
			PersistedValidationDataHash: testHash,
			PovHash:                     testHash,
			ErasureRoot:                 testHash,
			Signature:                   collatorSignature,
			ParaHead:                    testHash,
			ValidationCodeHash:          ValidationCodeHash(testHash),
		},
		Commitments: CandidateCommitments{
			UpwardMessages:            []UpwardMessage{{1, 2, 3}},
			NewValidationCode:         &ValidationCode{1, 2, 3},
			HeadData:                  headData{1, 2, 3},
			ProcessedDownwardMessages: uint32(5),
			HrmpWatermark:             uint32(0),
		},
	}

	encodedValue := common.MustHexToBytes(testDataStatement["hexOfStatementFetchingResponse"])

	t.Run("encode_statement_fetching_response", func(t *testing.T) {
		t.Parallel()

		response := NewStatementFetchingResponse()
		err := response.Set(missingDataInStatement)
		require.NoError(t, err)

		actualEncode, err := response.Encode()
		require.NoError(t, err)

		require.Equal(t, encodedValue, actualEncode)
	})

	t.Run("Decode_statement_fetching_response", func(t *testing.T) {
		t.Parallel()

		response := NewStatementFetchingResponse()
		err := response.Decode(encodedValue)
		require.NoError(t, err)

		actualData, err := response.Value()
		require.NoError(t, err)

		require.EqualValues(t, missingDataInStatement, actualData)
	})
}
