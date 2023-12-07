package types

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/ChainSafe/gossamer/lib/keystore"
)

// UncheckedDisputeMessage is a dispute message where signatures of statements have not yet been checked.
type UncheckedDisputeMessage struct {
	CandidateReceipt parachainTypes.CandidateReceipt
	SessionIndex     parachainTypes.SessionIndex
	InvalidVote      Vote
	ValidVote        Vote
}

// Index returns the index of the UncheckedDisputeMessage enum
func (UncheckedDisputeMessage) Index() uint {
	return 0
}

// DisputeMessageVDT is a dispute message.
type DisputeMessageVDT scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (dm *DisputeMessageVDT) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*dm)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*dm = DisputeMessageVDT(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (dm *DisputeMessageVDT) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*dm)
	val, err = vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from varying data type: %w", err)
	}
	return val, nil
}

// NewDisputeMessageVDT creates a new DisputeMessageVDT
func NewDisputeMessageVDT() (DisputeMessageVDT, error) {
	vdt, err := scale.NewVaryingDataType(UncheckedDisputeMessage{})
	if err != nil {
		return DisputeMessageVDT{}, fmt.Errorf("failed to create varying data type: %w", err)
	}
	return DisputeMessageVDT(vdt), nil
}

// NewDisputeMessageFromSignedStatements build a `SignedDisputeMessage` and check what can be checked.
//
// This function checks that:
//
//   - both statements concern the same candidate
//   - both statements concern the same session
//   - the invalid statement is indeed an invalid one
//   - the valid statement is indeed a valid one
//   - The passed `CandidateReceipt` has the correct hash (as signed in the statements).
//   - the given validator indices match with the given `ValidatorId`s in the statements,
//     given a `SessionInfo`.
//
// We don't check whether the given `SessionInfo` matches the `SessionIndex` in the
// statements, because we can't without doing a runtime query. Nevertheless, this smart
// constructor gives relative strong guarantees that the resulting `SignedDisputeStatement` is
// valid and good.  Even the passed `SessionInfo` is most likely right if this function
// returns `Some`, because otherwise the passed `ValidatorId`s in the `SessionInfo` at
// their given index would very likely not match the `ValidatorId`s in the statements.
func NewDisputeMessageFromSignedStatements(
	validStatement SignedDisputeStatement,
	validIndex parachainTypes.ValidatorIndex,
	invalidStatement SignedDisputeStatement,
	invalidIndex parachainTypes.ValidatorIndex,
	candidateReceipt parachainTypes.CandidateReceipt,
	sessionInfo parachainTypes.SessionInfo,
) (DisputeMessageVDT, error) {
	candidateHash := validStatement.CandidateHash

	// check that both statements concern the same candidate
	if candidateHash != invalidStatement.CandidateHash {
		return DisputeMessageVDT{}, fmt.Errorf("candidate hashes do not match")
	}

	sessionIndex := validStatement.SessionIndex

	// check that both statements concern the same session
	if sessionIndex != invalidStatement.SessionIndex {
		return DisputeMessageVDT{}, fmt.Errorf("session indices do not match")
	}

	if validIndex > parachainTypes.ValidatorIndex(len(sessionInfo.Validators)) {
		return DisputeMessageVDT{}, fmt.Errorf("invalid validator index")
	}
	validID := sessionInfo.Validators[validIndex]
	if validID != validStatement.ValidatorPublic {
		return DisputeMessageVDT{}, fmt.Errorf("valid validator ID does not match")
	}

	if invalidIndex > parachainTypes.ValidatorIndex(len(sessionInfo.Validators)) {
		return DisputeMessageVDT{}, fmt.Errorf("invalid validator index")
	}
	invalidID := sessionInfo.Validators[invalidIndex]
	if invalidID != invalidStatement.ValidatorPublic {
		return DisputeMessageVDT{}, fmt.Errorf("invalid validator ID does not match")
	}

	candidateReceiptHash, err := candidateReceipt.Hash()
	if err != nil {
		return DisputeMessageVDT{}, fmt.Errorf("hash candidate receipt: %w", err)
	}

	// check that the passed `CandidateReceipt` has the correct hash (as signed in the statements)
	if candidateReceiptHash != candidateHash {
		return DisputeMessageVDT{}, fmt.Errorf("candidate receipt hash does not match")
	}

	kind, err := validStatement.DisputeStatement.Value()
	if err != nil {
		return DisputeMessageVDT{}, fmt.Errorf("get valid dispute statement value: %w", err)
	}
	validKind, ok := kind.(inherents.ValidDisputeStatementKind)
	if !ok {
		return DisputeMessageVDT{}, fmt.Errorf("valid dispute statement kind has invalid type")
	}

	kind, err = invalidStatement.DisputeStatement.Value()
	if err != nil {
		return DisputeMessageVDT{}, fmt.Errorf("get invalid dispute statement value: %w", err)
	}
	invalidKind, ok := kind.(inherents.InvalidDisputeStatementKind)
	if !ok {
		return DisputeMessageVDT{}, fmt.Errorf("invalid dispute statement kind has valid type")
	}

	validVote := Vote{
		ValidatorIndex:     validIndex,
		DisputeStatement:   inherents.DisputeStatement(validKind),
		ValidatorSignature: validStatement.ValidatorSignature,
	}
	invalidVote := Vote{
		ValidatorIndex:     invalidIndex,
		DisputeStatement:   inherents.DisputeStatement(invalidKind),
		ValidatorSignature: invalidStatement.ValidatorSignature,
	}

	disputeMessage, err := NewDisputeMessageVDT()
	if err != nil {
		return DisputeMessageVDT{}, fmt.Errorf("failed to create varying data type: %w", err)
	}
	message := UncheckedDisputeMessage{
		CandidateReceipt: candidateReceipt,
		SessionIndex:     sessionIndex,
		InvalidVote:      invalidVote,
		ValidVote:        validVote,
	}
	if err := disputeMessage.Set(message); err != nil {
		return DisputeMessageVDT{}, fmt.Errorf("set dispute message: %w", err)
	}

	return disputeMessage, nil
}

// NewDisputeMessage creates a new dispute message.
func NewDisputeMessage(
	keypair keystore.KeyPair,
	votes CandidateVotes,
	ourVote *SignedDisputeStatement,
	ourIndex parachainTypes.ValidatorIndex,
	info parachainTypes.SessionInfo,
) (DisputeMessageVDT, error) {
	disputeStatement, err := ourVote.DisputeStatement.Value()
	if err != nil {
		return DisputeMessageVDT{}, fmt.Errorf("get dispute statement value: %w", err)
	}

	var (
		validStatement   SignedDisputeStatement
		validIndex       parachainTypes.ValidatorIndex
		invalidStatement SignedDisputeStatement
		invalidIndex     parachainTypes.ValidatorIndex
	)

	var firstVote Vote
	switch disputeStatement.(type) {
	case inherents.ValidDisputeStatementKind:
		if votes.Invalid.Len() == 0 {
			return DisputeMessageVDT{}, fmt.Errorf("no opposite votes")
		}
		votes.Invalid.Descend(0, func(key parachainTypes.ValidatorIndex, vote Vote) bool {
			firstVote = vote
			return true
		})

		validStatement = *ourVote
		validIndex = ourIndex

		invalidStatement, err = NewSignedDisputeStatement(
			keypair,
			false,
			ourVote.CandidateHash,
			ourVote.SessionIndex,
		)
		if err != nil {
			return DisputeMessageVDT{}, fmt.Errorf("new signed dispute statement: %w", err)
		}

		invalidIndex = firstVote.ValidatorIndex
	case inherents.InvalidDisputeStatementKind:
		if votes.Valid.Value.Len() == 0 {
			return DisputeMessageVDT{}, fmt.Errorf("no opposite votes")
		}
		votes.Valid.Value.Map.Descend(0, func(key parachainTypes.ValidatorIndex, vote Vote) bool {
			firstVote = vote
			return true
		})

		validIndex = firstVote.ValidatorIndex
		validStatement, err = NewSignedDisputeStatement(
			keypair,
			true,
			ourVote.CandidateHash,
			ourVote.SessionIndex,
		)

		invalidStatement = *ourVote
		invalidIndex = ourIndex
	}

	return NewDisputeMessageFromSignedStatements(
		validStatement,
		validIndex,
		invalidStatement,
		invalidIndex,
		votes.CandidateReceipt,
		info,
	)
}

// ImportStatements import statements by validators about a candidate
type ImportStatements struct {
	CandidateReceipt parachainTypes.CandidateReceipt
	Session          parachainTypes.SessionIndex
	Statements       []Statement
}

// RecentDisputesMessage message to request recent disputes
type RecentDisputesMessage struct{}

// ActiveDisputes message to request active disputes
type ActiveDisputes struct{}

// CandidateVotesQuery message to request candidate votes
type CandidateVotesQuery struct {
	Session       parachainTypes.SessionIndex
	CandidateHash common.Hash
}

// QueryCandidateVotes message to request candidate votes
type QueryCandidateVotes struct {
	Queries []CandidateVotesQuery
}

// QueryCandidateVotesResponse response to a candidate votes query
type QueryCandidateVotesResponse struct {
	Session       parachainTypes.SessionIndex
	CandidateHash common.Hash
	Votes         *CandidateVotes
}

// IssueLocalStatementMessage message to issue a local statement
type IssueLocalStatementMessage struct {
	Session          parachainTypes.SessionIndex
	CandidateHash    common.Hash
	CandidateReceipt parachainTypes.CandidateReceipt
	Valid            bool
}

// BlockDescription describes a block with its session and candidates
type BlockDescription struct {
	BlockHash  common.Hash
	Session    parachainTypes.SessionIndex
	Candidates []parachainTypes.CandidateHash
}

// DetermineUndisputedChainMessage message to determine the undisputed chain
type DetermineUndisputedChainMessage struct {
	Base              overseer.Block
	BlockDescriptions []BlockDescription
}

// DetermineUndisputedChainResponse response to a DetermineUndisputedChainMessage
type DetermineUndisputedChainResponse struct {
	Block overseer.Block
	Err   error
}

// Message messages to be handled in this subsystem.
type Message[data any] struct {
	Data            data
	ResponseChannel chan any
}
