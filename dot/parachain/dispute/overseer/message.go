package overseer

import (
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type ChainAPIMessage struct {
	RelayParent     parachainTypes.CandidateHash
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
