// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa/app"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

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
	AuthorityID
	AuthorityWeight
}

// AuthorityList is a list of Grandpa authorities with associated weights.
type AuthorityList []AuthorityIDWeight

// SignedMessage is a signed message.
type SignedMessage[H, N any] grandpa.SignedMessage[H, N, AuthoritySignature, AuthorityID]

// Commit is a commit message for this chain's block type.
type Commit[H, N any] grandpa.Commit[H, N, AuthoritySignature, AuthorityID]

// GrandpaJustification is A GRANDPA justification for block finality, it includes
// a commit message and an ancestry proof including all headers routing all
// precommit target blocks to the commit target block. Due to the current voting
// strategy the precommit targets should be the same as the commit target, since
// honest voters don't vote past authority set change blocks.
//
// This is meant to be stored in the db and passed around the network to other
// nodes, and are used by syncing nodes to prove authority set handoffs.
type GrandpaJustification[Ordered runtime.Hash, N runtime.Number] struct {
	Round          uint64
	Commit         Commit[Ordered, N]
	VoteAncestries []runtime.Header[N, Ordered]
}

// CheckMessageSignature will check a message signature by encoding the message as
// a localised payload and verifying the provided signature using the expected
// authority id.
func CheckMessageSignature[H comparable, N constraints.Unsigned](
	message grandpa.Message[H, N],
	id AuthorityID,
	signature AuthoritySignature,
	round RoundNumber,
	setID SetID) bool {

	buf := LocalizedPayload(round, setID, message)
	valid := id.Verify(signature, buf)

	if !valid {
		logger.Debugf("Bad signature on message from %v", id)
	}
	return valid
}

// LocalizedPayload will encode round message localised to a given round and set id.
func LocalizedPayload(round RoundNumber, setID SetID, message any) []byte {
	return scale.MustMarshal(struct {
		Message any
		RoundNumber
		SetID
	}{message, round, setID})
}
