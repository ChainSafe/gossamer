// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/stretchr/testify/mock"
)

func TestGrandpaModule_ProveFinality(t *testing.T) {
	testHash := common.NewHash([]byte{0x01, 0x02})
	testHashSlice := []common.Hash{testHash, testHash, testHash}

	mockBlockFinalityAPI := new(apimocks.BlockFinalityAPI)
	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(testHashSlice, nil)
	mockBlockAPI.On("HasJustification", mock.AnythingOfType("common.Hash")).Return(true, nil)
	mockBlockAPI.On("GetJustification", mock.AnythingOfType("common.Hash")).Return([]byte("test"), nil)

	mockBlockAPIHasJustErr := new(apimocks.BlockAPI)
	mockBlockAPIHasJustErr.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(testHashSlice, nil)
	mockBlockAPIHasJustErr.On("HasJustification", mock.AnythingOfType("common.Hash")).Return(false, nil)

	mockBlockAPIGetJustErr := new(apimocks.BlockAPI)
	mockBlockAPIGetJustErr.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(testHashSlice, nil)
	mockBlockAPIGetJustErr.On("HasJustification", mock.AnythingOfType("common.Hash")).Return(true, nil)
	mockBlockAPIGetJustErr.On("GetJustification", mock.AnythingOfType("common.Hash")).Return(nil, errors.New("GetJustification error"))

	mockBlockAPISubChainErr := new(apimocks.BlockAPI)
	mockBlockAPISubChainErr.On("SubChain", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("common.Hash")).Return(nil, errors.New("SubChain error"))

	grandpaModule := NewGrandpaModule(mockBlockAPISubChainErr, mockBlockFinalityAPI)
	var res ProveFinalityResponse
	type fields struct {
		blockAPI         BlockAPI
		blockFinalityAPI BlockFinalityAPI
	}
	type args struct {
		r   *http.Request
		req *ProveFinalityRequest
		res *ProveFinalityResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "SubChain Err",
			fields: fields{
				grandpaModule.blockAPI,
				grandpaModule.blockFinalityAPI,
			},
			args: args{
				req: &ProveFinalityRequest{
					blockHashStart: testHash,
					blockHashEnd:   testHash,
					authorityID:    uint64(21),
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "OK Case",
			fields: fields{
				mockBlockAPI,
				mockBlockFinalityAPI,
			},
			args: args{
				req: &ProveFinalityRequest{
					blockHashStart: testHash,
					blockHashEnd:   testHash,
					authorityID:    uint64(21),
				},
				res: &res,
			},
		},
		{
			name: "HasJustification Error",
			fields: fields{
				mockBlockAPIHasJustErr,
				mockBlockFinalityAPI,
			},
			args: args{
				req: &ProveFinalityRequest{
					blockHashStart: testHash,
					blockHashEnd:   testHash,
					authorityID:    uint64(21),
				},
				res: &res,
			},
		},
		{
			name: "GetJustification Error",
			fields: fields{
				mockBlockAPIGetJustErr,
				mockBlockFinalityAPI,
			},
			args: args{
				req: &ProveFinalityRequest{
					blockHashStart: testHash,
					blockHashEnd:   testHash,
					authorityID:    uint64(21),
				},
				res: &res,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := &GrandpaModule{
				blockAPI:         tt.fields.blockAPI,
				blockFinalityAPI: tt.fields.blockFinalityAPI,
			}
			if err := gm.ProveFinality(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("ProveFinality() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGrandpaModule_RoundState(t *testing.T) {
	var kr, _ = keystore.NewEd25519Keyring()
	var voters grandpa.Voters

	for _, k := range kr.Keys {
		voters = append(voters, types.GrandpaVoter{
			Key: *k.Public().(*ed25519.PublicKey),
			ID:  1,
		})
	}

	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockFinalityAPI := new(apimocks.BlockFinalityAPI)
	mockBlockFinalityAPI.On("GetVoters").Return(voters)
	mockBlockFinalityAPI.On("GetSetID").Return(uint64(0))
	mockBlockFinalityAPI.On("GetRound").Return(uint64(2))
	mockBlockFinalityAPI.On("PreVotes").Return([]ed25519.PublicKeyBytes{
		kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Bob().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Charlie().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Dave().Public().(*ed25519.PublicKey).AsBytes(),
	})
	mockBlockFinalityAPI.On("PreCommits").Return([]ed25519.PublicKeyBytes{
		kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Bob().Public().(*ed25519.PublicKey).AsBytes(),
	})

	var res RoundStateResponse
	type fields struct {
		blockAPI         BlockAPI
		blockFinalityAPI BlockFinalityAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
		res *RoundStateResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "GetJustification Error",
			fields: fields{
				mockBlockAPI,
				mockBlockFinalityAPI,
			},
			args: args{
				req: &EmptyRequest{},
				res: &res,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := &GrandpaModule{
				blockAPI:         tt.fields.blockAPI,
				blockFinalityAPI: tt.fields.blockFinalityAPI,
			}
			if err := gm.RoundState(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("RoundState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
