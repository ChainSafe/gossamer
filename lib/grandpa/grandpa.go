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

// getPossiblePreVotedBlocks returns all blocks with total votes >= 2/3 the total number of voters
// this should be 1 exactly block, except in the case where exactly 1/3 voters equivocate.
// in that case, there may be 2 blocks returned.
// if there are > 1/3 byzantine nodes, then there may be 0, or more than 2 blocks returned.
// that would be very bad.
func (s *Service) getPossiblePreVotedBlocks() ([]Vote, error) {
	votes := s.getDirectVotes()
	blocks := []Vote{}

	for v, _ := range votes {
		total, err := s.getTotalVotesForBlock(v.hash)
		if err != nil {
			return nil, err
		}

		if total >= uint64(2*len(s.state.voters)/3) {
			blocks = append(blocks, v)
		}
	}

	return blocks, nil
}

// getPreVotedBlock returns the current pre-voted block B.
// the pre-voted block is the block with the highest block number in the set of all the blocks with
// total votes >= 2/3 the total number of voters, where the total votes is determined by getTotalVotesForBlock.
// note that by voting for a block, all of its predecessor blocks are automatically voted for.
// thus, if there are two blocks both with 2/3 total votes, and the same block number, this function
// returns their first common predecessor.
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
		return blocks[0], nil
	}

	// if there are two, find the greatest common predecessor and return it
	highest := Vote{
		number: uint64(0),
	}

	return highest, nil
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

	msg, err := scale.Encode(&FullVote{
		stage: m.stage,
		vote:  NewVote(m.message.hash, m.message.number),
		round: m.round,
		setID: m.setID,
	})
	if err != nil {
		return nil, err
	}
	ok, err := pk.Verify(msg, m.message.signature[:])
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, ErrInvalidSignature
	}

	// check that setIDs match
	if m.setID != s.state.setID {
		return nil, ErrSetIDMismatch
	}

	// check for equivocation ie. votes for blocks that do not reside on the same branch of the blocktree
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
