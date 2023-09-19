// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"bytes"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	runtimewazero "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ValidationResult is result received from validate_block. It is  similar to CandidateCommitments, but different order.
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

// ValidationParameters contains parameters for evaluating the parachain validity function.
type ValidationParameters struct {
	// Previous head-data.
	ParentHeadData parachaintypes.HeadData
	// The collation body.
	BlockData []byte //types.BlockData
	// The current relay-chain block number.
	RelayParentNumber uint32
	// The relay-chain block's storage root.
	RelayParentStorageRoot common.Hash
}

// Instance is a wrapper around the wazero runtime instance.
type Instance struct {
	*runtimewazero.Instance
}

func SetupVM(code []byte) (*Instance, error) {
	cfg := runtimewazero.Config{}

	instance, err := runtimewazero.NewInstance(code, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating instance: %w", err)
	}
	return &Instance{instance}, nil
}

// ValidateBlock validates a block by calling parachain runtime's validate_block call and returns the result.
func (in *Instance) ValidateBlock(params ValidationParameters) (
	*ValidationResult, error) {

	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err := encoder.Encode(params)
	if err != nil {
		return nil, fmt.Errorf("encoding validation parameters: %w", err)
	}

	encodedValidationResult, err := in.Exec("validate_block", buffer.Bytes())
	if err != nil {
		return nil, err
	}

	validationResult := ValidationResult{}
	err = scale.Unmarshal(encodedValidationResult, &validationResult)
	if err != nil {
		return nil, fmt.Errorf("scale decoding: %w", err)
	}
	return &validationResult, nil
}

// RuntimeInstance for runtime methods
type RuntimeInstance interface {
	ParachainHostPersistedValidationData(parachaidID uint32, assumption parachaintypes.OccupiedCoreAssumption,
	) (*parachaintypes.PersistedValidationData, error)
	ParachainHostValidationCode(parachaidID uint32, assumption parachaintypes.OccupiedCoreAssumption,
	) (*parachaintypes.ValidationCode, error)
	ParachainHostCheckValidationOutputs(parachainID uint32, outputs parachaintypes.CandidateCommitments) (bool, error)
	ParachainHostValidationCodeByHash(
		blockHash common.Hash,
		validationCodeHash parachaintypes.ValidationCodeHash,
	) (*parachaintypes.ValidationCode, error)
	ParachainHostOnChainVotes(blockHash common.Hash) (*parachaintypes.ScrapedOnChainVotes, error)
	ParachainHostCandidateEvents(blockHash common.Hash) (*scale.VaryingDataTypeSlice, error)
	ParachainHostSessionInfo(blockHash common.Hash, sessionIndex parachaintypes.SessionIndex) (
		*parachaintypes.SessionInfo, error)
	ParachainHostSessionIndexForChild(blockHash common.Hash) (parachaintypes.SessionIndex, error)
}
