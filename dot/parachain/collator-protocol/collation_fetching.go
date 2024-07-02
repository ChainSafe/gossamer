// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"fmt"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const (
	/// Maximum PoV size we support right now.
	maxPoVSize                       = 5 * 1024 * 1024
	collationFetchingRequestTimeout  = time.Millisecond * 1200
	collationFetchingMaxResponseSize = maxPoVSize + 10000 // 10MB
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
func (c CollationFetchingRequest) Encode() ([]byte, error) {
	return scale.Marshal(c)
}

type CollationVDT parachaintypes.Collation

// Index returns the index of varying data type
func (CollationVDT) Index() uint {
	return 0
}

type CollationFetchingResponseValues interface {
	parachaintypes.Collation
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
	case parachaintypes.Collation:
		setCollationFetchingResponse(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt CollationFetchingResponse) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case parachaintypes.Collation:
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
		return *new(parachaintypes.Collation), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
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
	collation := v.(CollationVDT)
	return fmt.Sprintf("CollationFetchingResponse Collation=%+v", collation)
}
