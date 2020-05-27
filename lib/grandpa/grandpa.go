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
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
)

// Service represents the current state of the grandpa protocol
type Service struct {
	state              *State // current state
	blockState         BlockState
	keypair            *ed25519.Keypair                   // our keypair
	subround           subround                           // current sub-round
	prevotes           map[ed25519.PublicKeyBytes]*Vote   // pre-votes for next state
	precommits         map[ed25519.PublicKeyBytes]*Vote   // pre-commits for next state
	pvEquivocations    map[ed25519.PublicKeyBytes][]*Vote // equivocatory votes for current pre-vote stage
	pcEquivocations    map[ed25519.PublicKeyBytes][]*Vote // equivocatory votes for current pre-commit stage
	head               *types.Header                      // most recently finalized block hash
	primaryVotes       map[uint64]*Vote                   // map of round number to votes from primary, can clear every round
	bestFinalCandidate map[uint64]*Vote                   // map of round number to best final candidate
}

// NewService returns a new GRANDPA Service instance.
// TODO: determine what needs to be exported.
func NewService(blockState BlockState, voters []*Voter) (*Service, error) {
	head, err := blockState.GetFinalizedHead()
	if err != nil {
		return nil, err
	}

	return &Service{
		state:           NewState(voters, 0, 0),
		blockState:      blockState,
		subround:        prevote,
		prevotes:        make(map[ed25519.PublicKeyBytes]*Vote),
		precommits:      make(map[ed25519.PublicKeyBytes]*Vote),
		pvEquivocations: make(map[ed25519.PublicKeyBytes][]*Vote),
		pcEquivocations: make(map[ed25519.PublicKeyBytes][]*Vote),
		head:            head,
	}, nil
}

// derivePrimary returns the primary for the current round
func (s *Service) derivePrimary() *Voter {
	return s.state.voters[s.state.round%uint64(len(s.state.voters))]
}

// getBestFinalCandidate calculates the set of blocks that are less than or equal to the pre-voted block in height,
// with >= 2/3 pre-commit votes, then returns the block with the highest number from this set.
func (s *Service) getBestFinalCandidate() (*Vote, error) {
	prevoted, err := s.getPreVotedBlock()
	if err != nil {
		return nil, err
	}

	// get all blocks with >=2/3 pre-commits
	blocks, err := s.getPossibleSelectedBlocks(precommit)
	if err != nil {
		return nil, err
	}

	if len(blocks) == 0 {
		return nil, ErrNoBestFinalCandidate
	}

	// if there are blocks, check if it's number is <= prevoted block's number
	precommited := []*Vote{}

	for h, n := range blocks {
		if n <= prevoted.number {
			precommited = append(precommited, &Vote{
				hash:   h,
				number: n,
			})

			continue
		}

		// if the number is greater than that of the prevoted block, find ancestor block
		// that is at the same number as prevoted block
		p, err := s.findParentWithNumber(&Vote{
			hash:   h,
			number: n,
		}, prevoted.number)
		if err != nil {
			return nil, err
		}

		precommited = append(precommited, p)
	}

	// find block with highest number from remaining blocks

	return &Vote{}, nil
}

// findParentWithNumber returns a Vote for an ancestor with number n given an existing Vote
func (s *Service) findParentWithNumber(v *Vote, n uint64) (*Vote, error) {
	if v.number <= n {
		return v, nil
	}

	b, err := s.blockState.GetHeader(v.hash)
	if err != nil {
		return nil, err
	}

	// # of iterations
	l := int(v.number - n)

	for i := 0; i < l; i++ {
		p, err := s.blockState.GetHeader(b.ParentHash)
		if err != nil {
			return nil, err
		}

		b = p
	}

	return NewVoteFromHeader(b), nil
}

// isCompletable returns true if the round is completable, false otherwise
func (s *Service) isCompletable() (bool, error) {
	votes := s.getVotes(prevote)
	prevoted, err := s.getPreVotedBlock()
	if err != nil {
		return false, err
	}

	for _, v := range votes {
		// check if the current block is a descendant of prevoted block
		isDescendant, err := s.blockState.IsDescendantOf(prevoted.hash, v.hash)
		if err != nil {
			return false, err
		}

		if !isDescendant {
			continue
		}

		// if it's a descendant, check if has >=2/3 votes
		c, err := s.getTotalVotesForBlock(v.hash, prevote)
		if err != nil {
			return false, err
		}

		if c > s.state.threshold() {
			// round isn't completable
			return false, nil
		}
	}

	return true, nil
}

// getPreVotedBlock returns the current pre-voted block B.
// the pre-voted block is the block with the highest block number in the set of all the blocks with
// total votes >= 2/3 the total number of voters, where the total votes is determined by getTotalVotesForBlock.
func (s *Service) getPreVotedBlock() (Vote, error) {
	blocks, err := s.getPossibleSelectedBlocks(prevote)
	if err != nil {
		return Vote{}, err
	}

	if len(blocks) == 0 {
		return Vote{}, ErrNoPreVotedBlock
	}

	// if there is one block, return it
	if len(blocks) == 1 {
		for h, n := range blocks {
			return Vote{
				hash:   h,
				number: n,
			}, nil
		}
	}

	// if there are multiple, find the one with the highest number and return it
	highest := Vote{
		number: uint64(0),
	}
	for h, n := range blocks {
		if n > highest.number {
			highest = Vote{
				hash:   h,
				number: n,
			}
		}
	}

	return highest, nil
}

// getPossibleSelectedBlocks returns blocks with total votes >=2/3 the total number of voters in the map of block hash to block number.
// if there are no blocks that have >=2/3 direct votes, this function will find predecessors of those blocks that do have >=2/3 votes.
// note that by voting for a block, all of its predecessor blocks are automatically voted for.
// thus, if there are no blocks with >=2/3 total votes, but the sum of votes for blocks A and B is >=2/3, then this function returns
// the first common predecessor of A and B.
func (s *Service) getPossibleSelectedBlocks(stage subround) (map[common.Hash]uint64, error) {
	// get blocks that were directly voted for
	votes := s.getDirectVotes(stage)
	blocks := make(map[common.Hash]uint64)

	// check if any of them have >=2/3 votes
	for v := range votes {
		total, err := s.getTotalVotesForBlock(v.hash, stage)
		if err != nil {
			return nil, err
		}

		if total >= s.state.threshold() {
			blocks[v.hash] = v.number
		}
	}

	// since we want to select the block with the highest number that has >=2/3 votes,
	// we can return here since their predecessors won't have a higher number.
	if len(blocks) != 0 {
		return blocks, nil
	}

	// no block has >=2/3 direct votes, check for votes for predecessors recursively
	var err error
	va := s.getVotes(stage)

	for v := range votes {
		blocks, err = s.getPossibleSelectedPredecessors(va, v.hash, blocks, stage)
		if err != nil {
			return nil, err
		}
	}

	return blocks, nil
}

// getPossibleSelectedPredecessors recursively searches for predecessors with >=2/3 votes
// it returns a map of block hash -> number, such that the blocks in the map have >=2/3 votes
func (s *Service) getPossibleSelectedPredecessors(votes []Vote, curr common.Hash, prevoted map[common.Hash]uint64, stage subround) (map[common.Hash]uint64, error) {
	for _, v := range votes {
		if v.hash == curr {
			continue
		}

		// find common predecessor, check if votes for it is >=2/3 or not
		pred, err := s.blockState.HighestCommonAncestor(v.hash, curr)
		if err != nil {
			return nil, err
		}

		if pred == curr {
			return prevoted, nil
		}

		total, err := s.getTotalVotesForBlock(pred, stage)
		if err != nil {
			return nil, err
		}

		if total >= s.state.threshold() {
			var h *types.Header
			h, err = s.blockState.GetHeader(pred)
			if err != nil {
				return nil, err
			}

			prevoted[pred] = uint64(h.Number.Int64())
		} else {
			prevoted, err = s.getPossibleSelectedPredecessors(votes, pred, prevoted, stage)
			if err != nil {
				return nil, err
			}
		}
	}

	return prevoted, nil
}

// getTotalVotesForBlock returns the total number of observed votes for a block B, which is equal
// to the direct votes for B and B's descendants plus the total number of equivocating voters
func (s *Service) getTotalVotesForBlock(hash common.Hash, stage subround) (uint64, error) {
	// observed votes for block
	dv, err := s.getVotesForBlock(hash, stage)
	if err != nil {
		return 0, err
	}

	// equivocatory votes
	var ev int
	if stage == prevote {
		ev = len(s.pvEquivocations)
	} else {
		ev = len(s.pcEquivocations)
	}

	return dv + uint64(ev), nil
}

// getVotesForBlock returns the number of observed votes for a block B.
// The set of all observed votes by v in the sub-round stage of round r for block B is
// equal to all of the observed direct votes cast for block B and all of the B's descendants
func (s *Service) getVotesForBlock(hash common.Hash, stage subround) (uint64, error) {
	votes := s.getDirectVotes(stage)

	// B will be counted as in it's own subchain, so don't need to start with B's vote count
	votesForBlock := uint64(0)

	for v, c := range votes {

		// check if the current block is a descendant of B
		isDescendant, err := s.blockState.IsDescendantOf(hash, v.hash)
		if err != nil {
			return 0, err
		}

		if !isDescendant {
			continue
		}

		votesForBlock += c
	}

	return votesForBlock, nil
}

// getDirectVotes returns a map of Votes to direct vote counts
func (s *Service) getDirectVotes(stage subround) map[Vote]uint64 {
	votes := make(map[Vote]uint64)

	var src map[ed25519.PublicKeyBytes]*Vote
	if stage == prevote {
		src = s.prevotes
	} else {
		src = s.precommits
	}

	for _, v := range src {
		votes[*v]++
	}

	return votes
}

// getVotes returns all the current votes as an array
func (s *Service) getVotes(stage subround) []Vote {
	votes := s.getDirectVotes(stage)
	va := make([]Vote, len(votes))
	i := 0

	for v := range votes {
		va[i] = v
		i++
	}

	return va
}
