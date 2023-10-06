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
	candidateReceipt parachainTypes.CandidateReceipt
	sessionIndex     parachainTypes.SessionIndex
	invalidVote      Vote
	validVote        Vote
}

// Index returns the index of the UncheckedDisputeMessage enum
func (UncheckedDisputeMessage) Index() uint {
	return 0
}

// DisputeMessage is a dispute message.
type DisputeMessage scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (dm *DisputeMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*dm)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*dm = DisputeMessage(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (dm *DisputeMessage) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*dm)
	val, err = vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from varying data type: %w", err)
	}
	return val, nil
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
) (DisputeMessage, error) {
	candidateHash := validStatement.CandidateHash

	// check that both statements concern the same candidate
	if candidateHash != invalidStatement.CandidateHash {
		return DisputeMessage{}, fmt.Errorf("candidate hashes do not match")
	}

	sessionIndex := validStatement.SessionIndex

	// check that both statements concern the same session
	if sessionIndex != invalidStatement.SessionIndex {
		return DisputeMessage{}, fmt.Errorf("session indices do not match")
	}

	if validIndex > parachainTypes.ValidatorIndex(len(sessionInfo.Validators)) {
		return DisputeMessage{}, fmt.Errorf("invalid validator index")
	}
	validID := sessionInfo.Validators[validIndex]
	if validID != validStatement.ValidatorPublic {
		return DisputeMessage{}, fmt.Errorf("valid validator ID does not match")
	}

	if invalidIndex > parachainTypes.ValidatorIndex(len(sessionInfo.Validators)) {
		return DisputeMessage{}, fmt.Errorf("invalid validator index")
	}
	invalidID := sessionInfo.Validators[invalidIndex]
	if invalidID != invalidStatement.ValidatorPublic {
		return DisputeMessage{}, fmt.Errorf("invalid validator ID does not match")
	}

	candidateReceiptHash, err := candidateReceipt.Hash()
	if err != nil {
		return DisputeMessage{}, fmt.Errorf("hash candidate receipt: %w", err)
	}

	// check that the passed `CandidateReceipt` has the correct hash (as signed in the statements)
	if candidateReceiptHash != candidateHash {
		return DisputeMessage{}, fmt.Errorf("candidate receipt hash does not match")
	}

	kind, err := validStatement.DisputeStatement.Value()
	if err != nil {
		return DisputeMessage{}, fmt.Errorf("get valid dispute statement value: %w", err)
	}
	validKind, ok := kind.(inherents.ValidDisputeStatementKind)
	if !ok {
		return DisputeMessage{}, fmt.Errorf("valid dispute statement kind has invalid type")
	}

	kind, err = invalidStatement.DisputeStatement.Value()
	if err != nil {
		return DisputeMessage{}, fmt.Errorf("get invalid dispute statement value: %w", err)
	}
	invalidKind, ok := kind.(inherents.InvalidDisputeStatementKind)
	if !ok {
		return DisputeMessage{}, fmt.Errorf("invalid dispute statement kind has valid type")
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

	vdt, err := scale.NewVaryingDataType(UncheckedDisputeMessage{})
	if err != nil {
		return DisputeMessage{}, fmt.Errorf("failed to create varying data type: %w", err)
	}
	disputeMessage := UncheckedDisputeMessage{
		candidateReceipt: candidateReceipt,
		sessionIndex:     sessionIndex,
		invalidVote:      invalidVote,
		validVote:        validVote,
	}
	if err := vdt.Set(disputeMessage); err != nil {
		return DisputeMessage{}, fmt.Errorf("set dispute message: %w", err)
	}

	return DisputeMessage(vdt), nil
}

// NewDisputeMessage creates a new dispute message.
func NewDisputeMessage(
	keypair keystore.KeyPair,
	votes CandidateVotes,
	ourVote *SignedDisputeStatement,
	ourIndex parachainTypes.ValidatorIndex,
	info parachainTypes.SessionInfo,
) (DisputeMessage, error) {
	disputeStatement, err := ourVote.DisputeStatement.Value()
	if err != nil {
		return DisputeMessage{}, fmt.Errorf("get dispute statement value: %w", err)
	}

	var (
		validStatement   SignedDisputeStatement
		validIndex       parachainTypes.ValidatorIndex
		invalidStatement SignedDisputeStatement
		invalidIndex     parachainTypes.ValidatorIndex
	)

	var firstVote Vote
	_, ok := disputeStatement.(inherents.ValidDisputeStatementKind)
	if ok {
		votes.Invalid.Descend(nil, func(i interface{}) bool {
			firstVote, ok = i.(Vote)
			return ok
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
			return DisputeMessage{}, fmt.Errorf("new signed dispute statement: %w", err)
		}

		invalidIndex = firstVote.ValidatorIndex
	} else {
		votes.Valid.Value.Descend(nil, func(i interface{}) bool {
			firstVote, ok = i.(Vote)
			return ok
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

// ImportStatementsMessage import statements by validators about a candidate
type ImportStatementsMessage struct {
	CandidateReceipt    parachainTypes.CandidateReceipt
	Session             parachainTypes.SessionIndex
	Statements          []Statement
	PendingConfirmation overseer.Sender
}

// RecentDisputesMessage message to request recent disputes
type RecentDisputesMessage struct {
	Sender overseer.Sender
}

// ActiveDisputesMessage message to request active disputes
type ActiveDisputesMessage struct {
	Sender overseer.Sender
}

// CandidateVotesMessage message to request candidate votes
type CandidateVotesMessage struct {
	Session       parachainTypes.SessionIndex
	CandidateHash common.Hash
}

// QueryCandidateVotesMessage message to request candidate votes
type QueryCandidateVotesMessage struct {
	Sender  overseer.Sender
	Queries []CandidateVotesMessage
}

// QueryCandidateVotesResponse response to a candidate votes query
type QueryCandidateVotesResponse struct {
	Session       parachainTypes.SessionIndex
	CandidateHash common.Hash
	Votes         CandidateVotes
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
	Tx                overseer.Sender
}

// DisputeCoordinatorMessage messages received by the dispute coordinator subsystem
type DisputeCoordinatorMessage struct {
	ImportStatements         *ImportStatementsMessage
	RecentDisputes           *RecentDisputesMessage
	ActiveDisputes           *ActiveDisputesMessage
	QueryCandidateVotes      *QueryCandidateVotesMessage
	IssueLocalStatement      *IssueLocalStatementMessage
	DetermineUndisputedChain *DetermineUndisputedChainMessage
}

// OverseerSignal signals received by the overseer subsystem
type OverseerSignal struct {
	ActiveLeaves   *overseer.ActiveLeavesUpdate
	BlockFinalised *overseer.Block
	Concluded      bool
}
