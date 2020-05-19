package grandpa

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// Service represents the current state of the grandpa protocol
type Service struct {
	state         *state // current state
	blockState    BlockState
	subround      subround          // current sub-round
	votes         map[Voter]*Vote   // votes for next state
	equivocations map[Voter][]*Vote // equivocatory votes for this stage
	head          common.Hash       // most recently finalized block hash
}

// NewService returns a new GRANDPA Service instance.
// TODO: determine GRANDPA initialization and entrypoint, as well as what needs to be exported.
func NewService(blockState BlockState, voters []*Voter) *Service {
	return &Grandpa{
		state:      newState(voters, 0, 0),
		blockState: blockState,
	}
}

// CreateVoteMessage returns a signed VoteMessage given a header
func (s *Service) CreateVoteMessage(header *types.Header, kp *crypto.Keypair) *VoteMessage {
	return &VoteMessage{}
}

func (s *Service) ValidateMessage(m *VoteMessage) (*Vote, error) {
	// check for message signature
	pk, err := ed25519.NewPublicKey(m.authorityID)
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
	voter := s.state.pubkeyToVoter(pk)
	vote := NewVote(m.message.hash, m.message.number)

	return vote, nil
}

// checkForEquivocation checks if the vote is an equivocatory vote.
// it returns true if so, false otherwise.
func (s *Service) checkForEquivocation(voter *Voter, vote *vote) (bool, error) {
	if s.equivocations[voter] != nil {
		// if the voter has already equivocated, every vote in that round is an equivocatory vote
		s.equivocations[voter] = append(s.equivocations[voter], vote)
		return true, nil
	}

	if s.votes[voter] != nil {
		// the voter has already voted, check if they are voting for a block on the same branch
		prev := s.votes[voter]

		// check if block in current vote is descendent of block in previous vote
		_, err = s.blockState.SubChain(prev.hash, vote.hash)
		if err == blocktree.ErrDescendantNotFound {

			// check if block in previous vote is descendent of block in current vote
			_, err = s.blockState.SubChain(vote.hash, prev.hash)
			if err == blocktree.ErrDescendantNotFound {

				// block producer equivocated
				s.equivocations[voter] = []*Vote{prev, vote}
				s.votes[voter] = nil
				return true, nil

			} else if err != nil {
				return false, err
			}

		} else if err != nil {
			return false, err
		}
	}

	return false, nil
}

// NewState returns a new GRANDPA state
func NewState(voters []*voter, setID, round uint64) *State {
	return &state{
		voters:  voters,
		counter: counter,
		round:   round,
	}
}

func (s *State) pubkeyToVoter(pk ed25519.PublicKey) *Voter {
	return &Voter{
		key: pk,
		// TODO: get voterID by searching through :grandpa_authorities
	}
}

func (s *State) validateVote(v *Vote) error {
	// check if v.hash corresponds to a valid block
	has, err := s.blockState.HasHeader(v.hash)
	if err != nil {
		return err
	}

	if !has {
		return ErrBlockDoesNotExist
	}

	// check if the block is an eventual descendant of a previously finalized block
	_, err = s.blockState.SubChain(s.head, v.hash)
	if err != nil {
		return err
	}
}
