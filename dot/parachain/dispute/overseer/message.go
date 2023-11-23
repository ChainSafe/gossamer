package overseer

import (
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type ChainAPIMessage[message any] struct {
	Message         message
	ResponseChannel chan any
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
	ResponseChannel  chan any
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
	ResponseChannel    chan any
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

type BlockNumberResponse struct {
	Number uint32
	Err    error
}

type BlockNumberRequest struct {
	Hash common.Hash
}

type FinalizedBlockNumberRequest struct {
	Number uint32
}

type AncestorsResponse struct {
	Ancestors []common.Hash
	Error     error
}

type AncestorsRequest struct {
	Hash common.Hash
	K    uint32
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

type ApprovalVotingMessage[message any] struct {
	Message      message
	ResponseChan chan any
}

type ApprovalSignature struct {
	ValidatorIndex     parachainTypes.ValidatorIndex
	ValidatorSignature parachainTypes.ValidatorSignature
}

type ApprovalSignatureResponse struct {
	Signature []ApprovalSignature
	Error     error
}

type ApprovalSignatureForCandidate struct {
	CandidateHash common.Hash
}
