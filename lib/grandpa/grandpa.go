package grandpa

import (
	"bytes"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// Service represents the current state of the grandpa protocol
type Service struct {
	state         *State // current state
	blockState    BlockState
	subround      subround                           // current sub-round
	votes         map[ed25519.PublicKeyBytes]*Vote   // votes for next state
	equivocations map[ed25519.PublicKeyBytes][]*Vote // equivocatory votes for this stage
	head          common.Hash                        // most recently finalized block hash
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
		head:          head.Hash(),
	}, nil
}

// getDirectVotes returns a map of Votes to direct vote counts
func (s *Service) getDirectVotes() map[Vote]uint64 {
	votes := make(map[Vote]uint64)

	for _, v := range s.votes {
		votes[*v]++
	}

	return votes
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

		if total >= uint64(2*len(s.state.voters)/3) {
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
	for v := range votes {
		blocks, err = s.getPossiblePreVotedPredecessors(votes, v.hash, blocks)
		if err != nil {
			return nil, err
		}
	}

	return blocks, nil
}

// getPossiblePreVotedPredecessors recursively searches for predecessors with >=2/3 votes
// it returns a map of block hash -> number, such that the blocks in the map have >=2/3 votes
func (s *Service) getPossiblePreVotedPredecessors(votes map[Vote]uint64, curr common.Hash, prevoted map[common.Hash]uint64) (map[common.Hash]uint64, error) {
	for v := range votes {
		if v.hash == curr {
			continue
		}

		// find common predecessor, check if votes for it is >=2/3 or not
		pred, err := s.blockState.HighestCommonPredecessor(v.hash, curr)
		if err != nil {
			return nil, err
		}

		total, err := s.getTotalVotesForBlock(pred)
		if err != nil {
			return nil, err
		}

		if total >= uint64(2*len(s.state.voters)/3) {
			h, err := s.blockState.GetHeader(pred)
			if err != nil {
				return nil, err
			}

			prevoted[pred] = uint64(h.Number.Int64())
		} else {
			prevoted, err = s.getPossiblePreVotedPredecessors(votes, pred, prevoted)
		}
	}

	return prevoted, nil
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
		for v := range blocks {
			return Vote{
				hash: v,
			}, nil
		}
	}

	// if there are multiple, find the one with the highest number and return it

	return Vote{}, nil
}

// CreateVoteMessage returns a signed VoteMessage given a header
func (s *Service) CreateVoteMessage(header *types.Header, kp crypto.Keypair) (*VoteMessage, error) {
	vote := NewVoteFromHeader(header)

	msg, err := scale.Encode(&FullVote{
		stage: s.subround,
		vote:  vote,
		round: s.state.round,
		setID: s.state.setID,
	})
	if err != nil {
		return nil, err
	}

	sig, err := kp.Sign(msg)
	if err != nil {
		return nil, err
	}

	sm := &SignedMessage{
		hash:        vote.hash,
		number:      vote.number,
		signature:   ed25519.NewSignatureBytes(sig),
		authorityID: kp.Public().(*ed25519.PublicKey).AsBytes(),
	}

	return &VoteMessage{
		setID:   s.state.setID,
		round:   s.state.round,
		stage:   s.subround,
		message: sm,
	}, nil
}

// ValidateMessage validates a VoteMessage and adds it to the current votes
// it returns the resulting vote if validated, error otherwise
func (s *Service) ValidateMessage(m *VoteMessage) (*Vote, error) {
	// check for message signature
	pk, err := ed25519.NewPublicKey(m.message.authorityID[:])
	if err != nil {
		return nil, err
	}

	err = validateMessageSignature(pk, m)
	if err != nil {
		return nil, err
	}

	// check that setIDs match
	if m.setID != s.state.setID {
		return nil, ErrSetIDMismatch
	}

	// check for equivocation ie. multiple votes within one subround
	voter, err := s.state.pubkeyToVoter(pk)
	if err != nil {
		return nil, err
	}

	vote := NewVote(m.message.hash, m.message.number)

	equivocated := s.checkForEquivocation(voter, vote)
	if equivocated {
		return nil, ErrEquivocation
	}

	err = s.validateVote(vote)
	if err != nil {
		return nil, err
	}

	s.votes[pk.AsBytes()] = vote

	return vote, nil
}

func validateMessageSignature(pk *ed25519.PublicKey, m *VoteMessage) error {
	msg, err := scale.Encode(&FullVote{
		stage: m.stage,
		vote:  NewVote(m.message.hash, m.message.number),
		round: m.round,
		setID: m.setID,
	})
	if err != nil {
		return err
	}
	ok, err := pk.Verify(msg, m.message.signature[:])
	if err != nil {
		return err
	}

	if !ok {
		return ErrInvalidSignature
	}

	return nil
}

// checkForEquivocation checks if the vote is an equivocatory vote.
// it returns true if so, false otherwise.
// additionally, if the vote is equivocatory, it updates the service's votes and equivocations.
func (s *Service) checkForEquivocation(voter *Voter, vote *Vote) bool {
	v := voter.key.AsBytes()

	if s.equivocations[v] != nil {
		// if the voter has already equivocated, every vote in that round is an equivocatory vote
		s.equivocations[v] = append(s.equivocations[v], vote)
		return true
	}

	if s.votes[v] != nil {
		// the voter has already voter, all their votes are now equivocatory
		prev := s.votes[v]
		s.equivocations[v] = []*Vote{prev, vote}
		delete(s.votes, v)
		return true
	}

	return false
}

// validateVote checks if the block that is being voted for exists, and that it is a descendant of a
// previously finalized block.
func (s *Service) validateVote(v *Vote) error {
	// check if v.hash corresponds to a valid block
	has, err := s.blockState.HasHeader(v.hash)
	if err != nil {
		return err
	}

	if !has {
		return ErrBlockDoesNotExist
	}

	// check if the block is an eventual descendant of a previously finalized block
	isDescendant, err := s.blockState.IsDescendantOf(s.head, v.hash)
	if err != nil {
		return err
	}

	if !isDescendant {
		return ErrDescendantNotFound
	}

	return nil
}

// NewState returns a new GRANDPA state
func NewState(voters []*Voter, setID, round uint64) *State {
	return &State{
		voters: voters,
		setID:  setID,
		round:  round,
	}
}

// pubkeyToVoter returns a Voter given a public key
func (s *State) pubkeyToVoter(pk *ed25519.PublicKey) (*Voter, error) {
	id := uint64(2^64) - 1

	for i, v := range s.voters {
		if bytes.Equal(pk.Encode(), v.key.Encode()) {
			id = uint64(i)
		}
	}

	if id == (2^64)-1 {
		return nil, ErrVoterNotFound
	}

	return &Voter{
		key: pk,
		id:  id,
	}, nil
}
