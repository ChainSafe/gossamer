// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package prospectiveparachains

import (
	"bytes"
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func introduceSecondedCandidate(
	t *testing.T,
	overseerToSubsystem chan any,
	candidate parachaintypes.CommittedCandidateReceiptV2,
	pvd parachaintypes.PersistedValidationData,
) {
	req := messages.IntroduceSecondedCandidateRequest{
		CandidateParaID:         candidate.Descriptor.ParaID,
		CandidateReceipt:        candidate,
		PersistedValidationData: pvd,
	}

	response := make(chan bool)

	msg := messages.IntroduceSecondedCandidate{
		Request:  req,
		Response: response,
	}

	overseerToSubsystem <- msg

	assert.True(t, <-response)
}

func introduceSecondedCandidateFailed(
	t *testing.T,
	overseerToSubsystem chan any,
	candidate parachaintypes.CommittedCandidateReceiptV2,
	pvd parachaintypes.PersistedValidationData,
) {
	req := messages.IntroduceSecondedCandidateRequest{
		CandidateParaID:         candidate.Descriptor.ParaID,
		CandidateReceipt:        candidate,
		PersistedValidationData: pvd,
	}

	response := make(chan bool)

	msg := messages.IntroduceSecondedCandidate{
		Request:  req,
		Response: response,
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

	prospectiveParachains := NewProspectiveParachains(subsystemToOverseer, nil)

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

	prospectiveParachains := NewProspectiveParachains(subsystemToOverseer, nil)

	relayParent := relayChainBlockInfo{
		Hash:        candidateRelayParent,
		Number:      0,
		StorageRoot: common.Hash{0x00},
	}

	baseConstraints := &parachaintypes.VStagingConstraints{
		RequiredParent:       parachaintypes.HeadData{Data: []byte{byte(0)}},
		MinRelayParentNumber: 0,
		ValidationCodeHash:   parachaintypes.ValidationCodeHash(common.Hash{0x03}),
	}
	scope, err := newScopeWithAncestors(relayParent, baseConstraints, nil, 10, nil)
	assert.NoError(t, err)

	prospectiveParachains.view.perRelayParent[candidateRelayParent] = &relayParentData{
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
	validationCodeHash := parachaintypes.ValidationCodeHash(common.Hash{0x03})
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

	prospectiveParachains := NewProspectiveParachains(subsystemToOverseer, nil)

	relayParent := relayChainBlockInfo{
		Hash:        candidateRelayParent,
		Number:      0,
		StorageRoot: common.Hash{0x00},
	}

	baseConstraints := &parachaintypes.VStagingConstraints{
		RequiredParent:       parachaintypes.HeadData{Data: []byte{byte(0)}},
		MaxHeadDataSize:      1_000_000,
		MinRelayParentNumber: 0,
		ValidationCodeHash:   parachaintypes.ValidationCodeHash(common.Hash{0x03}),
	}

	scope, err := newScopeWithAncestors(relayParent, baseConstraints, nil, 10, nil)
	assert.NoError(t, err)

	prospectiveParachains.view.perRelayParent[candidateRelayParent] = &relayParentData{
		fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
			paraId: newFragmentChain(scope, newCandidateStorage()),
		},
	}
	go prospectiveParachains.Run(context.Background(), overseerToSubsystem)

	introduceSecondedCandidate(t, overseerToSubsystem, candidate, pvd)
}

const MaxPoVSize = 1_000_000

func dummyConstraintsV2(
	minRelayParentNumber parachaintypes.BlockNumber,
	validWatermarks []parachaintypes.BlockNumber,
	requiredParent parachaintypes.HeadData,
	validationCodeHash parachaintypes.ValidationCodeHash,
) *parachaintypes.VStagingConstraints {
	return &parachaintypes.VStagingConstraints{
		MinRelayParentNumber:   minRelayParentNumber,
		MaxPoVSize:             MaxPoVSize,
		MaxCodeSize:            1_000_000,
		MaxHeadDataSize:        20480,
		UMPRemaining:           10,
		UMPRemainingBytes:      1_000,
		MaxNumUMPPerCandidate:  10,
		DMPRemainingMessages:   nil,
		HRMPInbound:            parachaintypes.InboundHRMPLimitations{ValidWatermarks: validWatermarks},
		HRMPChannelsOut:        nil,
		MaxNumHRMPPerCandidate: 0,
		RequiredParent:         requiredParent,
		ValidationCodeHash:     validationCodeHash,
		UpgradeRestriction:     nil,
		FutureValidationCode:   nil,
	}
}

func dummyPVD(
	parentHead parachaintypes.HeadData,
	relayParentNumber uint32,
) parachaintypes.PersistedValidationData {
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
) parachaintypes.CommittedCandidateReceiptV2 {
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

	return result.V2()
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

	baseConstraints := &parachaintypes.VStagingConstraints{
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
		view: mockView,
	}

	// Create a channel to capture the output
	sender := make(chan []messages.ParaIDBlockNumber, 1)

	// Execute the method under test
	pp.getMinimumRelayParents(common.BytesToHash([]byte("active_hash")), sender)

	expected := []messages.ParaIDBlockNumber{
		{
			ParaId:      1,
			BlockNumber: 10,
		},
		{
			ParaId:      2,
			BlockNumber: 10,
		},
	}

	result := <-sender
	assert.Len(t, result, 2)

	// orders might be different, so sort the result to compare
	sort.Slice(result, func(i, j int) bool {
		return result[i].ParaId < result[j].ParaId
	})

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
		view: mockView,
	}

	// Create a channel to capture the output
	sender := make(chan []messages.ParaIDBlockNumber, 1)

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

	baseConstraints := &parachaintypes.VStagingConstraints{
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
		msg            messages.GetBackableCandidates
		expectedLength int
	}

	cases := []testCase{
		{
			name: "relay_parent_inactive",
			view: &view{
				activeLeaves:   map[common.Hash]bool{},
				perRelayParent: map[common.Hash]*relayParentData{},
			},
			msg: messages.GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    1,
				Ancestors:       messages.Ancestors{},
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
			msg: messages.GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    1,
				Ancestors:       messages.Ancestors{},
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
			msg: messages.GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    1,
				Ancestors:       messages.Ancestors{},
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
			msg: messages.GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(3),
				RequestedQty:    1,
				Ancestors:       messages.Ancestors{},
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
			msg: messages.GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    2,
				Ancestors: messages.Ancestors{
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
				view: tc.view,
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

func TestAnswerProspectiveValidationDataRequest(t *testing.T) {
	// Create a new prospective parachains instance
	subsystemToOverseer := make(chan any)

	t.Run("no_active_leaves", func(t *testing.T) {
		view := &view{
			activeLeaves: map[common.Hash]bool{},
		}

		// Create a prospective validation data request
		request := messages.ProspectiveValidationDataRequest{
			ParaId:               parachaintypes.ParaID(1),
			CandidateRelayParent: common.Hash{0x01},
			ParentHeadData: messages.ParentHeadDataWithHash{
				Hash: common.Hash{0x01},
				Data: parachaintypes.HeadData{Data: []byte{0x01}},
			},
		}

		pp := &ProspectiveParachains{
			view: view,
		}

		sender := make(chan *parachaintypes.PersistedValidationData, 1)

		// Execute the method under test
		pp.answerProspectiveValidationDataRequest(request, sender)

		result := <-sender

		assert.Nil(t, result, "Expected result to be nil when no active leaves are present")
	})

	t.Run("with_active_and_only_hash", func(t *testing.T) {
		reqParent := parachaintypes.HeadData{Data: []byte{0xc3}}
		reqParentHash, err := reqParent.Hash()
		assert.NoError(t, err)

		baseConstraints := &parachaintypes.VStagingConstraints{
			RequiredParent:       reqParent,
			MinRelayParentNumber: 0,
			ValidationCodeHash:   parachaintypes.ValidationCodeHash(common.Hash{0x03}),
			MaxPoVSize:           3,
		}

		scope, err := newScopeWithAncestors(
			relayChainBlockInfo{
				Hash:        common.Hash{0x01},
				StorageRoot: common.Hash{0x01},
				Number:      2,
			},
			baseConstraints,
			nil,
			3,
			[]relayChainBlockInfo{
				{
					Hash:        common.Hash{0x012},
					StorageRoot: common.Hash{0x022},
					Number:      1,
				},
			},
		)

		assert.NoError(t, err)

		view := &view{
			activeLeaves: map[common.Hash]bool{
				{0x01}: true,
			},
			perRelayParent: map[common.Hash]*relayParentData{
				{0x01}: {
					fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
						parachaintypes.ParaID(1): newFragmentChain(scope, newCandidateStorage()),
					},
				},
			},
		}

		// Create a prospective validation data request
		request := messages.ProspectiveValidationDataRequest{
			ParaId:               parachaintypes.ParaID(1),
			CandidateRelayParent: common.Hash{0x01},
			ParentHeadData:       messages.OnlyHash(reqParentHash),
		}

		pp := &ProspectiveParachains{
			view: view,
		}

		sender := make(chan *parachaintypes.PersistedValidationData, 1)

		// Execute the method under test
		pp.answerProspectiveValidationDataRequest(request, sender)

		result := <-sender

		assert.Equal(t, result.ParentHead, reqParent)
		assert.Equal(t, result.RelayParentNumber, uint32(2))
		assert.Equal(t, result.RelayParentStorageRoot, common.Hash{0x01})
		assert.Equal(t, result.MaxPovSize, uint32(3))
	})

	t.Run("with_active_and_with_head_data", func(t *testing.T) {
		reqParent := parachaintypes.HeadData{Data: []byte{0xc3}}
		reqParentHash, err := reqParent.Hash()
		assert.NoError(t, err)

		baseConstraints := &parachaintypes.VStagingConstraints{
			RequiredParent:       reqParent,
			MinRelayParentNumber: 0,
			ValidationCodeHash:   parachaintypes.ValidationCodeHash(common.Hash{0x03}),
			MaxPoVSize:           3,
		}

		scope, err := newScopeWithAncestors(
			relayChainBlockInfo{
				Hash:        common.Hash{0x01},
				StorageRoot: common.Hash{0x01},
				Number:      2,
			},
			baseConstraints,
			nil,
			3,
			[]relayChainBlockInfo{
				{
					Hash:        common.Hash{0x012},
					StorageRoot: common.Hash{0x022},
					Number:      1,
				},
			},
		)

		assert.NoError(t, err)

		view := &view{
			activeLeaves: map[common.Hash]bool{
				{0x01}: true,
			},
			perRelayParent: map[common.Hash]*relayParentData{
				{0x01}: {
					fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
						parachaintypes.ParaID(1): newFragmentChain(scope, newCandidateStorage()),
					},
				},
			},
		}

		// Create a prospective validation data request
		request := messages.ProspectiveValidationDataRequest{
			ParaId:               parachaintypes.ParaID(1),
			CandidateRelayParent: common.Hash{0x01},
			ParentHeadData: messages.ParentHeadDataWithHash{
				Data: reqParent,
				Hash: reqParentHash,
			},
		}

		pp := &ProspectiveParachains{
			view: view,
		}

		sender := make(chan *parachaintypes.PersistedValidationData, 1)

		// Execute the method under test
		pp.answerProspectiveValidationDataRequest(request, sender)

		result := <-sender

		assert.Equal(t, result.ParentHead, reqParent)
		assert.Equal(t, result.RelayParentNumber, uint32(2))
		assert.Equal(t, result.RelayParentStorageRoot, common.Hash{0x01})
		assert.Equal(t, result.MaxPovSize, uint32(3))
	})

	t.Run("with_head_data_hash_doesnt_match", func(t *testing.T) {
		reqParent := parachaintypes.HeadData{Data: []byte{0xc4}}

		baseConstraints := &parachaintypes.VStagingConstraints{
			RequiredParent:       reqParent,
			MinRelayParentNumber: 0,
			ValidationCodeHash:   parachaintypes.ValidationCodeHash(common.Hash{0x03}),
			MaxPoVSize:           3,
		}

		scope, err := newScopeWithAncestors(
			relayChainBlockInfo{
				Hash:        common.Hash{0x01},
				StorageRoot: common.Hash{0x01},
				Number:      2,
			},
			baseConstraints,
			nil,
			3,
			[]relayChainBlockInfo{
				{
					Hash:        common.Hash{0x012},
					StorageRoot: common.Hash{0x022},
					Number:      1,
				},
			},
		)

		assert.NoError(t, err)

		view := &view{
			activeLeaves: map[common.Hash]bool{
				{0x01}: true,
			},
			perRelayParent: map[common.Hash]*relayParentData{
				{0x01}: {
					fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
						parachaintypes.ParaID(1): newFragmentChain(scope, newCandidateStorage()),
					},
				},
			},
		}

		// Create a prospective validation data request
		request := messages.ProspectiveValidationDataRequest{
			ParaId:               parachaintypes.ParaID(1),
			CandidateRelayParent: common.Hash{0x01},
			ParentHeadData: messages.ParentHeadDataWithHash{
				Data: reqParent,
				Hash: common.Hash{0xc3},
			},
		}

		pp := &ProspectiveParachains{
			view: view,
		}

		sender := make(chan *parachaintypes.PersistedValidationData, 1)

		// Execute the method under test
		pp.answerProspectiveValidationDataRequest(request, sender)

		result := <-sender

		assert.NotEqual(t, result.ParentHead, common.Hash{0xc3})
	})

	// Close the channels
	close(subsystemToOverseer)
}

func TestGetHypotheticalMembership(t *testing.T) {
	t.Run("no_active_leaves", func(t *testing.T) {
		candidate := parachaintypes.HypotheticalCandidateComplete{}

		pp := &ProspectiveParachains{
			view: &view{
				activeLeaves: map[common.Hash]bool{},
			},
		}

		response := make(chan []*messages.HypotheticalMembershipResponseItem, 1)
		msg := messages.GetHypotheticalMembership{
			Candidates: []parachaintypes.HypotheticalCandidate{
				candidate,
			},
			// setting as nil to search all active leaves
			FragmentChainRelayParent: nil,
			Response:                 response,
		}

		pp.getHypotheticalMembership(msg)

		out := <-response
		expected := []*messages.HypotheticalMembershipResponseItem{
			{
				HypotheticalCandidate:  candidate,
				HypotheticalMembership: make([]common.Hash, 0),
			},
		}

		require.Equal(t, expected, out)
	})

	t.Run("candidate_unconnected", func(t *testing.T) {
		paraID := parachaintypes.ParaID(1000)
		relayBlockHash1 := common.Hash([32]byte{0xAB, 0xCD})
		relayBlockNumber1 := uint32(1)

		candidateHash := parachaintypes.CandidateHash{
			Value: common.Hash([32]byte{0x01, 0x02, 0x03}),
		}

		fcForParaID := &fragmentChain{
			unconnected: &candidateStorage{
				byCandidateHash: map[parachaintypes.CandidateHash]*candidateEntry{
					candidateHash: {},
				},
			},
			bestChain: &backedChain{
				candidates: map[parachaintypes.CandidateHash]struct{}{},
			},
		}

		pp := &ProspectiveParachains{
			view: &view{
				activeLeaves: map[common.Hash]bool{
					relayBlockHash1: true,
				},

				perRelayParent: map[common.Hash]*relayParentData{
					relayBlockHash1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
							paraID: fcForParaID,
						},
					},
				},
			},
		}

		parentHead := parachaintypes.HeadData{Data: []byte{0x10, 0x10, 0x10}}
		paraHead := parachaintypes.HeadData{Data: []byte{0x09, 0x09, 0x09}}
		pvd, commitment := makeCommittedCandidate(t,
			paraID, relayBlockHash1, relayBlockNumber1, parentHead, paraHead, uint32(0))

		candidate := parachaintypes.HypotheticalCandidateComplete{
			ClaimedCandidateHash:      candidateHash,
			CommittedCandidateReceipt: commitment,
			PersistedValidationData:   pvd,
		}

		response := make(chan []*messages.HypotheticalMembershipResponseItem, 1)
		msg := messages.GetHypotheticalMembership{
			Candidates:               []parachaintypes.HypotheticalCandidate{candidate},
			FragmentChainRelayParent: nil,
			Response:                 response,
		}

		pp.getHypotheticalMembership(msg)
		out := <-response

		expected := []*messages.HypotheticalMembershipResponseItem{
			{
				HypotheticalCandidate:  candidate,
				HypotheticalMembership: []common.Hash{relayBlockHash1},
			},
		}

		require.Equal(t, expected, out)
	})

	t.Run("candidate_at_best_chain", func(t *testing.T) {
		paraID := parachaintypes.ParaID(1000)
		relayBlockHash1 := common.Hash([32]byte{0xAB, 0xCD})
		relayBlockNumber1 := uint32(1)

		// relayBlockHash2 := common.Hash([]byte{0xFF, 0xFE})

		candidateHash := parachaintypes.CandidateHash{
			Value: common.Hash([32]byte{0x01, 0x02, 0x03}),
		}

		fcForParaID := &fragmentChain{
			unconnected: &candidateStorage{
				byCandidateHash: map[parachaintypes.CandidateHash]*candidateEntry{},
			},
			bestChain: &backedChain{
				candidates: map[parachaintypes.CandidateHash]struct{}{
					candidateHash: {},
				},
			},
		}

		pp := &ProspectiveParachains{
			view: &view{
				activeLeaves: map[common.Hash]bool{
					relayBlockHash1: true,
				},

				perRelayParent: map[common.Hash]*relayParentData{
					relayBlockHash1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
							paraID: fcForParaID,
						},
					},
				},
			},
		}

		parentHead := parachaintypes.HeadData{Data: []byte{0x10, 0x10, 0x10}}
		paraHead := parachaintypes.HeadData{Data: []byte{0x09, 0x09, 0x09}}
		pvd, commitment := makeCommittedCandidate(t,
			paraID, relayBlockHash1, relayBlockNumber1, parentHead, paraHead, uint32(0))

		candidate := parachaintypes.HypotheticalCandidateComplete{
			ClaimedCandidateHash:      candidateHash,
			CommittedCandidateReceipt: commitment,
			PersistedValidationData:   pvd,
		}

		response := make(chan []*messages.HypotheticalMembershipResponseItem, 1)
		msg := messages.GetHypotheticalMembership{
			Candidates:               []parachaintypes.HypotheticalCandidate{candidate},
			FragmentChainRelayParent: nil,
			Response:                 response,
		}

		pp.getHypotheticalMembership(msg)
		out := <-response

		expected := []*messages.HypotheticalMembershipResponseItem{
			{
				HypotheticalCandidate:  candidate,
				HypotheticalMembership: []common.Hash{relayBlockHash1},
			},
		}

		require.Equal(t, expected, out)
	})

	t.Run("failed_to_check_potential", func(t *testing.T) {
		paraID := parachaintypes.ParaID(1000)
		relayBlockHash1 := common.Hash([32]byte{0xAB, 0xCD})
		relayBlockNumber1 := uint32(1)

		// relayBlockHash2 := common.Hash([]byte{0xFF, 0xFE})

		candidateHash := parachaintypes.CandidateHash{
			Value: common.Hash([32]byte{0x01, 0x02, 0x03}),
		}

		fcForParaID := &fragmentChain{
			unconnected: &candidateStorage{
				byCandidateHash: map[parachaintypes.CandidateHash]*candidateEntry{},
			},
			bestChain: &backedChain{
				candidates: map[parachaintypes.CandidateHash]struct{}{},
			},
		}

		pp := &ProspectiveParachains{
			view: &view{
				activeLeaves: map[common.Hash]bool{
					relayBlockHash1: true,
				},

				perRelayParent: map[common.Hash]*relayParentData{
					relayBlockHash1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
							paraID: fcForParaID,
						},
					},
				},
			},
		}

		// making parentHead and paraHead equals causes a cycle, which is not allowed
		parentHead := parachaintypes.HeadData{Data: []byte{0x10, 0x10, 0x10}}
		paraHead := parachaintypes.HeadData{Data: []byte{0x10, 0x10, 0x10}}
		pvd, commitment := makeCommittedCandidate(t,
			paraID, relayBlockHash1, relayBlockNumber1, parentHead, paraHead, uint32(0))

		candidate := parachaintypes.HypotheticalCandidateComplete{
			ClaimedCandidateHash:      candidateHash,
			CommittedCandidateReceipt: commitment,
			PersistedValidationData:   pvd,
		}

		response := make(chan []*messages.HypotheticalMembershipResponseItem, 1)
		msg := messages.GetHypotheticalMembership{
			Candidates:               []parachaintypes.HypotheticalCandidate{candidate},
			FragmentChainRelayParent: nil,
			Response:                 response,
		}

		pp.getHypotheticalMembership(msg)
		out := <-response

		// should not have any hypothetical membership
		expected := []*messages.HypotheticalMembershipResponseItem{
			{
				HypotheticalCandidate:  candidate,
				HypotheticalMembership: []common.Hash{},
			},
		}

		require.Equal(t, expected, out)
	})

	t.Run("successfully_check_potential_single_leaf", func(t *testing.T) {
		// setup the current contraints
		relayParent := &relayChainBlockInfo{
			Hash:        common.Hash([32]byte{0xAB, 0xCD}),
			StorageRoot: common.Hash([32]byte{}),
			Number:      parachaintypes.BlockNumber(9),
		}

		parentHead := parachaintypes.HeadData{Data: []byte{0x10, 0x10, 0x10}}

		ancestors := []relayChainBlockInfo{
			{
				Hash:   common.Hash{0x03},
				Number: parachaintypes.BlockNumber(8),
			},
		}

		baseConstraints := &parachaintypes.VStagingConstraints{
			MinRelayParentNumber: 8,
			RequiredParent:       parentHead,
			MaxPoVSize:           MaxPoVSize,
			DMPRemainingMessages: []parachaintypes.BlockNumber{1},
			ValidationCodeHash: parachaintypes.ValidationCodeHash(
				common.BytesToHash(bytes.Repeat([]byte{42}, 32)),
			),
		}

		mockScope, err := newScopeWithAncestors(
			*relayParent,
			baseConstraints,
			nil,
			10,
			ancestors,
		)
		assert.NoError(t, err)

		candidateStorage := newCandidateStorage()
		paraID := parachaintypes.ParaID(1000)

		candidateHash := parachaintypes.CandidateHash{
			Value: common.Hash([32]byte{0x01, 0x02, 0x03}),
		}

		fcForParaID := newFragmentChain(mockScope, candidateStorage)

		pp := &ProspectiveParachains{
			view: &view{
				activeLeaves: map[common.Hash]bool{
					relayParent.Hash: true,
				},

				perRelayParent: map[common.Hash]*relayParentData{
					relayParent.Hash: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
							paraID: fcForParaID,
						},
					},
				},
			},
		}

		paraHead := parachaintypes.HeadData{Data: []byte{0x09, 0x09, 0x09}}
		pvd, commitment := makeCommittedCandidate(t,
			paraID, relayParent.Hash, uint32(relayParent.Number),
			parentHead, paraHead, uint32(relayParent.Number))

		candidate := parachaintypes.HypotheticalCandidateComplete{
			ClaimedCandidateHash:      candidateHash,
			CommittedCandidateReceipt: commitment,
			PersistedValidationData:   pvd,
		}

		response := make(chan []*messages.HypotheticalMembershipResponseItem, 1)
		msg := messages.GetHypotheticalMembership{
			Candidates:               []parachaintypes.HypotheticalCandidate{candidate},
			FragmentChainRelayParent: nil,
			Response:                 response,
		}

		pp.getHypotheticalMembership(msg)
		out := <-response

		expected := []*messages.HypotheticalMembershipResponseItem{
			{
				HypotheticalCandidate:  candidate,
				HypotheticalMembership: []common.Hash{relayParent.Hash},
			},
		}

		require.Equal(t, expected, out)
	})

	t.Run("successfully_check_potential_two_leaves", func(t *testing.T) {
		parentHead := parachaintypes.HeadData{Data: []byte{0x10, 0x10, 0x10}}

		// setup the current contraints for relay parent 9
		relayParent9 := &relayChainBlockInfo{
			Hash:        common.Hash([32]byte{0xAB, 0xCD}),
			StorageRoot: common.Hash([32]byte{}),
			Number:      parachaintypes.BlockNumber(9),
		}

		baseConstraints := &parachaintypes.VStagingConstraints{
			MinRelayParentNumber: 8,
			RequiredParent:       parentHead,
			MaxPoVSize:           MaxPoVSize,
			DMPRemainingMessages: []parachaintypes.BlockNumber{1},
			ValidationCodeHash: parachaintypes.ValidationCodeHash(
				common.BytesToHash(bytes.Repeat([]byte{42}, 32)),
			),
		}

		mockScope, err := newScopeWithAncestors(
			*relayParent9,
			baseConstraints,
			nil,
			10,
			[]relayChainBlockInfo{
				{
					Hash:   common.Hash{0x03},
					Number: parachaintypes.BlockNumber(8),
				},
			},
		)
		assert.NoError(t, err)

		candidateStorage := newCandidateStorage()
		paraID := parachaintypes.ParaID(1000)

		fcForParaIDRelayParent9 := newFragmentChain(mockScope, candidateStorage)

		// setup the current constraints for parent 10
		relayParent10 := &relayChainBlockInfo{
			Hash:        common.Hash([32]byte{0xEC, 0xDE}),
			StorageRoot: common.Hash([32]byte{}),
			Number:      parachaintypes.BlockNumber(10),
		}

		baseConstraints = &parachaintypes.VStagingConstraints{
			MinRelayParentNumber: 8,
			RequiredParent:       parentHead,
			MaxPoVSize:           MaxPoVSize,
			DMPRemainingMessages: []parachaintypes.BlockNumber{1},
			ValidationCodeHash: parachaintypes.ValidationCodeHash(
				common.BytesToHash(bytes.Repeat([]byte{42}, 32)),
			),
		}

		mockScope, err = newScopeWithAncestors(
			*relayParent10,
			baseConstraints,
			nil,
			10,
			[]relayChainBlockInfo{
				// includes the relay parent 9 in the  list of ancestors
				*relayParent9,
				{
					Hash:   common.Hash{0x03},
					Number: parachaintypes.BlockNumber(8),
				},
			},
		)
		assert.NoError(t, err)

		fcForParaIDRelayParent10 := newFragmentChain(mockScope, newCandidateStorage())

		pp := &ProspectiveParachains{
			view: &view{
				activeLeaves: map[common.Hash]bool{
					relayParent9.Hash:  true,
					relayParent10.Hash: true,
				},

				perRelayParent: map[common.Hash]*relayParentData{
					relayParent9.Hash: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
							paraID: fcForParaIDRelayParent9,
						},
					},
					relayParent10.Hash: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
							paraID: fcForParaIDRelayParent10,
						},
					},
				},
			},
		}

		paraHead := parachaintypes.HeadData{Data: []byte{0x09, 0x09, 0x09}}
		pvd, commitment := makeCommittedCandidate(t,
			paraID, relayParent9.Hash, uint32(relayParent9.Number),
			parentHead, paraHead, uint32(relayParent9.Number))

		candidateHash := parachaintypes.CandidateHash{
			Value: common.Hash([32]byte{0x01, 0x02, 0x03}),
		}

		candidate := parachaintypes.HypotheticalCandidateComplete{
			ClaimedCandidateHash:      candidateHash,
			CommittedCandidateReceipt: commitment,
			PersistedValidationData:   pvd,
		}

		response := make(chan []*messages.HypotheticalMembershipResponseItem, 1)
		msg := messages.GetHypotheticalMembership{
			Candidates:               []parachaintypes.HypotheticalCandidate{candidate},
			FragmentChainRelayParent: nil,
			Response:                 response,
		}

		pp.getHypotheticalMembership(msg)
		out := <-response

		// candidate is a potential in both relay parents
		expected := []*messages.HypotheticalMembershipResponseItem{
			{
				HypotheticalCandidate: candidate,
				HypotheticalMembership: []common.Hash{
					relayParent9.Hash,
					relayParent10.Hash,
				},
			},
		}

		require.Len(t, out, 1)

		sort.Slice(out[0].HypotheticalMembership, func(i, j int) bool {
			return bytes.Compare(
				out[0].HypotheticalMembership[i].ToBytes(),
				out[0].HypotheticalMembership[j].ToBytes()) == -1
		})

		sort.Slice(expected[0].HypotheticalMembership, func(i, j int) bool {
			return bytes.Compare(
				expected[0].HypotheticalMembership[i].ToBytes(),
				expected[0].HypotheticalMembership[j].ToBytes()) == -1
		})

		require.Equal(t, expected, out)
	})

	t.Run("target_relay_parent_successfully_check_potential", func(t *testing.T) {
		parentHead := parachaintypes.HeadData{Data: []byte{0x10, 0x10, 0x10}}

		// setup the current contraints for relay parent 9
		relayParent9 := &relayChainBlockInfo{
			Hash:        common.Hash([32]byte{0xAB, 0xCD}),
			StorageRoot: common.Hash([32]byte{}),
			Number:      parachaintypes.BlockNumber(9),
		}

		baseConstraints := &parachaintypes.VStagingConstraints{
			MinRelayParentNumber: 8,
			RequiredParent:       parentHead,
			MaxPoVSize:           MaxPoVSize,
			DMPRemainingMessages: []parachaintypes.BlockNumber{1},
			ValidationCodeHash: parachaintypes.ValidationCodeHash(
				common.BytesToHash(bytes.Repeat([]byte{42}, 32)),
			),
		}

		mockScope, err := newScopeWithAncestors(
			*relayParent9,
			baseConstraints,
			nil,
			10,
			[]relayChainBlockInfo{
				{
					Hash:   common.Hash{0x03},
					Number: parachaintypes.BlockNumber(8),
				},
			},
		)
		assert.NoError(t, err)

		candidateStorage := newCandidateStorage()
		paraID := parachaintypes.ParaID(1000)

		fcForParaIDRelayParent9 := newFragmentChain(mockScope, candidateStorage)

		// setup the current constraints for parent 10
		relayParent10 := &relayChainBlockInfo{
			Hash:        common.Hash([32]byte{0xEC, 0xDE}),
			StorageRoot: common.Hash([32]byte{}),
			Number:      parachaintypes.BlockNumber(10),
		}

		baseConstraints = &parachaintypes.VStagingConstraints{
			MinRelayParentNumber: 8,
			RequiredParent:       parentHead,
			MaxPoVSize:           MaxPoVSize,
			DMPRemainingMessages: []parachaintypes.BlockNumber{1},
			ValidationCodeHash: parachaintypes.ValidationCodeHash(
				common.BytesToHash(bytes.Repeat([]byte{42}, 32)),
			),
		}

		mockScope, err = newScopeWithAncestors(
			*relayParent10,
			baseConstraints,
			nil,
			10,
			[]relayChainBlockInfo{
				// includes the relay parent 9 in the  list of ancestors
				*relayParent9,
				{
					Hash:   common.Hash{0x03},
					Number: parachaintypes.BlockNumber(8),
				},
			},
		)
		assert.NoError(t, err)

		fcForParaIDRelayParent10 := newFragmentChain(mockScope, newCandidateStorage())

		pp := &ProspectiveParachains{
			view: &view{
				activeLeaves: map[common.Hash]bool{
					relayParent9.Hash:  true,
					relayParent10.Hash: true,
				},

				perRelayParent: map[common.Hash]*relayParentData{
					relayParent9.Hash: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
							paraID: fcForParaIDRelayParent9,
						},
					},
					relayParent10.Hash: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
							paraID: fcForParaIDRelayParent10,
						},
					},
				},
			},
		}

		paraHead := parachaintypes.HeadData{Data: []byte{0x09, 0x09, 0x09}}
		pvd, commitment := makeCommittedCandidate(t,
			paraID, relayParent9.Hash, uint32(relayParent9.Number),
			parentHead, paraHead, uint32(relayParent9.Number))

		candidateHash := parachaintypes.CandidateHash{
			Value: common.Hash([32]byte{0x01, 0x02, 0x03}),
		}

		candidate := parachaintypes.HypotheticalCandidateComplete{
			ClaimedCandidateHash:      candidateHash,
			CommittedCandidateReceipt: commitment,
			PersistedValidationData:   pvd,
		}

		response := make(chan []*messages.HypotheticalMembershipResponseItem, 1)
		msg := messages.GetHypotheticalMembership{
			Candidates:               []parachaintypes.HypotheticalCandidate{candidate},
			FragmentChainRelayParent: &relayParent10.Hash,
			Response:                 response,
		}

		pp.getHypotheticalMembership(msg)
		out := <-response

		// candidate is a potential in both relay parents
		expected := []*messages.HypotheticalMembershipResponseItem{
			{
				HypotheticalCandidate: candidate,
				HypotheticalMembership: []common.Hash{
					relayParent10.Hash,
				},
			},
		}

		require.Equal(t, expected, out)
	})
}

func markCandidatedBacked(
	t *testing.T,
	overseerToSubsystem chan any,
	candidate parachaintypes.CommittedCandidateReceiptV2,
) {
	hash, err := candidate.Hash()

	assert.NoError(t, err)

	msg := messages.CandidateBacked{
		ParaID:        candidate.Descriptor.ParaID,
		CandidateHash: parachaintypes.CandidateHash{Value: hash},
	}

	overseerToSubsystem <- msg
}

func TestHandleBacked(
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
	candidateRelayParentNumber := uint32(0)

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

	prospectiveParachains := NewProspectiveParachains(subsystemToOverseer, nil)

	relayParent := relayChainBlockInfo{
		Hash:        candidateRelayParent,
		Number:      0,
		StorageRoot: common.Hash{0x00},
	}

	baseConstraints := &parachaintypes.VStagingConstraints{
		RequiredParent:       parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x01}, 32)},
		MinRelayParentNumber: 0,
		MaxHeadDataSize:      1_000_000,
		ValidationCodeHash:   validationCodeHash,
		MaxPoVSize:           1000000,
	}

	scope, err := newScopeWithAncestors(relayParent, baseConstraints, nil, 10, nil)
	assert.NoError(t, err)

	prospectiveParachains.view.perRelayParent[candidateRelayParent] = &relayParentData{
		fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
			paraId: newFragmentChain(scope, newCandidateStorage()),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		prospectiveParachains.Run(ctx, overseerToSubsystem)
	}(&wg)

	introduceSecondedCandidate(t, overseerToSubsystem, candidate, pvd)

	markCandidatedBacked(t, overseerToSubsystem, candidate)

	// cancel the subsystem context to shutdown and evaluate the
	// subsystem state
	cancel()

	wg.Wait()

	rpData, ok := prospectiveParachains.view.perRelayParent[candidateRelayParent]
	require.True(t, ok)

	chains := rpData.fragmentChains

	fragmentChain, exist := chains[paraId]

	require.True(t, exist)

	hash, err := candidate.Hash()
	assert.NoError(t, err)

	isCandidateBacked := fragmentChain.isCandidateBacked(parachaintypes.CandidateHash{Value: hash})

	require.True(t, isCandidateBacked)

	hashes := fragmentChain.bestChainVec()

	require.Len(t, hashes, 1)

	require.Equal(t, hashes[0], parachaintypes.CandidateHash{Value: hash})
}

func TestActivateLeafSignalHandler(t *testing.T) {
	cases := map[string]struct {
		setup  func(*testing.T) (*ProspectiveParachains, parachaintypes.ActiveLeavesUpdateSignal)
		assert func(*testing.T, parachaintypes.ActiveLeavesUpdateSignal, *ProspectiveParachains)
	}{
		"no_ancestry_and_no_previous_fragment_chains": {
			setup: func(t *testing.T) (*ProspectiveParachains, parachaintypes.ActiveLeavesUpdateSignal) {
				parentHeader := &types.Header{
					ParentHash: common.Hash{0x00},
					Number:     0,
					StateRoot:  common.Hash{0x00},
				}

				activeLeafHeader := &types.Header{
					ParentHash: parentHeader.Hash(),
					Number:     1,
					StateRoot:  common.Hash{0x00},
				}
				activeLeafHash := activeLeafHeader.Hash()

				claimQueue := map[parachaintypes.CoreIndex][]parachaintypes.ParaID{
					{Index: 0}: {
						parachaintypes.ParaID(1),
					},
				}

				ctrl := gomock.NewController(t)
				mockRuntime := NewMockInstance(ctrl)
				mockRuntime.EXPECT().
					ParachainHostClaimQueue().
					Return(claimQueue, nil)
				mockRuntime.EXPECT().
					ParachainHostSessionIndexForChild().
					Return(parachaintypes.SessionIndex(1), nil)
				mockRuntime.EXPECT().
					ParachainHostSchedulingLookAhead().
					Return(uint32(3), nil)

				constraints := dummyConstraintsV2(0,
					[]parachaintypes.BlockNumber{0},
					parachaintypes.HeadData{Data: []byte{0x00}},
					parachaintypes.ValidationCodeHash{0x00},
				)
				mockRuntime.EXPECT().
					ParachainHostBackingConstraints(parachaintypes.ParaID(1)).
					Return(constraints, nil)

				mockRuntime.EXPECT().
					ParachainHostCandidatesPendingAvailability(parachaintypes.ParaID(1)).
					Return([]parachaintypes.CommittedCandidateReceiptV2{}, nil)

				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().
					GetRuntime(activeLeafHash).
					Return(mockRuntime, nil)

				mockBlockState.EXPECT().
					GetHeader(activeLeafHash).
					Return(activeLeafHeader, nil)

				// mocking fetchAncestry get header call
				// here we return a different session index
				// for the parent header
				mockBlockState.EXPECT().
					GetHeader(activeLeafHeader.ParentHash).
					Return(parentHeader, nil)

				innerRuntimeMock := NewMockInstance(ctrl)
				innerRuntimeMock.EXPECT().
					ParachainHostSessionIndexForChild().
					Return(parachaintypes.SessionIndex(0), nil)

				mockBlockState.EXPECT().
					GetRuntime(activeLeafHeader.ParentHash).
					Return(innerRuntimeMock, nil)

				subsystemToOverseer := make(chan any)

				pp := NewProspectiveParachains(subsystemToOverseer, mockBlockState)

				msg := parachaintypes.ActiveLeavesUpdateSignal{
					Activated: &parachaintypes.ActivatedLeaf{
						Hash:   activeLeafHash,
						Number: uint32(activeLeafHeader.Number),
					},
				}

				return pp, msg
			},
			assert: func(t *testing.T, msg parachaintypes.ActiveLeavesUpdateSignal, pp *ProspectiveParachains) {
				require.True(t, pp.view.activeLeaves[msg.Activated.Hash])

				rp, ok := pp.view.perRelayParent[msg.Activated.Hash]
				require.True(t, ok)

				fc, ok := rp.fragmentChains[parachaintypes.ParaID(1)]
				require.True(t, ok)
				require.NotNil(t, fc)
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			pp, activeLeafMsg := tt.setup(t)

			err := pp.ProcessActiveLeavesUpdateSignal(activeLeafMsg)
			require.NoError(t, err)

			tt.assert(t, activeLeafMsg, pp)
		})
	}
}
