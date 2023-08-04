// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ChunkFetchingRequest represents a request to retrieve chunks of a parachain candidate
type ChunkFetchingRequest struct {
	// Hash of candidate we want a chunk for.
	CandidateHash CandidateHash `scale:"1"`

	// The index of the chunk to fetch.
	Index parachaintypes.ValidatorIndex `scale:"2"`
}

// Encode returns the SCALE encoding of the ChunkFetchingRequest
func (c ChunkFetchingRequest) Encode() ([]byte, error) {
	return scale.Marshal(c)
}

// ChunkFetchingResponse represents the response for a requested erasure chunk
type ChunkFetchingResponse scale.VaryingDataType

// NewChunkFetchingResponse returns a new chunk fetching response varying data type
func NewChunkFetchingResponse() ChunkFetchingResponse {
	vdt := scale.MustNewVaryingDataType(ChunkResponse{}, NoSuchChunk{})
	return ChunkFetchingResponse(vdt)
}

// Set will set a value using the underlying  varying data type
func (c *ChunkFetchingResponse) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*c)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*c = ChunkFetchingResponse(vdt)
	return
}

// Value returns the value from the underlying varying data type
func (c *ChunkFetchingResponse) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*c)
	return vdt.Value()
}

// ChunkResponse represents the requested chunk data
type ChunkResponse struct {
	// The erasure-encoded chunk of data belonging to the candidate block
	Chunk []byte `scale:"1"`

	// Proof for this chunk's branch in the Merkle tree
	Proof [][]byte `scale:"2"`
}

// Index returns the index of varying data type
func (ChunkResponse) Index() uint {
	return 0
}

// NoSuchChunk indicates that the requested chunk was not found
type NoSuchChunk struct{}

// Index returns the index of varying data type
func (NoSuchChunk) Index() uint {
	return 1
}

// Encode returns the SCALE encoding of the ChunkFetchingResponse
func (c *ChunkFetchingResponse) Encode() ([]byte, error) {
	return scale.Marshal(*c)
}

// Decode returns the SCALE decoding of the ChunkFetchingResponse.
func (c *ChunkFetchingResponse) Decode(in []byte) (err error) {
	return scale.Unmarshal(in, c)
}

// String formats a ChunkFetchingResponse as a string
func (c *ChunkFetchingResponse) String() string {
	if c == nil {
		return "ChunkFetchingResponse=nil"
	}

	v, _ := c.Value()
	chunkRes, ok := v.(ChunkResponse)
	if !ok {
		return "ChunkFetchingResponse=NoSuchChunk"
	}
	return fmt.Sprintf("ChunkFetchingResponse ChunkResponse=%+v", chunkRes)
}
