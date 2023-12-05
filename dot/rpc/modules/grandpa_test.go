// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGrandpaModule_ProveFinality(t *testing.T) {
	t.Parallel()

	mockError := errors.New("test mock error")

	tests := map[string]struct {
		blockAPIBuilder func(ctrl *gomock.Controller) BlockAPI
		request         *ProveFinalityRequest
		expErr          error
		exp             ProveFinalityResponse
	}{
		"error_during_get_hash_by_number": {
			blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
				mockBlockAPI := NewMockBlockAPI(ctrl)
				mockBlockAPI.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{}, mockError)
				return mockBlockAPI
			},
			request: &ProveFinalityRequest{
				BlockNumber: 1,
			},
			expErr: mockError,
		},
		"error_during_has_justification": {
			blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
				mockBlockAPI := NewMockBlockAPI(ctrl)
				mockBlockAPI.EXPECT().GetHashByNumber(uint(2)).Return(common.Hash{2}, nil)
				mockBlockAPI.EXPECT().HasJustification(common.Hash{2}).Return(false, mockError)
				return mockBlockAPI
			},
			request: &ProveFinalityRequest{
				BlockNumber: 2,
			},
			expErr: mockError,
		},
		"has_justification_is_false": {
			blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
				mockBlockAPI := NewMockBlockAPI(ctrl)
				mockBlockAPI.EXPECT().GetHashByNumber(uint(2)).Return(common.Hash{2}, nil)
				mockBlockAPI.EXPECT().HasJustification(common.Hash{2}).Return(false, nil)
				return mockBlockAPI
			},
			request: &ProveFinalityRequest{
				BlockNumber: 2,
			},
			exp: ProveFinalityResponse{"GRANDPA prove finality rpc failed: Block not covered by authority set changes"},
		},
		"error_during_getJustification": {
			blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
				mockBlockAPI := NewMockBlockAPI(ctrl)
				mockBlockAPI.EXPECT().GetHashByNumber(uint(3)).Return(common.Hash{3}, nil)
				mockBlockAPI.EXPECT().HasJustification(common.Hash{3}).Return(true, nil)
				mockBlockAPI.EXPECT().GetJustification(common.Hash{3}).Return(nil, mockError)
				return mockBlockAPI
			},
			request: &ProveFinalityRequest{
				BlockNumber: 3,
			},
			expErr: mockError,
		},
		"happy_path": {
			blockAPIBuilder: func(ctrl *gomock.Controller) BlockAPI {
				mockBlockAPI := NewMockBlockAPI(ctrl)
				mockBlockAPI.EXPECT().GetHashByNumber(uint(4)).Return(common.Hash{4}, nil)
				mockBlockAPI.EXPECT().HasJustification(common.Hash{4}).Return(true, nil)
				mockBlockAPI.EXPECT().GetJustification(common.Hash{4}).Return([]byte(`justification`), nil)
				return mockBlockAPI
			},
			request: &ProveFinalityRequest{
				BlockNumber: 4,
			},
			exp: ProveFinalityResponse{common.BytesToHex([]byte(`justification`))},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gm := &GrandpaModule{
				blockAPI: tt.blockAPIBuilder(ctrl),
			}
			res := ProveFinalityResponse(nil)
			err := gm.ProveFinality(nil, tt.request, &res)
			assert.Equal(t, tt.exp, res)
			if tt.expErr != nil {
				assert.ErrorContains(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGrandpaModule_RoundState(t *testing.T) {
	ctrl := gomock.NewController(t)

	var kr, _ = keystore.NewEd25519Keyring()
	var voters grandpa.Voters

	for _, k := range kr.Keys {
		voters = append(voters, types.GrandpaVoter{
			Key: *k.Public().(*ed25519.PublicKey),
			ID:  1,
		})
	}

	mockBlockAPI := mocks.NewMockBlockAPI(ctrl)
	mockBlockFinalityAPI := mocks.NewMockBlockFinalityAPI(ctrl)
	mockBlockFinalityAPI.EXPECT().GetVoters().Return(voters)
	mockBlockFinalityAPI.EXPECT().GetSetID().Return(uint64(0))
	mockBlockFinalityAPI.EXPECT().GetRound().Return(uint64(2))
	mockBlockFinalityAPI.EXPECT().PreVotes().Return([]ed25519.PublicKeyBytes{
		kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Bob().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Charlie().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Dave().Public().(*ed25519.PublicKey).AsBytes(),
	})
	mockBlockFinalityAPI.EXPECT().PreCommits().Return([]ed25519.PublicKeyBytes{
		kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Bob().Public().(*ed25519.PublicKey).AsBytes(),
	})

	type fields struct {
		blockAPI         BlockAPI
		blockFinalityAPI BlockFinalityAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    RoundStateResponse
	}{
		{
			name: "GetJustification_Error",
			fields: fields{
				mockBlockAPI,
				mockBlockFinalityAPI,
			},
			args: args{
				req: &EmptyRequest{},
			},
			exp: RoundStateResponse{
				SetID: 0x0,
				Best: RoundState{
					Round:           0x2,
					TotalWeight:     0x9,
					ThresholdWeight: 0x6,
					Prevotes: Votes{
						CurrentWeight: 0x4,
						Missing: []string{
							"5Ck2miBfCe1JQ4cY3NDsXyBaD6EcsgiVmEFTWwqNSs25XDEq",
							"5E2BmpVFzYGd386XRCZ76cDePMB3sfbZp5ZKGUsrG1m6gomN",
							"5CGR8FbjxeV31JKaUUuVUgasW79k8xFGdoh8WG5MokEc78qj",
							"5E9ZP1w5qat63KrWEJLkh7aDr2fPTbu3UhetAjxeyBojKHYH",
							"5Cjb197EXcHehjxuyKUCF3wJm86owKiuKCzF18DcMhbgMhPX",
						},
					},
					Precommits: Votes{
						CurrentWeight: 0x2,
						Missing: []string{
							"5DbKjhNLpqX3zqZdNBc9BGb4fHU1cRBaDhJUskrvkwfraDi6",
							"5ECTwv6cZ5nJQPk6tWfaTrEk8YH2L7X1VT4EL5Tx2ikfFwb7",
							"5Ck2miBfCe1JQ4cY3NDsXyBaD6EcsgiVmEFTWwqNSs25XDEq",
							"5E2BmpVFzYGd386XRCZ76cDePMB3sfbZp5ZKGUsrG1m6gomN",
							"5CGR8FbjxeV31JKaUUuVUgasW79k8xFGdoh8WG5MokEc78qj",
							"5E9ZP1w5qat63KrWEJLkh7aDr2fPTbu3UhetAjxeyBojKHYH",
							"5Cjb197EXcHehjxuyKUCF3wJm86owKiuKCzF18DcMhbgMhPX",
						},
					},
				},
				Background: []RoundState{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := &GrandpaModule{
				blockAPI:         tt.fields.blockAPI,
				blockFinalityAPI: tt.fields.blockFinalityAPI,
			}
			res := RoundStateResponse{}
			err := gm.RoundState(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}
