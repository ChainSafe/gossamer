// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// AvailableDataFetchingRequest represents a request to retrieve all available data for a specific candidate.
type AvailableDataFetchingRequest struct {
	// Hash of the candidate for which the available data is requested.
	CandidateHash CandidateHash
}

// Encode returns the SCALE encoding of the AvailableDataFetchingRequest
func (a AvailableDataFetchingRequest) Encode() ([]byte, error) {
	return scale.Marshal(a)
}

type AvailableDataFetchingResponseValues interface {
	AvailableData | NoSuchData
}

// AvailableDataFetchingResponse represents the possible responses to an available data fetching request.
type AvailableDataFetchingResponse struct {
	inner any
}

func setAvailableDataFetchingResponse[Value AvailableDataFetchingResponseValues](mvdt *AvailableDataFetchingResponse, value Value) {
	mvdt.inner = value
}

func (mvdt *AvailableDataFetchingResponse) SetValue(value any) (err error) {
	switch value := value.(type) {
	case AvailableData:
		setAvailableDataFetchingResponse(mvdt, value)
		return

	case NoSuchData:
		setAvailableDataFetchingResponse(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt AvailableDataFetchingResponse) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case AvailableData:
		return 0, mvdt.inner, nil

	case NoSuchData:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt AvailableDataFetchingResponse) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt AvailableDataFetchingResponse) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(AvailableData), nil

	case 1:
		return *new(NoSuchData), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewAvailableDataFetchingResponse returns a new available data fetching response varying data type
func NewAvailableDataFetchingResponse() AvailableDataFetchingResponse {
	// vdt := scale.MustNewVaryingDataType(AvailableData{}, NoSuchData{})
	return AvailableDataFetchingResponse{}
}

// AvailableData represents the data that is kept available for each candidate included in the relay chain.
type AvailableData struct {
	// The Proof-of-Validation (PoV) of the candidate
	PoV PoV `scale:"1"`

	// The persisted validation data needed for approval checks
	ValidationData PersistedValidationData `scale:"2"`
}

// PoV represents a Proof-of-Validity block (PoV block) or a parachain block.
// It contains the necessary data for the parachain specific state transition logic.
type PoV struct {
	BlockData BlockData `scale:"1"`
}

// BlockData represents parachain block data.
// It contains everything required to validate para-block, may contain block and witness data.
type BlockData []byte

// PersistedValidationData provides information about how to create the inputs for the validation
// of a candidate by calling the Runtime.
// This information is derived from the parachain state and will vary from parachain to parachain,
// although some of the fields may be the same for every parachain.
type PersistedValidationData struct {
	// The parent head-data
	ParentHead []byte `scale:"1"`

	// The relay-chain block number this is in the context of
	RelayParentNumber parachaintypes.BlockNumber `scale:"2"`

	// The relay-chain block storage root this is in the context of
	RelayParentStorageRoot common.Hash `scale:"3"`

	// The maximum legal size of a POV block, in bytes
	MaxPovSize uint32 `scale:"4"`
}

// NoSuchData indicates that the requested data was not found.
type NoSuchData struct{}

// Encode returns the SCALE encoding of the AvailableDataFetchingResponse
func (a *AvailableDataFetchingResponse) Encode() ([]byte, error) {
	return scale.Marshal(*a)
}

// Decode returns the SCALE decoding of the AvailableDataFetchingResponse.
func (a *AvailableDataFetchingResponse) Decode(in []byte) (err error) {
	return scale.Unmarshal(in, a)
}

// String formats a AvailableDataFetchingResponse as a string
func (p *AvailableDataFetchingResponse) String() string {
	if p == nil {
		return "AvailableDataFetchingResponse=nil"
	}

	v, _ := p.Value()
	availableData, ok := v.(AvailableData)
	if !ok {
		return "AvailableDataFetchingResponse=NoSuchData"
	}
	return fmt.Sprintf("AvailableDataFetchingResponse AvailableData=%+v", availableData)
}
