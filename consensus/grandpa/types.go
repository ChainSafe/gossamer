//nolint:structcheck
package grandpa

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/crypto"
)

type Voter struct {
	key     crypto.Keypair //nolint:structcheck
	voterId uint64 //nolint:structcheck
}

type State struct {
	voters  []Voter //nolint:structcheck
	counter uint64 //nolint:structcheck
	round   uint64 //nolint:structcheck
}

type Vote struct {
	hash   common.Hash //nolint:structcheck
	number uint64 //nolint:structcheck
}

type VoteMessage struct {
	round   uint64 //nolint:structcheck
	counter uint64 //nolint:structcheck
	pubkey  [32]byte //nolint:structcheck // ed25519 public key
	stage   byte    //nolint:structcheck  // 0 for pre-vote, 1 for pre-commit
}

//nolint:structcheck
type Justification struct {
	vote      Vote //nolint:structcheck
	signature []byte //nolint:structcheck
	pubkey    [32]byte //nolint:structcheck
}

//nolint:structcheck
type FinalizationMessage struct {
	round         uint64 //nolint:structcheck
	vote          Vote //nolint:structcheck
	justification Justification //nolint:structcheck
}
