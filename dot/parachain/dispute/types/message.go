package types

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type UncheckedDisputeMessage struct {
	candidateReceipt parachainTypes.CandidateReceipt
	sessionIndex     parachainTypes.SessionIndex
	invalidVote      Vote
	validVote        Vote
}

func (UncheckedDisputeMessage) Index() uint {
	return 0
}

type DisputeMessage scale.VaryingDataType

func (dm *DisputeMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*dm)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*dm = DisputeMessage(vdt)
	return nil
}

func (dm *DisputeMessage) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*dm)
	val, err = vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from varying data type: %w", err)
	}
	return val, nil
}

func NewDisputeMessage(info parachainTypes.SessionInfo,
	votes CandidateVotes,
	ourVote *SignedDisputeStatement,
	ourIndex parachainTypes.ValidatorIndex,
) (DisputeMessage, error) {
	vdt, err := scale.NewVaryingDataType(UncheckedDisputeMessage{})
	if err != nil {
		return DisputeMessage{}, fmt.Errorf("failed to create varying data type: %w", err)
	}

	return DisputeMessage(vdt), nil
}

type ImportStatementsMessage struct {
	CandidateReceipt    parachainTypes.CandidateReceipt
	Session             parachainTypes.SessionIndex
	Statements          []Statement
	PendingConfirmation overseer.Sender
}

type RecentDisputesMessage struct {
	Sender overseer.Sender
}

type ActiveDisputesMessage struct {
	Sender overseer.Sender
}

type CandidateVotesMessage struct {
	Session       parachainTypes.SessionIndex
	CandidateHash parachainTypes.CandidateHash
}

type QueryCandidateVotesMessage struct {
	Sender  overseer.Sender
	Queries []CandidateVotesMessage
}

type IssueLocalStatementMessage struct {
	Session          parachainTypes.SessionIndex
	CandidateHash    parachainTypes.CandidateHash
	CandidateReceipt parachainTypes.CandidateReceipt
	Valid            bool
}

type Block struct {
	BlockNumber uint32
	Hash        common.Hash
}

type BlockDescription struct {
	BlockHash  common.Hash
	Session    parachainTypes.SessionIndex
	Candidates []parachainTypes.CandidateHash
}

type DetermineUndisputedChainMessage struct {
	Base             Block
	BlockDescription BlockDescription
	Tx               overseer.Sender
}

type DisputeCoordinatorMessage struct {
	ImportStatements         *ImportStatementsMessage
	RecentDisputes           *RecentDisputesMessage
	ActiveDisputes           *ActiveDisputesMessage
	QueryCandidateVotes      *QueryCandidateVotesMessage
	IssueLocalStatement      *IssueLocalStatementMessage
	DetermineUndisputedChain *DetermineUndisputedChainMessage
}

type OverseerSignal struct {
	ActiveLeaves   *overseer.ActiveLeavesUpdate
	BlockFinalised *Block
	Concluded      bool
}
