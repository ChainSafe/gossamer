// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/client/keystore"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa/app"
	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "primitives/consensus/grandpa"))

// AuthorityID is the identity of a Grandpa authority.
type AuthorityID = app.Public

// NewAuthorityID is constructor for AuthorityID
func NewAuthorityID(data []byte) (AuthorityID, error) {
	return app.NewPublic(data)
}

// AuthoritySignature is the signature for a Grandpa authority.
type AuthoritySignature = app.Signature

// GrandpaEngineID is the ConsensusEngineID of GRANDPA.
var GrandpaEngineID = runtime.ConsensusEngineID{'F', 'R', 'N', 'K'}

// AuthorityWeight is the weight of an authority.
type AuthorityWeight uint64

// AuthorityIndex is the index of an authority.
type AuthorityIndex uint64

// SetID is the monotonic identifier of a GRANDPA set of authorities.
type SetID uint64

// RoundNumber is the round indicator.
type RoundNumber uint64

// AuthorityIDWeight is struct containing AuthorityID and AuthorityWeight
type AuthorityIDWeight struct {
	AuthorityID AuthorityID // define attribute so SCALE doesn't call embedded AuthorityID.MarshalSCALE only
	AuthorityWeight
}

// AuthorityList is a list of Grandpa authorities with associated weights.
type AuthorityList []AuthorityIDWeight

// / A GRANDPA message for a substrate chain.
type Message[H, N any] grandpa.Message[H, N]

// SignedMessage is a signed message.
type SignedMessage[H, N any] grandpa.SignedMessage[H, N, AuthoritySignature, AuthorityID]

// / A primary propose message for this chain's block type.
type PrimaryPropose[H, N any] grandpa.PrimaryPropose[H, N]

// / A prevote message for this chain's block type.
type Prevote[H, N any] grandpa.Prevote[H, N]

// / A precommit message for this chain's block type.
type Precommit[H, N any] grandpa.Precommit[H, N]

// / A catch up message for this chain's block type.
type CatchUp[H, N any] grandpa.CatchUp[H, N, AuthoritySignature, AuthorityID]

// Commit is a commit message for this chain's block type.
type Commit[H, N any] grandpa.Commit[H, N, AuthoritySignature, AuthorityID]

// / A compact commit message for this chain's block type.
type CompactCommit[H, N any] grandpa.CompactCommit[H, N, AuthoritySignature, AuthorityID]

// ScheduledChange is a scheduled authority change.
type ScheduledChange[N runtime.Number] struct {
	NextAuthorities AuthorityList
	Delay           N
}

// GrandpaJustification is A GRANDPA justification for block finality, it includes a commit message and an ancestry
// proof including all headers routing all precommit target blocks to the commit target block. Due to the current
// voting strategy the precommit targets should be the same as the commit target, since honest voters don't vote past
// authority set change blocks.
//
// This is meant to be stored in the db and passed around the network to other nodes, and are used by syncing nodes to
// prove authority set handoffs.
type GrandpaJustification[Ordered runtime.Hash, N runtime.Number] struct {
	Round          uint64
	Commit         Commit[Ordered, N]
	VoteAncestries []runtime.Header[N, Ordered]
}

// CheckMessageSignature will check a message signature by encoding the message as a localised payload and verifying
// the provided signature using the expected authority id.
func CheckMessageSignature[H comparable, N constraints.Unsigned](
	message grandpa.Message[H, N],
	id AuthorityID,
	signature AuthoritySignature,
	round RoundNumber,
	setID SetID) bool {

	buf := NewLocalizedPayload(round, setID, message)
	valid := id.Verify(signature, buf)

	if !valid {
		logger.Debugf("Bad signature on message from %v", id)
	}
	return valid
}

// LocalizedPayload will encode round message localised to a given round and set id.
func NewLocalizedPayload[H comparable, N constraints.Unsigned](
	round RoundNumber,
	setID SetID,
	message grandpa.Message[H, N],
) []byte {
	return scale.MustMarshal(struct {
		Message grandpa.MessageVDT[H, N]
		RoundNumber
		SetID
	}{grandpa.NewMessageVDT(message), round, setID})
}

// / Localizes the message to the given set and round and signs the payload.
func SignMessage[H comparable, N constraints.Unsigned](
	keystore keystore.KeyStore,
	message grandpa.Message[H, N],
	public AuthorityID,
	round RoundNumber,
	setID SetID,
) *grandpa.SignedMessage[H, N, AuthoritySignature, AuthorityID] {
	encoded := NewLocalizedPayload(round, setID, message)
	signature, err := keystore.Ed25519Sign(crypto.GRANDPA, public[:], encoded)
	if err != nil {
		return nil
	}
	return &grandpa.SignedMessage[H, N, AuthoritySignature, AuthorityID]{
		Message:   message,
		Signature: AuthoritySignature(*signature),
		ID:        public,
	}
}
