// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/stretchr/testify/require"

	rpcmocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
)

var kr, _ = keystore.NewEd25519Keyring()

func TestGrandpaProveFinality(t *testing.T) {
	testStateService := newTestStateService(t)

	state.AddBlocksToState(t, testStateService.Block, 3, false)
	bestBlock, err := testStateService.Block.BestBlock()

	if err != nil {
		t.Errorf("Fail: bestblock failed")
	}

	gmSvc := NewGrandpaModule(testStateService.Block, nil)

	testStateService.Block.SetJustification(bestBlock.Header.ParentHash, make([]byte, 10))
	testStateService.Block.SetJustification(bestBlock.Header.Hash(), make([]byte, 11))

	var expectedResponse ProveFinalityResponse
	expectedResponse = append(expectedResponse, make([]byte, 10), make([]byte, 11))

	res := new(ProveFinalityResponse)
	err = gmSvc.ProveFinality(nil, &ProveFinalityRequest{
		blockHashStart: bestBlock.Header.ParentHash,
		blockHashEnd:   bestBlock.Header.Hash(),
	}, res)

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(*res, expectedResponse) {
		t.Errorf("Fail: expected: %+v got: %+v\n", res, &expectedResponse)
	}
}

func TestRoundState(t *testing.T) {
	var voters grandpa.Voters

	for _, k := range kr.Keys {
		voters = append(voters, types.GrandpaVoter{
			Key: *k.Public().(*ed25519.PublicKey),
			ID:  1,
		})
	}

	grandpamock := new(rpcmocks.BlockFinalityAPI)
	grandpamock.On("GetVoters").Return(voters)
	grandpamock.On("GetSetID").Return(uint64(0))
	grandpamock.On("GetRound").Return(uint64(2))

	grandpamock.On("PreVotes").Return([]ed25519.PublicKeyBytes{
		kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Bob().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Charlie().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Dave().Public().(*ed25519.PublicKey).AsBytes(),
	})

	grandpamock.On("PreCommits").Return([]ed25519.PublicKeyBytes{
		kr.Alice().Public().(*ed25519.PublicKey).AsBytes(),
		kr.Bob().Public().(*ed25519.PublicKey).AsBytes(),
	})

	mod := NewGrandpaModule(nil, grandpamock)

	res := new(RoundStateResponse)
	err := mod.RoundState(nil, nil, res)

	require.NoError(t, err)

	// newTestVoters has actually 9 keys with weight of 1
	require.Equal(t, uint32(9), res.Best.TotalWeight)
	require.Equal(t, uint32(6), res.Best.ThresholdWeight)

	expectedMissingPrevotes := []string{
		string(kr.Eve().Public().Address()),
		string(kr.Ferdie().Public().Address()),
		string(kr.George().Public().Address()),
		string(kr.Heather().Public().Address()),
		string(kr.Ian().Public().Address()),
	}

	expectedMissingPrecommits := append([]string{
		string(kr.Charlie().Public().Address()),
		string(kr.Dave().Public().Address()),
	}, expectedMissingPrevotes...)

	require.Equal(t, expectedMissingPrevotes, res.Best.Prevotes.Missing)
	require.Equal(t, expectedMissingPrecommits, res.Best.Precommits.Missing)

	require.Equal(t, uint32(4), res.Best.Prevotes.CurrentWeight)
	require.Equal(t, uint32(2), res.Best.Precommits.CurrentWeight)
}
