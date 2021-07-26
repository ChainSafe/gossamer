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

	"github.com/stretchr/testify/require"
)

func TestMessageHandler_VerifyPreVoteJustification(t *testing.T) {
	gs, _ := newTestService(t)
	h := newCatchUp(true, gs, newTestNetwork(t), nil)

	just := buildTestJustification(t, int(gs.state.threshold()), 1, gs.state.setID, kr, prevote)
	msg := &catchUpResponse{
		Round:                1,
		SetID:                gs.state.setID,
		PreVoteJustification: just,
	}

	prevote, err := h.verifyPreVoteJustification(msg)
	require.NoError(t, err)
	require.Equal(t, testHash, prevote)
}

func TestMessageHandler_VerifyPreCommitJustification(t *testing.T) {
	gs, _ := newTestService(t)
	h := newCatchUp(true, gs, newTestNetwork(t), nil)

	round := uint64(1)
	just := buildTestJustification(t, int(gs.state.threshold()), round, gs.state.setID, kr, precommit)
	msg := &catchUpResponse{
		Round:                  round,
		SetID:                  gs.state.setID,
		PreCommitJustification: just,
		Hash:                   testHash,
		Number:                 uint32(round),
	}

	err := h.verifyPreCommitJustification(msg)
	require.NoError(t, err)
}
