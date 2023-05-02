package grandpa

import (
	"github.com/tidwall/btree"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

type HashNumber[Hash, Number any] struct {
	Hash   Hash
	Number Number
}

type targetHashTargetNumber[Hash, Number any] struct {
	TargetHash   Hash
	TargetNumber Number
}

// A prevote for a block and its ancestors.
type Prevote[Hash, Number any] targetHashTargetNumber[Hash, Number]

// A precommit for a block and its ancestors.
type Precommit[Hash, Number any] targetHashTargetNumber[Hash, Number]

// A primary proposed block, this is a broadcast of the last round's estimate.
type PrimaryPropose[Hash, Number any] targetHashTargetNumber[Hash, Number]

// A protocol message or vote.
type Message[Hash, Number any] struct {
	value any
}

// Get the target block of the vote.
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

func (m Message[H, N]) Value() any {
	return m.value
}

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

// A signed message.
type SignedMessage[Hash, Number, Signature, ID any] struct {
	// The internal message which has been signed.
	Message Message[Hash, Number]
	// The signature on the message.
	Signature Signature
	// The Id of the signer
	ID ID
}

// A commit message which is an aggregate of precommits.
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

// A signed prevote message.
type SignedPrevote[Hash, Number, Signature, ID any] struct {
	// The prevote message which has been signed.
	Prevote Prevote[Hash, Number]
	// The signature on the message.
	Signature Signature
	// The ID of the signer.
	ID ID
}

// A signed precommit message.
type SignedPrecommit[Hash, Number, Signature, ID any] struct {
	// The precommit message which has been signed.
	Precommit Precommit[Hash, Number]
	// The signature on the message.
	Signature Signature
	// The ID of the signer.
	ID ID
}

// A commit message with compact representation of authentication data.
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

// A catch-up message, which is an aggregate of prevotes and precommits necessary
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

// Authentication data for a set of many messages, currently a set of precommit signatures but
// in the future could be optimized with BLS signature aggregation.
type MultiAuthData[Signature, ID any] []struct {
	Signature Signature
	ID        ID
}

// Struct returned from `validate_commit` function with information
// about the validation result.
type CommitValidationResult struct {
	valid                   bool
	numPrecommits           uint
	numDuplicatedPrecommits uint
	numEquivocations        uint
	numInvalidVoters        uint
}

// Returns `true` if the commit is valid, which implies that the target
// block in the commit is finalized.
func (cvr CommitValidationResult) Valid() bool {
	return cvr.valid
}

// Returns the number of precommits in the commit.
func (cvr CommitValidationResult) NumPrecommits() uint {
	return cvr.numPrecommits
}

// Returns the number of duplicate precommits in the commit.
func (cvr CommitValidationResult) NumDuplicatedPrecommits() uint {
	return cvr.numDuplicatedPrecommits
}

// Returns the number of equivocated precommits in the commit.
func (cvr CommitValidationResult) NumEquiovcations() uint {
	return cvr.numEquivocations
}

// Returns the number of invalid voters in the commit, i.e. votes from
// identities that are not part of the voter set.
func (cvr CommitValidationResult) NumInvalidVoters() uint {
	return cvr.numInvalidVoters
}

// Validates a GRANDPA commit message.
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
func ValidateCommit[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered](
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
			validationResult.numInvalidVoters += 1
			continue
		}
		validPrecommits = append(validPrecommits, signed)
	}

	// the base of the round should be the lowest block for which we can find a
	// precommit (any vote would only have been accepted if it was targetting a
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
	} else {
		base = targets[0]
	}

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
		importResult, err := round.ImportPrecommit(chain, signedPrecommit.Precommit, signedPrecommit.ID, signedPrecommit.Signature)
		if err != nil {
			return CommitValidationResult{}, err
		}
		switch {
		case importResult.Equivocation != nil:
			validationResult.numEquivocations += 1
			// allow only one equivocation per voter, as extras are redundant.
			if equivocated.Contains(signedPrecommit.ID) {
				return validationResult, nil
			}
			equivocated.Insert(signedPrecommit.ID)
		default:
			if importResult.Duplicated {
				validationResult.numDuplicatedPrecommits += 1
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

// Historical votes seen in a round.
type HistoricalVotes[Hash, Number, Signature, ID any] struct {
	seen         []SignedMessage[Hash, Number, Signature, ID]
	prevoteIdx   *uint64
	precommitIdx *uint64
}

// Create a new HistoricalVotes.
func NewHistoricalVotes[Hash, Number, Signature, ID any]() HistoricalVotes[Hash, Number, Signature, ID] {
	return HistoricalVotes[Hash, Number, Signature, ID]{
		seen:         make([]SignedMessage[Hash, Number, Signature, ID], 0),
		prevoteIdx:   nil,
		precommitIdx: nil,
	}
}

// Push a vote into the list. The value of `self` before this call
// is considered to be a prefix of the value post-call.
func (hv *HistoricalVotes[Hash, Number, Signature, ID]) PushVote(msg SignedMessage[Hash, Number, Signature, ID]) {
	hv.seen = append(hv.seen, msg)
}

// Set the number of messages seen before prevoting.
func (hv *HistoricalVotes[Hash, Number, Signature, ID]) SetPrevotedIdx() {
	pi := uint64(len(hv.seen))
	hv.prevoteIdx = &pi
}

// Set the number of messages seen before precommiting.
func (hv *HistoricalVotes[Hash, Number, Signature, ID]) SetPrecommittedIdx() {
	pi := uint64(len(hv.seen))
	hv.precommitIdx = &pi
}
