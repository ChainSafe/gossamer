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
	"bytes"
	//"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
)

// Service represents the current state of the grandpa protocol
type Service struct {
	state         *State // current state
	blockState    BlockState
	keypair       *ed25519.Keypair                   // our keypair
	subround      subround                           // current sub-round
	votes         map[ed25519.PublicKeyBytes]*Vote   // votes for next state
	equivocations map[ed25519.PublicKeyBytes][]*Vote // equivocatory votes for this stage
	head          common.Hash                        // most recently finalized block hash
	primaryVotes  map[uint64]*Vote                   // map of round number to votes from primary, can clear every round
}

// NewService returns a new GRANDPA Service instance.
// TODO: determine what needs to be exported.
func NewService(blockState BlockState, voters []*Voter) (*Service, error) {
	head, err := blockState.GetFinalizedHead()
	if err != nil {
		return nil, err
	}

	return &Service{
		state:         NewState(voters, 0, 0),
		blockState:    blockState,
		subround:      prevote,
		votes:         make(map[ed25519.PublicKeyBytes]*Vote),
		equivocations: make(map[ed25519.PublicKeyBytes][]*Vote),
		head:          head,
	}, nil
}

// initiate increments the round number and sets the finalized block hash in the database
func (s *Service) initiate() error {
	// TODO: check runtime digests for authority changes
	// needs a runtime update
	s.state.round += 1

	// set finalized head
	return s.blockState.SetFinalizedHead(s.head)
}

func (s *Service) beginPreVote() error {
	// save start time
	//start := time.Now().Unix()

	// derive primary
	primary := s.derivePrimary()

	// if primary, broadcast the best final candidate from the previous round
	if bytes.Equal(primary.key.Encode(), s.keypair.Public().Encode()) {
		// TODO: broadcast best final candidate
	}

	// TODO: receive messages until current time runs out or round is completable

	// determine what block we will vote for
	var vote *Vote
	// if we receive a vote message from the primary with a block that's greater than or equal to the current pre-voted block,
	// and greater than the best final candidate from the last round, we choose that
	if s.primaryVotes[s.state.round] != nil && s.primaryVotes[s.state.round].number > 0 {
		vote = s.primaryVotes[s.state.round]
	}

	head, err := s.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) endRound() {

}

func (s *Service) derivePrimary() *Voter {
	return s.state.voters[s.state.round%uint64(len(s.state.voters))]
}

// isCompletable returns true if the round is completable, false otherwise
func (s *Service) isCompletable() (bool, error) {
	votes := s.getVotes()
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
		c, err := s.getTotalVotesForBlock(v.hash)
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
	blocks, err := s.getPossiblePreVotedBlocks()
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

// getPossiblePreVotedBlocks returns blocks with total votes >=2/3 the total number of voters in the map of block hash to block number.
// if there are no blocks that have >=2/3 direct votes, this function will find predecessors of those blocks that do have >=2/3 votes.
// note that by voting for a block, all of its predecessor blocks are automatically voted for.
// thus, if there are no blocks with >=2/3 total votes, but the sum of votes for blocks A and B is >=2/3, then this function returns
// the first common predecessor of A and B.
func (s *Service) getPossiblePreVotedBlocks() (map[common.Hash]uint64, error) {
	// get blocks that were directly voted for
	votes := s.getDirectVotes()
	blocks := make(map[common.Hash]uint64)

	// check if any of them have >=2/3 votes
	for v := range votes {
		total, err := s.getTotalVotesForBlock(v.hash)
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
	va := s.getVotes()

	for v := range votes {
		blocks, err = s.getPossiblePreVotedPredecessors(va, v.hash, blocks)
		if err != nil {
			return nil, err
		}
	}

	return blocks, nil
}

// getPossiblePreVotedPredecessors recursively searches for predecessors with >=2/3 votes
// it returns a map of block hash -> number, such that the blocks in the map have >=2/3 votes
func (s *Service) getPossiblePreVotedPredecessors(votes []Vote, curr common.Hash, prevoted map[common.Hash]uint64) (map[common.Hash]uint64, error) {
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

		total, err := s.getTotalVotesForBlock(pred)
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
			prevoted, err = s.getPossiblePreVotedPredecessors(votes, pred, prevoted)
			if err != nil {
				return nil, err
			}
		}
	}

	return prevoted, nil
}

// getTotalVotesForBlock returns the total number of observed votes for a block B, which is equal
// to the direct votes for B and B's descendants plus the total number of equivocating voters
func (s *Service) getTotalVotesForBlock(hash common.Hash) (uint64, error) {
	// observed votes for block
	dv, err := s.getVotesForBlock(hash)
	if err != nil {
		return 0, err
	}

	// equivocatory votes
	ev := len(s.equivocations)

	return dv + uint64(ev), nil
}

// getVotesForBlock returns the number of observed votes for a block B.
// The set of all observed votes by v in the sub-round stage of round r for block B is
// equal to all of the observed direct votes cast for block B and all of the B's descendants
func (s *Service) getVotesForBlock(hash common.Hash) (uint64, error) {
	votes := s.getDirectVotes()

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
func (s *Service) getDirectVotes() map[Vote]uint64 {
	votes := make(map[Vote]uint64)

	for _, v := range s.votes {
		votes[*v]++
	}

	return votes
}

// getVotes returns all the current votes as an array
func (s *Service) getVotes() []Vote {
	votes := s.getDirectVotes()
	va := make([]Vote, len(votes))
	i := 0

	for v := range votes {
		va[i] = v
		i++
	}

	return va
}
