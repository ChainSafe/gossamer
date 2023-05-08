package parachaininteraction

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	parachaintypes "github.com/ChainSafe/gossamer/parachain-interaction/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Similar to CandidateCommitments, but different order.
type ValidationResult struct {
	// The head-data is the new head data that should be included in the relay chain state.
	HeadData parachaintypes.HeadData `scale:"1"`
	// NewValidationCode is an update to the validation code that should be scheduled in the relay chain.
	NewValidationCode *parachaintypes.ValidationCode `scale:"2"`
	// UpwardMessages are upward messages send by the Parachain.
	UpwardMessages []parachaintypes.UpwardMessage `scale:"3"`
	// HorizontalMessages are Outbound horizontal messages sent by the parachain.
	HorizontalMessages []parachaintypes.OutboundHrmpMessage `scale:"4"`

	// The number of messages processed from the DMQ. It is expected that the Parachain processes them from first to last.
	ProcessedDownwardMessages uint32 `scale:"5"`
	// The mark which specifies the block number up to which all inbound HRMP messages are processed.
	HrmpWatermark uint32 `scale:"6"`
}

// TODO: Implement PoV requestor
type PoVRequestor interface {
	RequestPoV(povHash common.Hash) PoV
}

func getValidationData(runtimeInstance RuntimeInstance, paraID uint32) (*parachaintypes.PersistedValidationData, *parachaintypes.ValidationCode, error) {
	var mergedError error

	for _, assumptionValue := range []scale.VaryingDataTypeValue{Included{}, TimedOut{}, Free{}} {
		assumption := parachaintypes.OccupiedCoreAssumption{}
		assumption.Set(assumptionValue)
		PersistedValidationData, err := runtimeInstance.ParachainHostPersistedValidationData(paraID, assumption)
		if err != nil {
			mergedError = fmt.Errorf("%s %w", mergedError, err)
			continue
		}

		validationCode, err := runtimeInstance.ParachainHostValidationCode(paraID, assumption)
		if err != nil {
			return nil, nil, fmt.Errorf("getting validation code: %w", err)
		}

		return PersistedValidationData, validationCode, nil
	}

	return nil, nil, fmt.Errorf("getting persisted validation data: %w", mergedError)
}

func ValidateFromChainState(runtimeInstance RuntimeInstance, povRequestor PoVRequestor, c CandidateReceipt) (*parachaintypes.CandidateCommitments, *parachaintypes.PersistedValidationData, bool, error) {
	PersistedValidationData, validationCode, err := getValidationData(runtimeInstance, c.descriptor.ParaID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("getting validation data: %w", err)
	}

	// check that the candidate does not exceed any parameters in the persisted validation data
	pov := povRequestor.RequestPoV(c.descriptor.PoVHash)

	// basic checks

	// check if encoded size of pov is less than max pov size
	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err = encoder.Encode(pov)
	if err != nil {
		return nil, nil, false, fmt.Errorf("encoding pov: %w", err)
	}
	encoded_pov_size := buffer.Len()
	if encoded_pov_size > int(PersistedValidationData.MaxPovSize) {
		return nil, nil, false, errors.New("validation input is over the limit")
	}

	validationCodeHash, err := common.Blake2bHash([]byte(*validationCode))
	if err != nil {
		return nil, nil, false, fmt.Errorf("hashing validation code: %w", err)
	}

	if validationCodeHash != common.Hash(c.descriptor.ValidationCodeHash) {
		return nil, nil, false, errors.New("validation code hash does not match")
	}

	// check candidate signature
	err = c.descriptor.CheckCollatorSignature()
	if err != nil {
		return nil, nil, false, fmt.Errorf("verifying collator signature: %w", err)
	}

	ValidationParams := ValidationParameters{
		ParentHeadData:         PersistedValidationData.ParentHead,
		BlockData:              pov.BlockData,
		RelayParentNumber:      PersistedValidationData.RelayParentNumber,
		RelayParentStorageRoot: PersistedValidationData.RelayParentStorageRoot,
	}

	parachainRuntimeInstance, err := setupVM(*validationCode)
	if err != nil {
		return nil, nil, false, fmt.Errorf("setting up VM: %w", err)
	}

	validationResults, err := parachainRuntimeInstance.ValidateBlock(ValidationParams)
	if err != nil {
		return nil, nil, false, fmt.Errorf("executing validate_block: %w", err)
	}

	candidateCommitments := parachaintypes.CandidateCommitments{
		UpwardMessages:            validationResults.UpwardMessages,
		HorizontalMessages:        validationResults.HorizontalMessages,
		NewValidationCode:         validationResults.NewValidationCode,
		HeadData:                  validationResults.HeadData,
		ProcessedDownwardMessages: validationResults.ProcessedDownwardMessages,
		HrmpWatermark:             validationResults.HrmpWatermark,
	}

	isValid, err := runtimeInstance.ParachainHostCheckValidationOutputs(c.descriptor.ParaID, candidateCommitments)
	if err != nil {
		return nil, nil, false, fmt.Errorf("executing validate_block: %w", err)
	}

	return &candidateCommitments, PersistedValidationData, isValid, nil
}

type ValidationParameters struct {
	// Previous head-data.
	ParentHeadData HeadData
	// The collation body.
	BlockData []byte //types.BlockData
	// The current relay-chain block number.
	RelayParentNumber uint32
	// The relay-chain block's storage root.
	RelayParentStorageRoot common.Hash
}

// RuntimeInstance for runtime methods
type RuntimeInstance interface {
	UpdateRuntimeCode([]byte) error
	Stop()
	NodeStorage() runtime.NodeStorage
	NetworkService() runtime.BasicNetwork
	Keystore() *keystore.GlobalKeystore
	Validator() bool
	Exec(function string, data []byte) ([]byte, error)
	SetContextStorage(s runtime.Storage)
	GetCodeHash() common.Hash
	Version() (version runtime.Version)
	Metadata() ([]byte, error)
	BabeConfiguration() (*types.BabeConfiguration, error)
	GrandpaAuthorities() ([]types.Authority, error)
	ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error)
	InitializeBlock(header *types.Header) error
	InherentExtrinsics(data []byte) ([]byte, error)
	ApplyExtrinsic(data types.Extrinsic) ([]byte, error)
	FinalizeBlock() (*types.Header, error)
	ExecuteBlock(block *types.Block) ([]byte, error)
	DecodeSessionKeys(enc []byte) ([]byte, error)
	PaymentQueryInfo(ext []byte) (*types.RuntimeDispatchInfo, error)
	CheckInherents()
	BabeGenerateKeyOwnershipProof(slot uint64, authorityID [32]byte) (
		types.OpaqueKeyOwnershipProof, error)
	BabeSubmitReportEquivocationUnsignedExtrinsic(
		equivocationProof types.BabeEquivocationProof,
		keyOwnershipProof types.OpaqueKeyOwnershipProof,
	) error
	RandomSeed()
	OffchainWorker()
	GenerateSessionKeys()
	GrandpaGenerateKeyOwnershipProof(authSetID uint64, authorityID ed25519.PublicKeyBytes) (
		types.GrandpaOpaqueKeyOwnershipProof, error)
	GrandpaSubmitReportEquivocationUnsignedExtrinsic(
		equivocationProof types.GrandpaEquivocationProof, keyOwnershipProof types.GrandpaOpaqueKeyOwnershipProof,
	) error
	ParachainHostPersistedValidationData(parachaidID uint32, assumption parachaintypes.OccupiedCoreAssumption) (*parachaintypes.PersistedValidationData, error)
	ParachainHostValidationCode(parachaidID uint32, assumption parachaintypes.OccupiedCoreAssumption) (*parachaintypes.ValidationCode, error)
	ParachainHostCheckValidationOutputs(parachainID uint32, outputs parachaintypes.CandidateCommitments) (bool, error)
}
