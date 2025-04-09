package parachaintypes

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func candidateDescriptorV1(t *testing.T) CandidateDescriptor {
	t.Helper()

	return CandidateDescriptor{
		ParaID:                      1000,
		RelayParent:                 common.MustHexToHash("0xded542bacb3ca6c033a57676f94ae7c8f36834511deb44e3164256fd3b1c0de0"), //nolint:lll
		Collator:                    collatorID(t),
		PersistedValidationDataHash: common.MustHexToHash("0x690d8f252ef66ab0f969c3f518f90012b849aa5ac94e1752c5e5ae5a8996de37"), //nolint:lll
		PovHash:                     common.MustHexToHash("0xe7df1126ac4b4f0fb1bc00367a12ec26ca7c51256735a5e11beecdc1e3eca274"), //nolint:lll
		ErasureRoot:                 common.MustHexToHash("0xc07f658163e93c45a6f0288d229698f09c1252e41076f4caa71c8cbc12f118a1"), //nolint:lll
		ParaHead:                    common.MustHexToHash("0x9a8a7107426ef873ab89fc8af390ec36bdb2f744a9ff71ad7f18a12d55a7f4f5"), //nolint:lll
	}
}

func collatorID(t *testing.T) CollatorID {
	t.Helper()

	return CollatorID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
}

func TestCandidateDescriptorV2_rebuildCollatorField(t *testing.T) {
	t.Parallel()

	descriptorV1 := CandidateDescriptor{
		Collator: collatorID(t),
	}

	descriptorV2 := descriptorV1.V2()
	collatorField := descriptorV2.RebuildCollatorField()

	require.Equal(t, descriptorV1.Collator[:], collatorField[:])

}

func TestCandidateDescriptorV2_BackwardCompatibility(t *testing.T) {
	t.Parallel()

	descriptorV1 := candidateDescriptorV1(t)
	descriptorV2 := descriptorV1.V2()

	encodedV1, err := scale.Marshal(descriptorV1)
	require.NoError(t, err)

	encodedV2, err := scale.Marshal(descriptorV2)
	require.NoError(t, err)

	require.Equal(t, encodedV1, encodedV2)
}

func TestUMPSignal(t *testing.T) {
	t.Parallel()

	ump := UMPSignal{}
	err := ump.SetValue(SelectCore{10, 20})
	require.NoError(t, err)

	encoded, err := scale.Marshal(ump)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 10, 20}, encoded)

	ump2 := UMPSignal{}
	err = scale.Unmarshal(encoded, &ump2)
	require.NoError(t, err)
	require.Equal(t, ump, ump2)
}

func TestCommittedCandidateReceiptV2_CheckCoreIndex(t *testing.T) {
	t.Parallel()

	// Helper to create test CommittedCandidateReceiptV2
	newTestReceipt := func(
		version CandidateDescriptorVersion, coreIndex uint16, paraID ParaID,
	) CommittedCandidateReceiptV2 {
		desc := CandidateDescriptorV2{
			ParaID:    paraID,
			CoreIndex: coreIndex,
		}

		// Set version
		if version == CandidateDescriptorVersion1 {
			desc.Reserved1[0] = 1 // Non-zero makes it version 1
		}

		return CommittedCandidateReceiptV2{
			Descriptor:  desc,
			Commitments: CandidateCommitments{},
		}
	}

	// Helper to create mock claim queue.
	// offset is a optional parameter to specify the offset for the claim queue.
	mockTransposedClaimQueue := func(paraID ParaID, cores []CoreIndex, offset ...uint8) TransposedClaimQueue {
		var depth uint8

		queue := make(TransposedClaimQueue)
		queue[paraID] = make(map[uint8]map[CoreIndex]struct{}, 1) // Create single depth for default offset 0

		if len(offset) > 0 {
			depth = offset[0] // Use provided offset or default to 0
		}
		queue[paraID][depth] = make(map[CoreIndex]struct{}) // Initialize map at depth 0

		// Add cores to the map
		for _, core := range cores {
			queue[paraID][depth][core] = struct{}{}
		}

		return queue
	}

	tests := []struct {
		name        string
		receipt     CommittedCandidateReceiptV2
		claimQueue  TransposedClaimQueue
		coreSelect  *SelectCore
		expectError error
	}{
		// V1 Descriptor Cases
		{
			name:       "v1_descriptor_without_core_selector",
			receipt:    newTestReceipt(CandidateDescriptorVersion1, 1, 100),
			claimQueue: mockTransposedClaimQueue(100, []CoreIndex{{Index: 1}}),
			coreSelect: nil,
		},
		{
			name:        "v1_descriptor_with_core_selector",
			receipt:     newTestReceipt(CandidateDescriptorVersion1, 1, 100),
			claimQueue:  mockTransposedClaimQueue(100, []CoreIndex{{Index: 1}}),
			coreSelect:  &SelectCore{CoreSelector: 0, ClaimQueueOffset: 0},
			expectError: ErrCoreSelectorWithV1Descriptor,
		},

		// V2 Descriptor - No Cores Assigned
		{
			name:        "v2_descriptor_empty_claim_queue",
			receipt:     newTestReceipt(CandidateDescriptorVersion2, 1, 100),
			claimQueue:  TransposedClaimQueue{},
			expectError: ErrNoCoresAssigned,
		},
		{
			name:        "v2_descriptor_no_cores_assigned",
			receipt:     newTestReceipt(CandidateDescriptorVersion2, 1, 100),
			claimQueue:  mockTransposedClaimQueue(100, []CoreIndex{}),
			expectError: ErrNoCoresAssigned,
		},

		// V2 Descriptor - Single Core Cases
		{
			name:       "v2_descriptor_single_core_without_selector",
			receipt:    newTestReceipt(CandidateDescriptorVersion2, 1, 100),
			claimQueue: mockTransposedClaimQueue(100, []CoreIndex{{Index: 1}}),
			coreSelect: nil,
		},
		{
			name:        "v2_descriptor_single_core_mismatch",
			receipt:     newTestReceipt(CandidateDescriptorVersion2, 2, 100),
			claimQueue:  mockTransposedClaimQueue(100, []CoreIndex{{Index: 1}}),
			coreSelect:  &SelectCore{CoreSelector: 0, ClaimQueueOffset: 0},
			expectError: ErrCoreIndexMismatch,
		},

		// V2 Descriptor - Multiple Cores Cases
		{
			name:       "v2_descriptor_multiple_cores_with_selector",
			receipt:    newTestReceipt(CandidateDescriptorVersion2, 2, 100),
			claimQueue: mockTransposedClaimQueue(100, []CoreIndex{{Index: 1}, {Index: 2}, {Index: 3}}),
			coreSelect: &SelectCore{CoreSelector: 1, ClaimQueueOffset: 0},
		},
		{
			name:       "v2_descriptor_multiple_cores_valid_wrapped_selector",
			receipt:    newTestReceipt(CandidateDescriptorVersion2, 2, 100),
			claimQueue: mockTransposedClaimQueue(100, []CoreIndex{{Index: 1}, {Index: 2}}),
			coreSelect: &SelectCore{CoreSelector: 5, ClaimQueueOffset: 0}, // 5 % 2 = 1, should select second core
		},

		// V2 Descriptor - Without Selector Cases
		{
			name:       "v2_descriptor_multiple_cores_valid_without_selector",
			receipt:    newTestReceipt(CandidateDescriptorVersion2, 2, 100),
			claimQueue: mockTransposedClaimQueue(100, []CoreIndex{{Index: 1}, {Index: 2}, {Index: 3}}),
			coreSelect: nil,
		},
		{
			name:        "v2_descriptor_multiple_cores_invalid_without_selector",
			receipt:     newTestReceipt(CandidateDescriptorVersion2, 4, 100),
			claimQueue:  mockTransposedClaimQueue(100, []CoreIndex{{Index: 1}, {Index: 2}, {Index: 3}}),
			coreSelect:  nil,
			expectError: ErrInvalidCoreIndex,
		},

		// Claim Queue Offset Cases
		{
			name:       "v2_descriptor_with_non_zero_claim_queue_offset",
			receipt:    newTestReceipt(CandidateDescriptorVersion2, 1, 100),
			claimQueue: mockTransposedClaimQueue(100, []CoreIndex{{Index: 1}}, 1),
			coreSelect: &SelectCore{CoreSelector: 0, ClaimQueueOffset: 1},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Set core selector if provided
			if tt.coreSelect != nil {
				signal := UMPSignal{}
				err := signal.SetValue(*tt.coreSelect)
				require.NoError(t, err)

				signalBytes, err := scale.Marshal(signal)
				require.NoError(t, err)

				// Add empty message separator followed by the signal
				tt.receipt.Commitments.UpwardMessages = []UpwardMessage{
					{}, // empty message as separator
					signalBytes,
				}
			}

			err := tt.receipt.CheckCoreIndex(tt.claimQueue)

			if tt.expectError != nil {
				require.ErrorIs(t, err, tt.expectError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
