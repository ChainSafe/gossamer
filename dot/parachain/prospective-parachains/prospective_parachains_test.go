package prospectiveparachains

import (
	"bytes"
	"context"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
)

func introduceSecondedCandidate(
	t *testing.T,
	overseerToSubsystem chan any,
	candidate parachaintypes.CommittedCandidateReceipt,
	pvd parachaintypes.PersistedValidationData,
) {
	req := IntroduceSecondedCandidateRequest{
		CandidateParaID:         candidate.Descriptor.ParaID,
		CandidateReceipt:        candidate,
		PersistedValidationData: pvd,
	}

	response := make(chan bool)

	msg := IntroduceSecondedCandidate{
		IntroduceSecondedCandidateRequest: req,
		Response:                          response,
	}

	overseerToSubsystem <- msg

	assert.True(t, <-response)
}

func introduceSecondedCandidateFailed(
	t *testing.T,
	overseerToSubsystem chan any,
	candidate parachaintypes.CommittedCandidateReceipt,
	pvd parachaintypes.PersistedValidationData,
) {
	req := IntroduceSecondedCandidateRequest{
		CandidateParaID:         candidate.Descriptor.ParaID,
		CandidateReceipt:        candidate,
		PersistedValidationData: pvd,
	}

	response := make(chan bool)

	msg := IntroduceSecondedCandidate{
		IntroduceSecondedCandidateRequest: req,
		Response:                          response,
	}

	overseerToSubsystem <- msg

	assert.False(t, <-response)
}

func TestFailedIntroduceSecondedCandidateWhenMissingViewPerRelayParent(
	t *testing.T,
) {
	candidateRelayParent := common.Hash{0x01}
	paraId := parachaintypes.ParaID(1)
	parentHead := parachaintypes.HeadData{
		Data: bytes.Repeat([]byte{0x01}, 32),
	}
	headData := parachaintypes.HeadData{
		Data: bytes.Repeat([]byte{0x02}, 32),
	}
	validationCodeHash := parachaintypes.ValidationCodeHash{0x01}
	candidateRelayParentNumber := uint32(1)

	candidate := makeCandidate(
		candidateRelayParent,
		candidateRelayParentNumber,
		paraId,
		parentHead,
		headData,
		validationCodeHash,
	)

	pvd := dummyPVD(parentHead, candidateRelayParentNumber)

	subsystemToOverseer := make(chan any)
	overseerToSubsystem := make(chan any)

	prospectiveParachains := NewProspectiveParachains(subsystemToOverseer)

	go prospectiveParachains.Run(context.Background(), overseerToSubsystem)

	introduceSecondedCandidateFailed(t, overseerToSubsystem, candidate, pvd)
}

func TestFailedIntroduceSecondedCandidateWhenParentHeadAndHeadDataEquals(
	t *testing.T,
) {
	candidateRelayParent := common.Hash{0x01}
	paraId := parachaintypes.ParaID(1)
	parentHead := parachaintypes.HeadData{
		Data: bytes.Repeat([]byte{0x01}, 32),
	}
	headData := parachaintypes.HeadData{
		Data: bytes.Repeat([]byte{0x01}, 32),
	}
	validationCodeHash := parachaintypes.ValidationCodeHash{0x01}
	candidateRelayParentNumber := uint32(1)

	candidate := makeCandidate(
		candidateRelayParent,
		candidateRelayParentNumber,
		paraId,
		parentHead,
		headData,
		validationCodeHash,
	)

	pvd := dummyPVD(parentHead, candidateRelayParentNumber)

	subsystemToOverseer := make(chan any)
	overseerToSubsystem := make(chan any)

	prospectiveParachains := NewProspectiveParachains(subsystemToOverseer)

	relayParent := relayChainBlockInfo{
		Hash:        candidateRelayParent,
		Number:      0,
		StorageRoot: common.Hash{0x00},
	}

	baseConstraints := &parachaintypes.Constraints{
		RequiredParent:       parachaintypes.HeadData{Data: []byte{byte(0)}},
		MinRelayParentNumber: 0,
		ValidationCodeHash:   parachaintypes.ValidationCodeHash(common.Hash{0x03}),
	}
	scope, err := newScopeWithAncestors(relayParent, baseConstraints, nil, 10, nil)
	assert.NoError(t, err)

	prospectiveParachains.View.perRelayParent[candidateRelayParent] = &relayParentData{
		fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
			paraId: newFragmentChain(scope, newCandidateStorage()),
		},
	}
	go prospectiveParachains.Run(context.Background(), overseerToSubsystem)

	introduceSecondedCandidateFailed(t, overseerToSubsystem, candidate, pvd)
}

func TestHandleIntroduceSecondedCandidate(
	t *testing.T,
) {
	candidateRelayParent := common.Hash{0x01}
	paraId := parachaintypes.ParaID(1)
	parentHead := parachaintypes.HeadData{
		Data: bytes.Repeat([]byte{0x01}, 32),
	}
	headData := parachaintypes.HeadData{
		Data: bytes.Repeat([]byte{0x02}, 32),
	}
	validationCodeHash := parachaintypes.ValidationCodeHash{0x01}
	candidateRelayParentNumber := uint32(1)

	candidate := makeCandidate(
		candidateRelayParent,
		candidateRelayParentNumber,
		paraId,
		parentHead,
		headData,
		validationCodeHash,
	)

	pvd := dummyPVD(parentHead, candidateRelayParentNumber)

	subsystemToOverseer := make(chan any)
	overseerToSubsystem := make(chan any)

	prospectiveParachains := NewProspectiveParachains(subsystemToOverseer)

	relayParent := relayChainBlockInfo{
		Hash:        candidateRelayParent,
		Number:      0,
		StorageRoot: common.Hash{0x00},
	}

	baseConstraints := &parachaintypes.Constraints{
		RequiredParent:       parachaintypes.HeadData{Data: []byte{byte(0)}},
		MinRelayParentNumber: 0,
		ValidationCodeHash:   parachaintypes.ValidationCodeHash(common.Hash{0x03}),
	}

	scope, err := newScopeWithAncestors(relayParent, baseConstraints, nil, 10, nil)
	assert.NoError(t, err)

	prospectiveParachains.View.perRelayParent[candidateRelayParent] = &relayParentData{
		fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
			paraId: newFragmentChain(scope, newCandidateStorage()),
		},
	}
	go prospectiveParachains.Run(context.Background(), overseerToSubsystem)

	introduceSecondedCandidate(t, overseerToSubsystem, candidate, pvd)
}

const MaxPoVSize = 1_000_000

func dummyPVD(parentHead parachaintypes.HeadData, relayParentNumber uint32) parachaintypes.PersistedValidationData {
	return parachaintypes.PersistedValidationData{
		ParentHead:             parentHead,
		RelayParentNumber:      relayParentNumber,
		RelayParentStorageRoot: common.EmptyHash,
		MaxPovSize:             MaxPoVSize,
	}
}

func dummyCandidateReceiptBadSig(
	relayParentHash common.Hash,
	commitments *common.Hash,
) parachaintypes.CandidateReceipt {
	var commitmentsHash common.Hash

	if commitments != nil {
		commitmentsHash = *commitments
	} else {
		commitmentsHash = common.EmptyHash
	}

	descriptor := parachaintypes.CandidateDescriptor{
		ParaID:                      parachaintypes.ParaID(0),
		RelayParent:                 relayParentHash,
		Collator:                    parachaintypes.CollatorID{},
		PovHash:                     common.EmptyHash,
		ErasureRoot:                 common.EmptyHash,
		Signature:                   parachaintypes.CollatorSignature{},
		ParaHead:                    common.EmptyHash,
		ValidationCodeHash:          parachaintypes.ValidationCodeHash{},
		PersistedValidationDataHash: common.EmptyHash,
	}

	return parachaintypes.CandidateReceipt{
		CommitmentsHash: commitmentsHash,
		Descriptor:      descriptor,
	}
}

func makeCandidate(
	relayParent common.Hash,
	relayParentNumber uint32,
	paraID parachaintypes.ParaID,
	parentHead parachaintypes.HeadData,
	headData parachaintypes.HeadData,
	validationCodeHash parachaintypes.ValidationCodeHash,
) parachaintypes.CommittedCandidateReceipt {
	pvd := dummyPVD(parentHead, relayParentNumber)

	commitments := parachaintypes.CandidateCommitments{
		HeadData:                  headData,
		HorizontalMessages:        []parachaintypes.OutboundHrmpMessage{},
		UpwardMessages:            []parachaintypes.UpwardMessage{},
		NewValidationCode:         nil,
		ProcessedDownwardMessages: 0,
		HrmpWatermark:             relayParentNumber,
	}

	commitmentsHash := commitments.Hash()

	candidate := dummyCandidateReceiptBadSig(relayParent, &commitmentsHash)
	candidate.CommitmentsHash = commitments.Hash()
	candidate.Descriptor.ParaID = paraID

	pvdh, err := pvd.Hash()

	if err != nil {
		panic(err)
	}

	candidate.Descriptor.PersistedValidationDataHash = pvdh
	candidate.Descriptor.ValidationCodeHash = validationCodeHash

	result := parachaintypes.CommittedCandidateReceipt{
		Descriptor:  candidate.Descriptor,
		Commitments: commitments,
	}

	return result
}

func padTo32Bytes(input []byte) common.Hash {
	var hash common.Hash
	copy(hash[:], input)
	return hash
}

// TestGetMinimumRelayParents ensures that getMinimumRelayParents
// processes the relay parent hash and correctly sends the output via the channel
func TestGetMinimumRelayParents(t *testing.T) {
	// Setup a mock View with active leaves and relay parent data

	mockRelayParent := relayChainBlockInfo{
		Hash:   padTo32Bytes([]byte("active_hash")),
		Number: 10,
	}

	ancestors := []relayChainBlockInfo{
		{
			Hash:   padTo32Bytes([]byte("active_hash_7")),
			Number: 9,
		},
		{
			Hash:   padTo32Bytes([]byte("active_hash_8")),
			Number: 8,
		},
		{
			Hash:   padTo32Bytes([]byte("active_hash_9")),
			Number: 7,
		},
	}

	baseConstraints := &parachaintypes.Constraints{
		MinRelayParentNumber: 5,
	}

	mockScope, err := newScopeWithAncestors(mockRelayParent, baseConstraints, nil, 10, ancestors)
	assert.NoError(t, err)

	mockScope2, err := newScopeWithAncestors(mockRelayParent, baseConstraints, nil, 10, nil)
	assert.NoError(t, err)

	mockView := &view{
		activeLeaves: map[common.Hash]bool{
			common.BytesToHash([]byte("active_hash")): true,
		},
		perRelayParent: map[common.Hash]*relayParentData{
			common.BytesToHash([]byte("active_hash")): {
				fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
					parachaintypes.ParaID(1): newFragmentChain(mockScope, newCandidateStorage()),
					parachaintypes.ParaID(2): newFragmentChain(mockScope2, newCandidateStorage()),
				},
			},
		},
	}

	// Initialize ProspectiveParachains with the mock view
	pp := &ProspectiveParachains{
		View: mockView,
	}

	// Create a channel to capture the output
	sender := make(chan []ParaIDBlockNumber, 1)

	// Execute the method under test
	pp.getMinimumRelayParents(common.BytesToHash([]byte("active_hash")), sender)

	expected := []ParaIDBlockNumber{
		{
			ParaId:      1,
			BlockNumber: 10,
		},
		{
			ParaId:      2,
			BlockNumber: 10,
		},
	}
	// Validate the results
	result := <-sender
	assert.Len(t, result, 2)
	assert.Equal(t, expected, result)
}

// TestGetMinimumRelayParents_NoActiveLeaves ensures that getMinimumRelayParents
// correctly handles the case where there are no active leaves.
func TestGetMinimumRelayParents_NoActiveLeaves(t *testing.T) {
	mockView := &view{
		activeLeaves:   map[common.Hash]bool{},
		perRelayParent: map[common.Hash]*relayParentData{},
	}

	// Initialize ProspectiveParachains with the mock view
	pp := &ProspectiveParachains{
		View: mockView,
	}

	// Create a channel to capture the output
	sender := make(chan []ParaIDBlockNumber, 1)

	// Execute the method under test
	pp.getMinimumRelayParents(common.BytesToHash([]byte("active_hash")), sender)
	// Validate the results
	result := <-sender
	assert.Empty(t, result, "Expected result to be empty when no active leaves are present")
}

func TestGetBackableCandidates(t *testing.T) {
	candidateRelayParent1 := common.Hash{0x01}
	candidateRelayParent2 := common.Hash{0x02}
	candidateRelayParent3 := common.Hash{0x03}

	paraId := parachaintypes.ParaID(1)
	paraId2 := parachaintypes.ParaID(2)
	paraId3 := parachaintypes.ParaID(3)

	parentHead1 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x01}, 32)}
	parentHead2 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x02}, 32)}
	parentHead3 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x03}, 32)}

	headData1 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x01}, 32)}
	headData2 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x02}, 32)}
	headData3 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x03}, 32)}

	validationCodeHash := parachaintypes.ValidationCodeHash{}

	candidate1 := makeCandidate(
		candidateRelayParent1,
		uint32(10),
		paraId,
		parentHead1,
		headData1,
		validationCodeHash,
	)

	candidate2 := makeCandidate(
		candidateRelayParent2,
		uint32(9),
		paraId2,
		parentHead2,
		headData2,
		validationCodeHash,
	)

	candidate3 := makeCandidate(
		candidateRelayParent3,
		uint32(8),
		paraId3,
		parentHead3,
		headData3,
		validationCodeHash,
	)

	mockRelayParent := relayChainBlockInfo{
		Hash:   candidateRelayParent1,
		Number: 10,
	}

	ancestors := []relayChainBlockInfo{
		{
			Hash:   candidateRelayParent2,
			Number: 9,
		},
		{
			Hash:   candidateRelayParent3,
			Number: 8,
		},
	}

	baseConstraints := &parachaintypes.Constraints{
		MinRelayParentNumber: 8,
		RequiredParent:       parentHead1,
		MaxPoVSize:           MaxPoVSize,
	}

	mockScope, err := newScopeWithAncestors(mockRelayParent, baseConstraints, nil, 10, ancestors)
	assert.NoError(t, err)

	parentHash1, err := parentHead1.Hash()
	assert.NoError(t, err)

	outputHash1, err := headData1.Hash()
	assert.NoError(t, err)

	candidateStorage := newCandidateStorage()
	err = candidateStorage.addCandidateEntry(&candidateEntry{
		candidateHash:      parachaintypes.CandidateHash{Value: candidateRelayParent1},
		parentHeadDataHash: parentHash1,
		outputHeadDataHash: outputHash1,
		relayParent:        candidateRelayParent1,
		candidate: &prospectiveCandidate{
			Commitments:             candidate1.Commitments,
			PersistedValidationData: dummyPVD(parentHead1, 10),
			PoVHash:                 candidate1.Descriptor.PovHash,
			ValidationCodeHash:      validationCodeHash,
		},
		state: backed,
	})
	assert.NoError(t, err)

	parentHash2, err := parentHead2.Hash()
	assert.NoError(t, err)

	outputHash2, err := headData2.Hash()
	assert.NoError(t, err)

	err = candidateStorage.addCandidateEntry(&candidateEntry{
		candidateHash:      parachaintypes.CandidateHash{Value: candidateRelayParent2},
		parentHeadDataHash: parentHash2,
		outputHeadDataHash: outputHash2,
		relayParent:        candidateRelayParent2,
		candidate: &prospectiveCandidate{
			Commitments:             candidate2.Commitments,
			PersistedValidationData: dummyPVD(parentHead2, 9),
			PoVHash:                 candidate2.Descriptor.PovHash,
			ValidationCodeHash:      validationCodeHash,
		},
		state: backed,
	})
	assert.NoError(t, err)

	parentHash3, err := parentHead3.Hash()
	assert.NoError(t, err)

	outputHash3, err := headData3.Hash()
	assert.NoError(t, err)

	err = candidateStorage.addCandidateEntry(&candidateEntry{
		candidateHash:      parachaintypes.CandidateHash{Value: candidateRelayParent3},
		parentHeadDataHash: parentHash3,
		outputHeadDataHash: outputHash3,
		relayParent:        candidateRelayParent3,
		candidate: &prospectiveCandidate{
			Commitments:             candidate3.Commitments,
			PersistedValidationData: dummyPVD(parentHead3, 8),
			PoVHash:                 candidate3.Descriptor.PovHash,
			ValidationCodeHash:      validationCodeHash,
		},
		state: backed,
	})
	assert.NoError(t, err)

	type testCase struct {
		name           string
		view           *view
		msg            GetBackableCandidates
		expectedLength int
	}

	cases := []testCase{
		{
			name: "relay_parent_inactive",
			view: &view{
				activeLeaves:   map[common.Hash]bool{},
				perRelayParent: map[common.Hash]*relayParentData{},
			},
			msg: GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    1,
				Ancestors:       Ancestors{},
				Response:        make(chan []parachaintypes.CandidateHashAndRelayParent, 1),
			},
			expectedLength: 0,
		},
		{
			name: "active_leaves_empty",
			view: &view{
				activeLeaves: map[common.Hash]bool{},
				perRelayParent: map[common.Hash]*relayParentData{
					candidateRelayParent1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{},
					},
				},
			},
			msg: GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    1,
				Ancestors:       Ancestors{},
				Response:        make(chan []parachaintypes.CandidateHashAndRelayParent, 1),
			},
			expectedLength: 0,
		},
		{
			name: "no_candidates_found",
			view: &view{
				activeLeaves: map[common.Hash]bool{
					candidateRelayParent1: true,
				},
				perRelayParent: map[common.Hash]*relayParentData{
					candidateRelayParent1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{},
					},
				},
			},
			msg: GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    1,
				Ancestors:       Ancestors{},
				Response:        make(chan []parachaintypes.CandidateHashAndRelayParent, 1),
			},
			expectedLength: 0,
		},
		{
			name: "not_found_parachain_based_on_paraid",
			view: &view{
				activeLeaves: map[common.Hash]bool{
					candidateRelayParent1: true,
				},
				perRelayParent: map[common.Hash]*relayParentData{
					candidateRelayParent1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{},
					},
				},
			},
			msg: GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(3),
				RequestedQty:    1,
				Ancestors:       Ancestors{},
				Response:        make(chan []parachaintypes.CandidateHashAndRelayParent, 1),
			},
			expectedLength: 0,
		},
		{
			name: "candidates_found",
			view: &view{
				activeLeaves: map[common.Hash]bool{
					candidateRelayParent1: true,
				},
				perRelayParent: map[common.Hash]*relayParentData{
					candidateRelayParent1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
							parachaintypes.ParaID(1): newFragmentChain(mockScope, candidateStorage),
						},
					},
				},
			},
			msg: GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    2,
				Ancestors: Ancestors{
					parachaintypes.CandidateHash{Value: candidateRelayParent2}: struct{}{},
					parachaintypes.CandidateHash{Value: candidateRelayParent3}: struct{}{},
				},
				Response: make(chan []parachaintypes.CandidateHashAndRelayParent, 1),
			},
			expectedLength: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pp := &ProspectiveParachains{
				View: tc.view,
			}

			pp.getBackableCandidates(tc.msg)

			select {
			case result := <-tc.msg.Response:
				assert.Equal(t, tc.expectedLength, len(result), "Unexpected number of candidates")
			default:
				t.Fatal("No response received from getBackableCandidates")
			}
		})
	}
}
