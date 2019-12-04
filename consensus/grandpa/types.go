package grandpa

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/crypto"
)

//nolint:structcheck
type Voter struct {
	key     crypto.Keypair
	voterId uint64
}

//nolint:structcheck
type State struct {
	voters  []Voter
	counter uint64
	round   uint64
}

//nolint:structcheck
type Vote struct {
	hash   common.Hash
	number uint64
}

//nolint:structcheck
type VoteMessage struct {
	round   uint64
	counter uint64
	pubkey  [32]byte // ed25519 public key
	stage   byte     // 0 for pre-vote, 1 for pre-commit
}

//nolint:structcheck
type Justification struct {
	vote      Vote
	signature []byte
	pubkey    [32]byte
}

//nolint:structcheck
type FinalizationMessage struct {
	round         uint64
	vote          Vote
	justification Justification
}
