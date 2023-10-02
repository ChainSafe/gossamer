// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// PoVFetchingRequest represents a request to fetch the advertised collation at the relay-parent.
type PoVFetchingRequest struct {
	// Hash of the candidate for which we want to retrieve a Proof-of-Validity (PoV).
	CandidateHash parachaintypes.CandidateHash
}

// Encode returns the SCALE encoding of the PoVFetchingRequest
func (p PoVFetchingRequest) Encode() ([]byte, error) {
	return scale.Marshal(p)
}

// PoVFetchingResponse represents the possible responses to a PoVFetchingRequest.
type PoVFetchingResponse scale.VaryingDataType

// NewPoVFetchingResponse returns a new PoV fetching response varying data type
func NewPoVFetchingResponse() PoVFetchingResponse {
	vdt := scale.MustNewVaryingDataType(parachaintypes.PoV{}, NoSuchPoV{})
	return PoVFetchingResponse(vdt)
}

// Set will set a value using the underlying  varying data type
func (p *PoVFetchingResponse) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*p)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*p = PoVFetchingResponse(vdt)
	return
}

// Value returns the value from the underlying varying data type
func (p *PoVFetchingResponse) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*p)
	return vdt.Value()
}

// NoSuchPoV indicates that the requested PoV was not found in the store.
type NoSuchPoV struct{}

// Index returns the index of varying data type
func (NoSuchPoV) Index() uint {
	return 1
}

// Encode returns the SCALE encoding of the PoVFetchingResponse
func (p *PoVFetchingResponse) Encode() ([]byte, error) {
	return scale.Marshal(*p)
}

// Decode returns the SCALE decoding of the PoVFetchingResponse.
func (p *PoVFetchingResponse) Decode(in []byte) (err error) {
	return scale.Unmarshal(in, p)
}

// String formats a PoVFetchingResponse as a string
func (p *PoVFetchingResponse) String() string {
	if p == nil {
		return "PoVFetchingResponse=nil"
	}

	v, _ := p.Value()
	pov, ok := v.(parachaintypes.PoV)
	if !ok {
		return "PoVFetchingResponse=NoSuchPoV"
	}
	return fmt.Sprintf("PoVFetchingResponse PoV=%+v", pov)
}
