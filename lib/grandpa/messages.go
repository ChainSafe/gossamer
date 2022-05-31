// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/grandpa/models"
)

func (s *Service) newCatchUpResponse(round, setID uint64) (*models.CatchUpResponse, error) {
	header, err := s.blockState.GetFinalisedHeader(round, setID)
	if err != nil {
		return nil, err
	}

	pvs, err := s.grandpaState.GetPrevotes(round, setID)
	if err != nil {
		return nil, err
	}

	pcs, err := s.grandpaState.GetPrecommits(round, setID)
	if err != nil {
		return nil, err
	}

	return &models.CatchUpResponse{
		SetID:                  setID,
		Round:                  round,
		PreVoteJustification:   pvs,
		PreCommitJustification: pcs,
		Hash:                   header.Hash(),
		Number:                 uint32(header.Number),
	}, nil
}

func (s *Service) newCommitMessage(header *types.Header, round uint64) (*models.CommitMessage, error) {
	grandpaSignedVotes, err := s.grandpaState.GetPrecommits(round, s.state.SetID)
	if err != nil {
		return nil, err
	}

	precommits, authData := justificationToCompact(grandpaSignedVotes)
	return &models.CommitMessage{
		Round:      round,
		Vote:       *models.NewVoteFromHeader(header),
		Precommits: precommits,
		AuthData:   authData,
	}, nil
}
