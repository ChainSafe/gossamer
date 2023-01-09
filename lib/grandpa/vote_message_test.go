// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errTestError  = errors.New("test dummy error")
	testDummyHash = common.NewHash([]byte{1})
)

func TestService_reportEquivocation(t *testing.T) {
	t.Parallel()
	keyOwnershipProof := types.OpaqueKeyOwnershipProof{1}
	testSignedVote := &types.GrandpaSignedVote{
		Vote:        *testVote,
		Signature:   testSignature,
		AuthorityID: testAuthorityID,
	}

	testSignedVote2 := &types.GrandpaSignedVote{
		Vote:        *testVote,
		Signature:   testSignature,
		AuthorityID: testAuthorityID,
	}

	grandpaEquivocation := types.GrandpaEquivocation{
		RoundNumber:     1,
		ID:              testAuthorityID,
		FirstVote:       *testVote,
		FirstSignature:  testSignature,
		SecondVote:      *testVote,
		SecondSignature: testSignature,
	}

	equivocationVote := types.NewGrandpaEquivocation()
	err := equivocationVote.Set(types.PreVoteEquivocation(grandpaEquivocation))
	require.NoError(t, err)

	equivocationProof := types.GrandpaEquivocationProof{
		SetID:        uint64(1),
		Equivocation: *equivocationVote,
	}

	ctrl := gomock.NewController(t)
	mockGrandpaStateSetIDErr := NewMockGrandpaState(ctrl)
	mockGrandpaStateSetIDErr.EXPECT().GetCurrentSetID().Return(uint64(0), errTestError)

	mockGrandpaStateRoundErr := NewMockGrandpaState(ctrl)
	mockGrandpaStateRoundErr.EXPECT().GetCurrentSetID().Return(uint64(1), nil)
	mockGrandpaStateRoundErr.EXPECT().GetLatestRound().Return(uint64(0), errTestError)

	mockGrandpaStateOk := NewMockGrandpaState(ctrl)
	mockGrandpaStateOk.EXPECT().GetCurrentSetID().Return(uint64(1), nil).Times(4)
	mockGrandpaStateOk.EXPECT().GetLatestRound().Return(uint64(1), nil).Times(4)

	mockRuntimeInstanceGenerateProofErr := NewMockRuntimeInstance(ctrl)
	mockRuntimeInstanceGenerateProofErr.EXPECT().GrandpaGenerateKeyOwnershipProof(uint64(1), testAuthorityID).
		Return(types.OpaqueKeyOwnershipProof{}, errTestError)

	mockRuntimeInstanceReportEquivocationErr := NewMockRuntimeInstance(ctrl)
	mockRuntimeInstanceReportEquivocationErr.EXPECT().GrandpaGenerateKeyOwnershipProof(uint64(1), testAuthorityID).
		Return(keyOwnershipProof, nil)
	mockRuntimeInstanceReportEquivocationErr.EXPECT().
		GrandpaSubmitReportEquivocationUnsignedExtrinsic(equivocationProof, keyOwnershipProof).
		Return(errTestError)

	mockRuntimeInstanceOk := NewMockRuntimeInstance(ctrl)
	mockRuntimeInstanceOk.EXPECT().GrandpaGenerateKeyOwnershipProof(uint64(1), testAuthorityID).
		Return(keyOwnershipProof, nil)
	mockRuntimeInstanceOk.EXPECT().
		GrandpaSubmitReportEquivocationUnsignedExtrinsic(equivocationProof, keyOwnershipProof).
		Return(nil)

	mockBlockStateGetRuntimeErr := NewMockBlockState(ctrl)
	mockBlockStateGetRuntimeErr.EXPECT().BestBlockHash().Return(testDummyHash)
	mockBlockStateGetRuntimeErr.EXPECT().GetRuntime(testDummyHash).Return(nil, errTestError)

	mockBlockStateGenerateProofErr := NewMockBlockState(ctrl)
	mockBlockStateGenerateProofErr.EXPECT().BestBlockHash().Return(testDummyHash)
	mockBlockStateGenerateProofErr.EXPECT().GetRuntime(testDummyHash).
		Return(mockRuntimeInstanceGenerateProofErr, nil)

	mockBlockStateReportEquivocationErr := NewMockBlockState(ctrl)
	mockBlockStateReportEquivocationErr.EXPECT().BestBlockHash().Return(testDummyHash)
	mockBlockStateReportEquivocationErr.EXPECT().GetRuntime(testDummyHash).
		Return(mockRuntimeInstanceReportEquivocationErr, nil)

	mockBlockStateOk := NewMockBlockState(ctrl)
	mockBlockStateOk.EXPECT().BestBlockHash().Return(testDummyHash)
	mockBlockStateOk.EXPECT().GetRuntime(testDummyHash).Return(mockRuntimeInstanceOk, nil)

	type args struct {
		stage        Subround
		existingVote *SignedVote
		currentVote  *SignedVote
	}
	tests := []struct {
		name      string
		service   *Service
		args      args
		expErr    error
		expErrMsg string
	}{
		{
			name:      "get setID error",
			service:   &Service{grandpaState: mockGrandpaStateSetIDErr},
			expErr:    errTestError,
			expErrMsg: "getting authority set id: test dummy error",
		},
		{
			name:      "get latest round error",
			service:   &Service{grandpaState: mockGrandpaStateRoundErr},
			expErr:    errTestError,
			expErrMsg: "getting latest round: test dummy error",
		},
		{
			name: "get runtime error",
			service: &Service{
				grandpaState: mockGrandpaStateOk,
				blockState:   mockBlockStateGetRuntimeErr,
			},
			args:      args{existingVote: testSignedVote},
			expErr:    errTestError,
			expErrMsg: "getting runtime: test dummy error",
		},
		{
			name: "get key ownership proof error",
			service: &Service{
				grandpaState: mockGrandpaStateOk,
				blockState:   mockBlockStateGenerateProofErr,
			},
			args:      args{existingVote: testSignedVote},
			expErr:    errTestError,
			expErrMsg: "getting key ownership proof: test dummy error",
		},
		{
			name: "submit equivocation proof error",
			service: &Service{
				grandpaState: mockGrandpaStateOk,
				blockState:   mockBlockStateReportEquivocationErr,
			},
			args: args{
				stage:        prevote,
				existingVote: testSignedVote,
				currentVote:  testSignedVote2,
			},
			expErr:    errTestError,
			expErrMsg: "reporting equivocation: test dummy error",
		},
		{
			name: "valid path",
			service: &Service{
				grandpaState: mockGrandpaStateOk,
				blockState:   mockBlockStateOk,
			},
			args: args{
				stage:        prevote,
				existingVote: testSignedVote,
				currentVote:  testSignedVote2,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := tt.service
			err := s.reportEquivocation(tt.args.stage, tt.args.existingVote, tt.args.currentVote)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
		})
	}
}
