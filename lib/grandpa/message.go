package grandpa

import (
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// GetVoteOutChannel returns a read-only VoteMessage channel
func (s *Service) GetVoteOutChannel() <-chan *VoteMessage {
	return s.out
}

// GetVoteInChannel returns a write-only VoteMessage channel
func (s *Service) GetVoteInChannel() chan<- *VoteMessage {
	return s.in
}

// GetFinalizedChannel returns a read-only FinalizationMessage channel
func (s *Service) GetFinalizedChannel() <-chan *FinalizationMessage {
	return s.finalized
}

// DecodeMessage decodes a network-level consensus message into a GRANDPA VoteMessage or FinalizationMessage
func (s *Service) DecodeMessage(msg *network.ConsensusMessage) (interface{}, error) {
	m, err := scale.Decode(msg.Data, new(VoteMessage))
	if err != nil {
		// try FinalizatioNmessage
		m, err = scale.Decode(msg.Data, new(FinalizationMessage))
		if err != nil {
			return nil, err
		}
	}

	return m, nil
}

// FullVote represents a vote with additional information about the state
// this is encoded and signed and the signature is included in SignedMessage
type FullVote struct {
	stage subround
	vote  *Vote
	round uint64
	setID uint64
}

// VoteMessage represents a network-level vote message
// https://github.com/paritytech/substrate/blob/master/client/finality-grandpa/src/communication/gossip.rs#L336
type VoteMessage struct {
	setID   uint64
	round   uint64
	stage   subround // 0 for pre-vote, 1 for pre-commit
	message *SignedMessage
}

// SignedMessage represents a block hash and number signed by an authority
// https://github.com/paritytech/substrate/blob/master/client/finality-grandpa/src/lib.rs#L146
type SignedMessage struct {
	hash        common.Hash
	number      uint64
	signature   [64]byte // ed25519.SignatureLength
	authorityID ed25519.PublicKeyBytes
}

// Justification represents a justification for a finalized block
//nolint:structcheck
type Justification struct {
	vote      *Vote    //nolint:unused
	signature []byte   //nolint:unused
	pubkey    [32]byte //nolint:unused
}

// FinalizationMessage represents a network finalization message
type FinalizationMessage struct {
	round         uint64
	vote          *Vote
	justification []*Justification //nolint:unused
}

// ToConsensusMessage converts the VoteMessage into a network-level consensus message
func (v *VoteMessage) ToConsensusMessage() (*network.ConsensusMessage, error) {
	enc, err := scale.Encode(v)
	if err != nil {
		return nil, err
	}

	return &network.ConsensusMessage{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              enc,
	}, nil
}

// ToConsensusMessage converts the FinalizationMessage into a network-level consensus message
func (f *FinalizationMessage) ToConsensusMessage() (*network.ConsensusMessage, error) {
	enc, err := scale.Encode(f)
	if err != nil {
		return nil, err
	}

	return &network.ConsensusMessage{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              enc,
	}, nil
}

func (s *Service) newFinalizationMessage(header *types.Header) (*FinalizationMessage, error) {
	return &FinalizationMessage{
		round: s.state.round,
		vote:  NewVoteFromHeader(header),
	}, nil
}
