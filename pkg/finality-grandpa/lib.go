// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/tidwall/btree"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

// HashNumber contains a block hash and block number
type HashNumber[Hash, Number any] struct {
	Hash   Hash
	Number Number
}

type targetHashTargetNumber[Hash, Number any] struct {
	TargetHash   Hash
	TargetNumber Number
}

// Prevote is a prevote for a block and its ancestors.
type Prevote[Hash, Number any] targetHashTargetNumber[Hash, Number]

// Precommit is a precommit for a block and its ancestors.
type Precommit[Hash, Number any] targetHashTargetNumber[Hash, Number]

// PrimaryPropose is a primary proposed block, this is a broadcast of the last round's estimate.
type PrimaryPropose[Hash, Number any] targetHashTargetNumber[Hash, Number]

// Chain context necessary for implementation of the finality gadget.
type Chain[Hash, Number comparable] interface {
	// Get the ancestry of a block up to but not including the base hash.
	// Should be in reverse order from `block`'s parent.
	//
	// If the block is not a descendent of `base`, returns an error.
	Ancestry(base, block Hash) ([]Hash, error)
	// Returns true if `block` is a descendent of or equal to the given `base`.
	IsEqualOrDescendantOf(base, block Hash) bool
}

// Equivocation is an equivocation (double-vote) in a given round.
type Equivocation[ID constraints.Ordered, Vote, Signature comparable] struct {
	// The round number equivocated in.
	RoundNumber uint64
	// The identity of the equivocator.
	Identity ID
	// The first vote in the equivocation.
	First voteSignature[Vote, Signature]
	// The second vote in the equivocation.
	Second voteSignature[Vote, Signature]
}

// Message is a protocol message or vote.
type Message[Hash, Number any] struct {
	value any
}

// Target returns the target block of the vote.
func (m Message[H, N]) Target() HashNumber[H, N] {
	switch message := m.value.(type) {
	case Prevote[H, N]:
		return HashNumber[H, N]{
			message.TargetHash,
			message.TargetNumber,
		}
	case Precommit[H, N]:
		return HashNumber[H, N]{
			message.TargetHash,
			message.TargetNumber,
		}
	case PrimaryPropose[H, N]:
		return HashNumber[H, N]{
			message.TargetHash,
			message.TargetNumber,
		}
	default:
		panic("unsupported type")
	}
}

// Value returns the message constrained by `Messages`
func (m Message[H, N]) Value() any {
	return m.value
}

// Messages is the interface constraint for `Message`
type Messages[Hash, Number any] interface {
	Prevote[Hash, Number] | Precommit[Hash, Number] | PrimaryPropose[Hash, Number]
}

func setMessage[Hash, Number any, T Messages[Hash, Number]](m *Message[Hash, Number], val T) {
	m.value = val
}

func newMessage[Hash, Number any, T Messages[Hash, Number]](val T) (m Message[Hash, Number]) {
	msg := Message[Hash, Number]{}
	setMessage(&msg, val)
	return msg
}

// SignedMessage is a signed message.
type SignedMessage[Hash, Number, Signature, ID any] struct {
	// The internal message which has been signed.
	Message Message[Hash, Number]
	// The signature on the message.
	Signature Signature
	// The Id of the signer
	ID ID
}

// Commit is a commit message which is an aggregate of precommits.
type Commit[Hash, Number, Signature, ID any] struct {
	// The target block's hash.
	TargetHash Hash
	// The target block's number.
	TargetNumber Number
	// Precommits for target block or any block after it that justify this commit.
	Precommits []SignedPrecommit[Hash, Number, Signature, ID]
}

func (c Commit[Hash, Number, Signature, ID]) CompactCommit() CompactCommit[Hash, Number, Signature, ID] {
	precommits := make([]Precommit[Hash, Number], len(c.Precommits))
	authData := make(MultiAuthData[Signature, ID], len(c.Precommits))
	for i, signed := range c.Precommits {
		precommits[i] = signed.Precommit
		authData[i] = struct {
			Signature Signature
			ID        ID
		}{signed.Signature, signed.ID}
	}
	return CompactCommit[Hash, Number, Signature, ID]{
		TargetHash:   c.TargetHash,
		TargetNumber: c.TargetNumber,
		Precommits:   precommits,
		AuthData:     authData,
	}
}

// SignedPrevote is a signed prevote message.
type SignedPrevote[Hash, Number, Signature, ID any] struct {
	// The prevote message which has been signed.
	Prevote Prevote[Hash, Number]
	// The signature on the message.
	Signature Signature
	// The ID of the signer.
	ID ID
}

// SignedPrecommit is a signed precommit message.
type SignedPrecommit[Hash, Number, Signature, ID any] struct {
	// The precommit message which has been signed.
	Precommit Precommit[Hash, Number]
	// The signature on the message.
	Signature Signature
	// The ID of the signer.
	ID ID
}

// CompactCommit is a commit message with compact representation of authentication data.
type CompactCommit[Hash, Number, Signature, ID any] struct {
	TargetHash   Hash
	TargetNumber Number
	Precommits   []Precommit[Hash, Number]
	AuthData     MultiAuthData[Signature, ID]
}

func (cc CompactCommit[Hash, Number, Signature, ID]) Commit() Commit[Hash, Number, Signature, ID] {
	signedPrecommits := make([]SignedPrecommit[Hash, Number, Signature, ID], len(cc.Precommits))
	for i, precommit := range cc.Precommits {
		signedPrecommits[i] = SignedPrecommit[Hash, Number, Signature, ID]{
			Precommit: precommit,
			Signature: cc.AuthData[i].Signature,
			ID:        cc.AuthData[i].ID,
		}
	}
	return Commit[Hash, Number, Signature, ID]{
		TargetHash:   cc.TargetHash,
		TargetNumber: cc.TargetNumber,
		Precommits:   signedPrecommits,
	}
}

// CatchUp is a catch-up message, which is an aggregate of prevotes and precommits necessary
// to complete a round.
//
// This message contains a "base", which is a block all of the vote-targets are
// a descendent of.
type CatchUp[Hash, Number, Signature, ID any] struct {
	// Round number.
	RoundNumber uint64
	// Prevotes for target block or any block after it that justify this catch-up.
	Prevotes []SignedPrevote[Hash, Number, Signature, ID]
	// Precommits for target block or any block after it that justify this catch-up.
	Precommits []SignedPrecommit[Hash, Number, Signature, ID]
	// The base hash. See struct docs.
	BaseHash Hash
	// The base number. See struct docs.
	BaseNumber Number
}

// MultiAuthData contains authentication data for a set of many messages, currently a set of precommit signatures but
// in the future could be optimised with BLS signature aggregation.
type MultiAuthData[Signature, ID any] []struct {
	Signature Signature
	ID        ID
}

// CommitValidationResult is type returned from `ValidateCommit` with information
// about the validation result.
type CommitValidationResult struct {
	valid                   bool
	numPrecommits           uint
	numDuplicatedPrecommits uint
	numEquivocations        uint
	numInvalidVoters        uint
}

// Valid returns `true` if the commit is valid, which implies that the target
// block in the commit is finalized.
func (cvr CommitValidationResult) Valid() bool {
	return cvr.valid
}

// NumPrecommits returns the number of precommits in the commit.
func (cvr CommitValidationResult) NumPrecommits() uint {
	return cvr.numPrecommits
}

// NumDuplicatedPrecommits returns the number of duplicate precommits in the commit.
func (cvr CommitValidationResult) NumDuplicatedPrecommits() uint {
	return cvr.numDuplicatedPrecommits
}

// NumEquiovcations returns the number of equivocated precommits in the commit.
func (cvr CommitValidationResult) NumEquiovcations() uint {
	return cvr.numEquivocations
}

// NumInvalidVoters returns the number of invalid voters in the commit, i.e. votes from
// identities that are not part of the voter set.
func (cvr CommitValidationResult) NumInvalidVoters() uint {
	return cvr.numInvalidVoters
}

// ValidateCommit validates a GRANDPA commit message.
//
// For a commit to be valid the round ghost is calculated using the precommits
// in the commit message, making sure that it exists and that it is the same
// as the commit target. The precommit with the lowest block number is used as
// the round base.
//
// Signatures on precommits are assumed to have been checked.
//
// Duplicate votes or votes from voters not in the voter-set will be ignored,
// but it is recommended for the caller of this function to remove those at
// signature-verification time.
func ValidateCommit[
	Hash constraints.Ordered,
	Number constraints.Unsigned,
	Signature comparable,
	ID constraints.Ordered,
](
	commit Commit[Hash, Number, Signature, ID],
	voters VoterSet[ID],
	chain Chain[Hash, Number],
) (CommitValidationResult, error) {
	validationResult := CommitValidationResult{
		numPrecommits: uint(len(commit.Precommits)),
	}

	// filter any precommits by voters that are not part of the set
	var validPrecommits []SignedPrecommit[Hash, Number, Signature, ID]
	for _, signed := range commit.Precommits {
		if !voters.Contains(signed.ID) {
			validationResult.numInvalidVoters++
			continue
		}
		validPrecommits = append(validPrecommits, signed)
	}

	// the base of the round should be the lowest block for which we can find a
	// precommit (any vote would only have been accepted if it was targeting a
	// block higher or equal to the round base)
	var base HashNumber[Hash, Number]
	var targets []HashNumber[Hash, Number]
	for _, signed := range validPrecommits {
		targets = append(targets, HashNumber[Hash, Number]{
			Hash:   signed.Precommit.TargetHash,
			Number: signed.Precommit.TargetNumber,
		})
	}
	slices.SortFunc(targets, func(a HashNumber[Hash, Number], b HashNumber[Hash, Number]) bool {
		return a.Number < b.Number
	})
	if len(targets) == 0 {
		return validationResult, nil
	}
	base = targets[0]

	// check that all precommits are for blocks that are equal to or descendants
	// of the round base
	var allPrecommitsHigherThanBase bool
	for i, signed := range validPrecommits {
		if chain.IsEqualOrDescendantOf(base.Hash, signed.Precommit.TargetHash) {
			if i == len(validPrecommits)-1 {
				allPrecommitsHigherThanBase = true
			}
			continue
		}
		break
	}

	if !allPrecommitsHigherThanBase {
		return validationResult, nil
	}

	equivocated := &btree.Set[ID]{}

	// add all precommits to the round with correct counting logic
	round := NewRound[ID, Hash, Number, Signature](
		RoundParams[ID, Hash, Number]{
			RoundNumber: 0,
			Voters:      voters,
			Base:        base,
		},
	)

	for _, signedPrecommit := range validPrecommits {
		importResult, err := round.importPrecommit(
			chain,
			signedPrecommit.Precommit,
			signedPrecommit.ID,
			signedPrecommit.Signature,
		)
		if err != nil {
			return CommitValidationResult{}, err
		}
		switch {
		case importResult.Equivocation != nil:
			validationResult.numEquivocations++
			// allow only one equivocation per voter, as extras are redundant.
			if equivocated.Contains(signedPrecommit.ID) {
				return validationResult, nil
			}
			equivocated.Insert(signedPrecommit.ID)
		default:
			if importResult.Duplicated {
				validationResult.numDuplicatedPrecommits++
			}
		}
	}

	// for the commit to be valid, then a precommit ghost must be found for the
	// round and it must be equal to the commit target
	precommitGHOST := round.PrecommitGHOST()
	switch {
	case precommitGHOST != nil:
		if precommitGHOST.Hash == commit.TargetHash && precommitGHOST.Number == commit.TargetNumber {
			validationResult.valid = true
		}
	default:
	}

	return validationResult, nil
}

// HistoricalVotes are the historical votes seen in a round.
type HistoricalVotes[Hash, Number, Signature, ID any] struct {
	seen         []SignedMessage[Hash, Number, Signature, ID]
	prevoteIdx   *uint64
	precommitIdx *uint64
}

// NewHistoricalVotes creates a new HistoricalVotes.
func NewHistoricalVotes[Hash, Number, Signature, ID any]() HistoricalVotes[Hash, Number, Signature, ID] {
	return HistoricalVotes[Hash, Number, Signature, ID]{
		seen:         make([]SignedMessage[Hash, Number, Signature, ID], 0),
		prevoteIdx:   nil,
		precommitIdx: nil,
	}
}

// PushVote pushes a vote into the list.
func (hv *HistoricalVotes[Hash, Number, Signature, ID]) PushVote(msg SignedMessage[Hash, Number, Signature, ID]) {
	hv.seen = append(hv.seen, msg)
}

// SetPrevotedIdx sets the number of messages seen before prevoting.
func (hv *HistoricalVotes[Hash, Number, Signature, ID]) SetPrevotedIdx() {
	pi := uint64(len(hv.seen))
	hv.prevoteIdx = &pi
}

// SetPrecommittedIdx sets the number of messages seen before precommiting.
func (hv *HistoricalVotes[Hash, Number, Signature, ID]) SetPrecommittedIdx() {
	pi := uint64(len(hv.seen))
	hv.precommitIdx = &pi
}
