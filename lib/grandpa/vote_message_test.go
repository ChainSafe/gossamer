// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	errTestError = errors.New("test dummy error")
	dummyHash    = common.NewHash([]byte{1})
)

func TestService_reportEquivocation(t *testing.T) {
	t.Parallel()
	keyOwnershipProof := types.GrandpaOpaqueKeyOwnershipProof{1}
	signedVote := &types.GrandpaSignedVote{
		Vote:        *testVote,
		Signature:   testSignature,
		AuthorityID: testAuthorityID,
	}

	signedVote2 := &types.GrandpaSignedVote{
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
	err := equivocationVote.Set(types.PreVote(grandpaEquivocation))
	require.NoError(t, err)

	equivocationProof := types.GrandpaEquivocationProof{
		SetID:        uint64(1),
		Equivocation: *equivocationVote,
	}
	type args struct {
		stage        Subround
		existingVote *SignedVote
		currentVote  *SignedVote
	}
	tests := []struct {
		name           string
		service        *Service
		serviceBuilder func(ctrl *gomock.Controller) *Service
		args           args
		expErr         error
		expErrMsg      string
	}{
		{
			name: "get_setID_error",
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				mockGrandpaStateSetIDErr := NewMockGrandpaState(ctrl)
				mockGrandpaStateSetIDErr.EXPECT().GetCurrentSetID().Return(uint64(0), errTestError)
				return &Service{grandpaState: mockGrandpaStateSetIDErr}
			},
			expErr:    errTestError,
			expErrMsg: "getting authority set id: test dummy error",
		},
		{
			name: "get_latest_round_error",
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				mockGrandpaStateRoundErr := NewMockGrandpaState(ctrl)
				mockGrandpaStateRoundErr.EXPECT().GetCurrentSetID().Return(uint64(1), nil)
				mockGrandpaStateRoundErr.EXPECT().GetLatestRound().Return(uint64(0), errTestError)
				return &Service{grandpaState: mockGrandpaStateRoundErr}
			},
			expErr:    errTestError,
			expErrMsg: "getting latest round: test dummy error",
		},
		{
			name: "get_runtime_error",
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				mockGrandpaStateOk := NewMockGrandpaState(ctrl)
				mockGrandpaStateOk.EXPECT().GetCurrentSetID().Return(uint64(1), nil)
				mockGrandpaStateOk.EXPECT().GetLatestRound().Return(uint64(1), nil)
				mockBlockStateGetRuntimeErr := NewMockBlockState(ctrl)
				mockBlockStateGetRuntimeErr.EXPECT().BestBlockHash().Return(dummyHash)
				mockBlockStateGetRuntimeErr.EXPECT().GetRuntime(dummyHash).Return(nil, errTestError)
				return &Service{
					grandpaState: mockGrandpaStateOk,
					blockState:   mockBlockStateGetRuntimeErr,
				}
			},
			args:      args{existingVote: signedVote},
			expErr:    errTestError,
			expErrMsg: "getting runtime: test dummy error",
		},
		{
			name: "get_key_ownership_proof_error",
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				mockRuntimeInstanceGenerateProofErr := NewMockInstance(ctrl)
				mockRuntimeInstanceGenerateProofErr.EXPECT().GrandpaGenerateKeyOwnershipProof(uint64(1), testAuthorityID).
					Return(types.GrandpaOpaqueKeyOwnershipProof{}, errTestError)
				mockGrandpaStateOk := NewMockGrandpaState(ctrl)
				mockGrandpaStateOk.EXPECT().GetCurrentSetID().Return(uint64(1), nil)
				mockGrandpaStateOk.EXPECT().GetLatestRound().Return(uint64(1), nil)
				mockBlockStateGenerateProofErr := NewMockBlockState(ctrl)
				mockBlockStateGenerateProofErr.EXPECT().BestBlockHash().Return(dummyHash)
				mockBlockStateGenerateProofErr.EXPECT().GetRuntime(dummyHash).
					Return(mockRuntimeInstanceGenerateProofErr, nil)
				return &Service{
					grandpaState: mockGrandpaStateOk,
					blockState:   mockBlockStateGenerateProofErr,
				}
			},
			args:      args{existingVote: signedVote},
			expErr:    errTestError,
			expErrMsg: "getting key ownership proof: test dummy error",
		},
		{
			name: "invalid_stage",
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				mockRuntimeInstanceReportEquivocationErr := NewMockInstance(ctrl)
				mockRuntimeInstanceReportEquivocationErr.EXPECT().GrandpaGenerateKeyOwnershipProof(uint64(1), testAuthorityID).
					Return(keyOwnershipProof, nil)
				mockGrandpaStateOk := NewMockGrandpaState(ctrl)
				mockGrandpaStateOk.EXPECT().GetCurrentSetID().Return(uint64(1), nil)
				mockGrandpaStateOk.EXPECT().GetLatestRound().Return(uint64(1), nil)
				mockBlockStateReportEquivocationErr := NewMockBlockState(ctrl)
				mockBlockStateReportEquivocationErr.EXPECT().BestBlockHash().Return(dummyHash)
				mockBlockStateReportEquivocationErr.EXPECT().GetRuntime(dummyHash).
					Return(mockRuntimeInstanceReportEquivocationErr, nil)
				return &Service{
					grandpaState: mockGrandpaStateOk,
					blockState:   mockBlockStateReportEquivocationErr,
				}
			},
			args: args{
				stage:        primaryProposal,
				existingVote: signedVote,
				currentVote:  signedVote2,
			},
			expErr:    errInvalidEquivocationStage,
			expErrMsg: "invalid stage for equivocating: primaryProposal (2)",
		},
		{
			name: "submit_equivocation_proof_error",
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				mockRuntimeInstanceReportEquivocationErr := NewMockInstance(ctrl)
				mockRuntimeInstanceReportEquivocationErr.EXPECT().GrandpaGenerateKeyOwnershipProof(uint64(1), testAuthorityID).
					Return(keyOwnershipProof, nil)
				mockRuntimeInstanceReportEquivocationErr.EXPECT().
					GrandpaSubmitReportEquivocationUnsignedExtrinsic(equivocationProof, keyOwnershipProof).
					Return(errTestError)
				mockGrandpaStateOk := NewMockGrandpaState(ctrl)
				mockGrandpaStateOk.EXPECT().GetCurrentSetID().Return(uint64(1), nil)
				mockGrandpaStateOk.EXPECT().GetLatestRound().Return(uint64(1), nil)
				mockBlockStateReportEquivocationErr := NewMockBlockState(ctrl)
				mockBlockStateReportEquivocationErr.EXPECT().BestBlockHash().Return(dummyHash)
				mockBlockStateReportEquivocationErr.EXPECT().GetRuntime(dummyHash).
					Return(mockRuntimeInstanceReportEquivocationErr, nil)
				return &Service{
					grandpaState: mockGrandpaStateOk,
					blockState:   mockBlockStateReportEquivocationErr,
				}
			},
			args: args{
				stage:        prevote,
				existingVote: signedVote,
				currentVote:  signedVote2,
			},
			expErr:    errTestError,
			expErrMsg: "submitting grandpa equivocation report to runtime: test dummy error",
		},
		{
			name: "valid_path",
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				mockRuntimeInstanceOk := NewMockInstance(ctrl)
				mockRuntimeInstanceOk.EXPECT().GrandpaGenerateKeyOwnershipProof(uint64(1), testAuthorityID).
					Return(keyOwnershipProof, nil)
				mockRuntimeInstanceOk.EXPECT().
					GrandpaSubmitReportEquivocationUnsignedExtrinsic(equivocationProof, keyOwnershipProof).
					Return(nil)
				mockGrandpaStateOk := NewMockGrandpaState(ctrl)
				mockGrandpaStateOk.EXPECT().GetCurrentSetID().Return(uint64(1), nil)
				mockGrandpaStateOk.EXPECT().GetLatestRound().Return(uint64(1), nil)
				mockBlockStateOk := NewMockBlockState(ctrl)
				mockBlockStateOk.EXPECT().BestBlockHash().Return(dummyHash)
				mockBlockStateOk.EXPECT().GetRuntime(dummyHash).Return(mockRuntimeInstanceOk, nil)
				return &Service{
					grandpaState: mockGrandpaStateOk,
					blockState:   mockBlockStateOk,
				}
			},
			args: args{
				stage:        prevote,
				existingVote: signedVote,
				currentVote:  signedVote2,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			service := tt.serviceBuilder(ctrl)
			err := service.reportEquivocation(tt.args.stage, tt.args.existingVote, tt.args.currentVote)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
		})
	}
}
