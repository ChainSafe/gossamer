// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package grandpa

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"

	"github.com/stretchr/testify/require"
)

// type votes struct {
// 	prevotes        map[ed25519.PublicKeyBytes]*Vote   // pre-votes for next state
// 	precommits      map[ed25519.PublicKeyBytes]*Vote   // pre-commits for next state
// 	pvEquivocations map[ed25519.PublicKeyBytes][]*Vote // equivocatory votes for current pre-vote stage
// 	pcEquivocations map[ed25519.PublicKeyBytes][]*Vote // equivocatory votes for current pre-commit stage
// }

// func newVotes() *votes {
// 	return &votes{
// 		prevotes:           make(map[ed25519.PublicKeyBytes]*Vote),
// 		precommits:         make(map[ed25519.PublicKeyBytes]*Vote),
// 		pvEquivocations:    make(map[ed25519.PublicKeyBytes][]*Vote),
// 		pcEquivocations:    make(map[ed25519.PublicKeyBytes][]*Vote),
// 	}
// }

func setupGrandpa(t *testing.T, kp *ed25519.Keypair) *Service {
	st := newTestState(t)
	voters := newTestVoters(t)

	cfg := &Config{
		BlockState: st.Block,
		Voters:     voters,
		Keypair:    kp,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)

	return gs
}

func TestGrandpa_BaseCase(t *testing.T) {
	// this is a base test cases that asserts that all validators finalize the same block if they all see the
	// same pre-votes and pre-commits, even if their chains are different
	numVoters := 9

	gss := make([]*Service, numVoters)
	prevotes := make(map[ed25519.PublicKeyBytes]*Vote)
	precommits := make(map[ed25519.PublicKeyBytes]*Vote)
	var err error

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	for i, gs := range gss {
		gs = setupGrandpa(t, kr.Keys[i])
		gss[i] = gs
		state.AddBlocksToState(t, gs.blockState.(*state.BlockState), 15)
		prevotes[gs.publicKeyBytes()], err = gs.determinePreVote()
		require.NoError(t, err)
	}

	for _, gs := range gss {
		gs.prevotes = prevotes
	}

	for _, gs := range gss {
		precommits[gs.publicKeyBytes()], err = gs.determinePreCommit()
		require.NoError(t, err)
		err = gs.finalize()
		require.NoError(t, err)
	}

	finalized := gss[0].head.Hash()
	for _, gs := range gss {
		require.Equal(t, finalized, gs.head.Hash())
	}
}
