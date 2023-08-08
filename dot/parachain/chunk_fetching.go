// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
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

type ChunkFetchingResponseValues interface {
	ChunkResponse | NoSuchChunk
}

type ChunkFetchingResponse struct {
	inner any
}

func setChunkFetchingResponse[Value ChunkFetchingResponseValues](mvdt *ChunkFetchingResponse, value Value) {
	mvdt.inner = value
}

func (mvdt *ChunkFetchingResponse) SetValue(value any) (err error) {
	switch value := value.(type) {
	case ChunkResponse:
		setChunkFetchingResponse(mvdt, value)
		return

	case NoSuchChunk:
		setChunkFetchingResponse(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt ChunkFetchingResponse) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case ChunkResponse:
		return 0, mvdt.inner, nil

	case NoSuchChunk:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt ChunkFetchingResponse) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt ChunkFetchingResponse) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(ChunkResponse), nil

	case 1:
		return *new(NoSuchChunk), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewChunkFetchingResponse returns a new chunk fetching response varying data type
func NewChunkFetchingResponse() ChunkFetchingResponse {
	return ChunkFetchingResponse{}
}

// ChunkResponse represents the requested chunk data
type ChunkResponse struct {
	// The erasure-encoded chunk of data belonging to the candidate block
	Chunk []byte `scale:"1"`

	// Proof for this chunk's branch in the Merkle tree
	Proof [][]byte `scale:"2"`
}

// NoSuchChunk indicates that the requested chunk was not found
type NoSuchChunk struct{}

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
