package overseer

import (
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type ChainAPIMessage struct {
	RelayParent     common.Hash
	ResponseChannel chan *uint32
}

type PersistedValidationData struct {
	ParentHead             []byte
	RelayParentNumber      uint32
	RelayParentStorageRoot common.Hash
	MaxPOVSize             uint32
}

type AvailableData struct {
	POV            []byte
	ValidationData PersistedValidationData
}

type RecoveryError uint32

const (
	RecoveryErrorInvalid RecoveryError = iota
	RecoveryErrorUnavailable
)

func (e RecoveryError) String() string {
	switch e {
	case RecoveryErrorInvalid:
		return "invalid"
	case RecoveryErrorUnavailable:
		return "unavailable"
	default:
		return "unknown"
	}
}

type AvailabilityRecoveryResponse struct {
	AvailableData *AvailableData
	Error         *RecoveryError
}

type AvailabilityRecoveryMessage struct {
	CandidateReceipt parachainTypes.CandidateReceipt
	SessionIndex     parachainTypes.SessionIndex
	GroupIndex       *uint32
	ResponseChannel  chan AvailabilityRecoveryResponse
}

type PvfExecTimeoutKind uint32

const (
	PvfExecTimeoutKindBacking PvfExecTimeoutKind = iota
	PvfExecTimeoutKindApproval
)

type ValidateFromChainState struct {
	CandidateReceipt   parachainTypes.CandidateReceipt
	PoV                []byte
	PvfExecTimeoutKind PvfExecTimeoutKind
	ResponseChannel    chan ValidationResult
}

type ValidValidationResult struct {
	CandidateCommitments    parachainTypes.CandidateCommitments
	PersistedValidationData parachainTypes.PersistedValidationData
}

type InvalidValidationResult struct {
	Reason string
}

type ValidationResult struct {
	IsValid       bool
	Error         error
	ValidResult   *ValidValidationResult
	InvalidResult *InvalidValidationResult
}

type FinalizedBlockNumberResponse struct {
	Number uint32
	Err    error
}

type FinalizedBlockNumberRequest struct {
	ResponseChannel chan FinalizedBlockNumberResponse
}

type AncestorsResponse struct {
	Ancestors []common.Hash
	Error     error
}

type AncestorsRequest struct {
	Hash            common.Hash
	K               uint32
	ResponseChannel chan AncestorsResponse
}

type ApprovalSignature struct {
	ValidatorIndex     parachainTypes.ValidatorIndex
	ValidatorSignature common.Hash
}

type ApprovalSignatureResponse struct {
	Signature []ApprovalSignature
	Error     error
}

type GetApprovalSignatureForCandidate struct {
	CandidateHash common.Hash
	ResponseChan  chan *ApprovalSignatureResponse
}

type ApprovalVotingMessage struct {
	GetApprovalSignature *GetApprovalSignatureForCandidate
}

// Block represents a block
type Block struct {
	Number uint32
	Hash   common.Hash
}

// NewBlock creates a new block
func NewBlock(blockNumber uint32, hash common.Hash) Block {
	return Block{
		Number: blockNumber,
		Hash:   hash,
	}
}

type RevertBlocksRequest struct {
	Blocks []Block
}

type ChainSelectionMessage struct {
	RevertBlocks *RevertBlocksRequest
}
