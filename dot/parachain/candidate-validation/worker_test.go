package candidatevalidation

import (
	"testing"
	"time"

	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_worker_executeRequest(t *testing.T) {
	candidateReceipt, validationCode := createTestCandidateReceiptAndValidationCodeWParaId(t, 1000)

	validationRuntime, err := parachain.SetupVM(validationCode)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	expectedValidationResult := &ValidationResult{
		ValidResult: &Valid{
			CandidateCommitments: parachaintypes.CandidateCommitments{
				HeadData: parachaintypes.HeadData{Data: []byte{2, 0, 0, 0, 0, 0, 0, 0, 123, 207, 206, 8, 219, 227,
					136, 82, 236, 169, 14, 100, 45, 100, 31, 177, 154, 160, 220, 245, 59, 106, 76, 168, 122, 109,
					164, 169, 22, 46, 144, 39, 103, 92, 31, 78, 66, 72, 252, 64, 24, 194, 129, 162, 128, 1, 77, 147,
					200, 229, 189, 242, 111, 198, 236, 139, 16, 143, 19, 245, 113, 233, 138, 210}},
				ProcessedDownwardMessages: 0,
				HrmpWatermark:             1,
			},
			PersistedValidationData: parachaintypes.PersistedValidationData{
				ParentHead: parachaintypes.HeadData{Data: []byte{1, 0, 0, 0, 0, 0, 0, 0, 1,
					2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7,
					8, 9, 0, 1, 2, 48, 246, 146, 178, 86, 226, 64, 9,
					188, 179, 77, 14, 232, 77, 167, 60, 41, 138, 250, 204, 9, 36, 224, 17, 5, 226, 235,
					15, 1, 168, 127, 226}},
				RelayParentNumber: 1,
				RelayParentStorageRoot: common.MustHexToHash(
					"0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
				MaxPovSize: 2048,
			},
		},
	}

	mockValidationInstance := NewMockValidatorInstance(ctrl)
	mockValidationInstance.EXPECT().ValidateBlock(gomock.Any()).DoAndReturn(func(parachain.
		ValidationParameters) (*parachain.ValidationResult, error) {
		time.Sleep(3 * time.Second) // sleep to simulate execution time
		return &parachain.ValidationResult{
			HeadData: parachaintypes.HeadData{Data: []byte{2, 0, 0, 0, 0, 0, 0, 0, 123, 207, 206, 8, 219, 227,
				136, 82, 236, 169, 14, 100, 45, 100, 31, 177, 154, 160, 220, 245, 59, 106, 76, 168, 122, 109,
				164, 169, 22, 46, 144, 39, 103, 92, 31, 78, 66, 72, 252, 64, 24, 194, 129, 162, 128, 1, 77, 147,
				200, 229, 189, 242, 111, 198, 236, 139, 16, 143, 19, 245, 113, 233, 138, 210}},
			ProcessedDownwardMessages: 0,
			HrmpWatermark:             1,
		}, nil
	}).Times(2)

	candidateReceiptCommitmentsMismatch := candidateReceipt
	candidateReceiptCommitmentsMismatch.CommitmentsHash = common.MustHexToHash(
		"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	// NOTE: adder parachain internally compares postState with bd.State in it's validate_block,
	// so following is necessary.
	encodedState, err := scale.Marshal(uint64(1))
	require.NoError(t, err)
	postState, err := common.Keccak256(encodedState)
	require.NoError(t, err)

	hd, err := scale.Marshal(HeadDataInAdderParachain{
		Number:     uint64(1),
		ParentHash: common.MustHexToHash("0x0102030405060708090001020304050607080900010203040506070809000102"),
		PostState:  postState,
	})
	require.NoError(t, err)

	blockData, err := scale.Marshal(BlockDataInAdderParachain{
		State: uint64(1),
		Add:   uint64(1),
	})
	require.NoError(t, err)

	timeoutKind := parachaintypes.NewPvfExecTimeoutKind()
	err = timeoutKind.SetValue(parachaintypes.Approval{})
	require.NoError(t, err)

	commitmentsHashMismatch := CommitmentsHashMismatch
	timeout := Timeout

	tests := map[string]struct {
		instance parachain.ValidatorInstance
		task     *workerTask
		want     *ValidationResult
	}{
		"commitments_hash_mismatch": {
			instance: validationRuntime,
			task: &workerTask{
				work: parachain.ValidationParameters{
					ParentHeadData:         parachaintypes.HeadData{Data: hd},
					BlockData:              blockData,
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
				},
				maxPoVSize:       uint32(2048),
				candidateReceipt: &candidateReceiptCommitmentsMismatch,
			},
			want: &ValidationResult{
				InvalidResult: &commitmentsHashMismatch,
			},
		},
		"execution_timeout": {
			instance: mockValidationInstance,
			task: &workerTask{
				candidateReceipt: &candidateReceipt,
			},
			want: &ValidationResult{
				InvalidResult: &timeout,
			},
		},
		"long_timeout_ok": {
			instance: mockValidationInstance,
			task: &workerTask{
				work: parachain.ValidationParameters{
					ParentHeadData:         parachaintypes.HeadData{Data: hd},
					BlockData:              blockData,
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
				},
				candidateReceipt: &candidateReceipt,
				timeoutKind:      timeoutKind,
				maxPoVSize:       2048,
			},
			want: expectedValidationResult,
		},
		"happy_path": {
			instance: validationRuntime,
			task: &workerTask{
				work: parachain.ValidationParameters{
					ParentHeadData:         parachaintypes.HeadData{Data: hd},
					BlockData:              blockData,
					RelayParentNumber:      uint32(1),
					RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
				},
				maxPoVSize:       uint32(2048),
				candidateReceipt: &candidateReceipt,
				timeoutKind:      parachaintypes.PvfExecTimeoutKind{},
			},
			want: expectedValidationResult,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			w := &worker{
				instance:    tt.instance,
				isProcessed: make(map[parachaintypes.CandidateHash]*ValidationResult),
			}
			got, err := w.executeRequest(tt.task)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
