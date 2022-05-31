// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import "github.com/ChainSafe/gossamer/lib/grandpa/models"

func compactToJustification(preCommitVotes []models.Vote, auths []models.AuthData) (
	signedVotes []models.SignedVote, err error) {
	if len(preCommitVotes) != len(auths) {
		return nil, errVoteToSignatureMismatch
	}

	signedVotes = make([]models.SignedVote, len(preCommitVotes))
	for i, v := range preCommitVotes {
		signedVotes[i] = models.SignedVote{
			Vote:        v,
			Signature:   auths[i].Signature,
			AuthorityID: auths[i].AuthorityID,
		}
	}

	return signedVotes, nil
}

func justificationToCompact(signedVotes []models.SignedVote) (
	preCommitVotes []models.Vote, auths []models.AuthData) {
	preCommitVotes = make([]models.Vote, len(signedVotes))
	auths = make([]models.AuthData, len(signedVotes))

	for i, j := range signedVotes {
		preCommitVotes[i] = j.Vote
		auths[i] = models.AuthData{
			Signature:   j.Signature,
			AuthorityID: j.AuthorityID,
		}
	}

	return preCommitVotes, auths
}
