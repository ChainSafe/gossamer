// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// CollationFetchingRequest represents a request to retrieve
// the advertised collation at the specified relay chain block.
type CollationFetchingRequest struct {
	// Relay parent we want a collation for
	RelayParent common.Hash `scale:"1"`

	// Parachain id of the collation
	ParaID parachaintypes.ParaID `scale:"2"`
}

// Encode returns the SCALE encoding of the CollationFetchingRequest
func (c *CollationFetchingRequest) Encode() ([]byte, error) {
	return scale.Marshal(*c)
}

type CollationFetchingResponseValues interface {
	Collation
}

// CollationFetchingResponse represents a response sent by collator
type CollationFetchingResponse struct {
	inner any
}

func setCollationFetchingResponse[Value CollationFetchingResponseValues](mvdt *CollationFetchingResponse, value Value) {
	mvdt.inner = value
}

func (mvdt *CollationFetchingResponse) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Collation:
		setCollationFetchingResponse(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt CollationFetchingResponse) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Collation:
		return 0, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt CollationFetchingResponse) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt CollationFetchingResponse) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Collation), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// Collation represents a requested collation to be delivered
type Collation struct {
	CandidateReceipt parachaintypes.CandidateReceipt `scale:"1"`
	PoV              parachaintypes.PoV              `scale:"2"`
}

// NewCollationFetchingResponse returns a new collation fetching response varying data type
func NewCollationFetchingResponse() CollationFetchingResponse {
	return CollationFetchingResponse{}
}

// Encode returns the SCALE encoding of the CollationFetchingResponse
func (c *CollationFetchingResponse) Encode() ([]byte, error) {
	return scale.Marshal(*c)
}

// Decode returns the SCALE decoding of the CollationFetchingResponse.
func (c *CollationFetchingResponse) Decode(in []byte) (err error) {
	return scale.Unmarshal(in, c)
}

// String formats a CollationFetchingResponse as a string
func (c *CollationFetchingResponse) String() string {
	if c == nil {
		return "CollationFetchingResponse=nil"
	}

	v, _ := c.Value()
	collation := v.(Collation)
	return fmt.Sprintf("CollationFetchingResponse Collation=%+v", collation)
}
