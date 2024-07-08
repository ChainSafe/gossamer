// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// StatementFetchingRequest represents a request for fetching a large statement via request/response.
type StatementFetchingRequest struct {
	// Data needed to locate and identify the needed statement.
	RelayParent common.Hash `scale:"1"`

	// Hash of candidate that was used create the `CommitedCandidateReceipt`.
	CandidateHash parachaintypes.CandidateHash `scale:"2"`
}

// Encode returns the SCALE encoding of the StatementFetchingRequest.
func (s *StatementFetchingRequest) Encode() ([]byte, error) {
	return scale.Marshal(*s)
}

type StatementFetchingResponseValues interface {
	MissingDataInStatement
}

// StatementFetchingResponse represents the statement fetching response is
// sent by nodes to the clients who issued a collation fetching request.
//
// Respond with found full statement.
type StatementFetchingResponse struct {
	inner any
}

func setStatementFetchingResponse[Value StatementFetchingResponseValues](mvdt *StatementFetchingResponse, value Value) {
	mvdt.inner = value
}

func (mvdt *StatementFetchingResponse) SetValue(value any) (err error) {
	switch value := value.(type) {
	case MissingDataInStatement:
		setStatementFetchingResponse(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt StatementFetchingResponse) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case MissingDataInStatement:
		return 0, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt StatementFetchingResponse) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt StatementFetchingResponse) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(MissingDataInStatement), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// MissingDataInStatement represents the data missing to reconstruct the full signed statement.
type MissingDataInStatement parachaintypes.CommittedCandidateReceipt

// NewStatementFetchingResponse returns a new statement fetching response varying data type
func NewStatementFetchingResponse() StatementFetchingResponse {
	return StatementFetchingResponse{}
}

// Encode returns the SCALE encoding of the StatementFetchingResponse.
func (s *StatementFetchingResponse) Encode() ([]byte, error) {
	return scale.Marshal(*s)
}

// Decode returns the SCALE decoding of the StatementFetchingResponse.
func (s *StatementFetchingResponse) Decode(in []byte) (err error) {
	return scale.Unmarshal(in, s)
}

// String formats a StatementFetchingResponse as a string
func (s *StatementFetchingResponse) String() string {
	if s == nil {
		return "StatementFetchingResponse=nil"
	}
	v, _ := s.Value()
	missingData := v.(MissingDataInStatement)
	return fmt.Sprintf("StatementFetchingResponse MissingDataInStatement=%+v", missingData)
}
