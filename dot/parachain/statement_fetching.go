// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// StatementFetchingRequest represents a request for fetching a large statement via request/response.
type StatementFetchingRequest struct {
	// Data needed to locate and identify the needed statement.
	RelayParent common.Hash `scale:"1"`

	// Hash of candidate that was used create the `CommitedCandidateRecept`.
	CandidateHash CandidateHash `scale:"2"`
}

// Encode returns the SCALE encoding of the StatementFetchingRequest.
func (s *StatementFetchingRequest) Encode() ([]byte, error) {
	return scale.Marshal(*s)
}

// StatementFetchingResponse represents the statement fetching response is
// sent by nodes to the clients who issued a collation fetching request.
//
// Respond with found full statement.
type StatementFetchingResponse scale.VaryingDataType

// MissingDataInStatement represents the data missing to reconstruct the full signed statement.
type MissingDataInStatement parachaintypes.CommittedCandidateReceipt

// Index returns the index of varying data type
func (MissingDataInStatement) Index() uint {
	return 0
}

// NewStatementFetchingResponse returns a new statement fetching response varying data type
func NewStatementFetchingResponse() StatementFetchingResponse {
	vdt := scale.MustNewVaryingDataType(MissingDataInStatement{})
	return StatementFetchingResponse(vdt)
}

// Set will set a value using the underlying  varying data type
func (s *StatementFetchingResponse) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*s)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*s = StatementFetchingResponse(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (s *StatementFetchingResponse) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
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
