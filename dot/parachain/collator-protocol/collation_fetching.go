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

// CollationFetchingResponse represents a response sent by collator
type CollationFetchingResponse scale.VaryingDataType

type CollationVDT parachaintypes.Collation

// Index returns the index of varying data type
func (CollationVDT) Index() uint {
	return 0
}

// NewCollationFetchingResponse returns a new collation fetching response varying data type
func NewCollationFetchingResponse() CollationFetchingResponse {
	vdt := scale.MustNewVaryingDataType(CollationVDT{})
	return CollationFetchingResponse(vdt)
}

// Set will set a value using the underlying  varying data type
func (c *CollationFetchingResponse) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*c)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*c = CollationFetchingResponse(vdt)
	return
}

// Value returns the value from the underlying varying data type
func (c *CollationFetchingResponse) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*c)
	return vdt.Value()
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
