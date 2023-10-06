// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// AvailabilityStoreMessage represents the possible availability store subsystem message
type AvailabilityStoreMessage scale.VaryingDataType

// QueryAvailableData query a AvailableData from the AV store
type QueryAvailableData struct {
	CandidateHash common.Hash
	AvailableData AvailableData
}

// Index returns the index of varying data type
func (QueryAvailableData) Index() uint {
	return 0
}

type QueryDataAvailability struct {
	CandidateHash common.Hash
	// TDDO: add oneshot sender
}

func (QueryDataAvailability) Index() uint {
	return 1
}

type QueryChunk struct {
	CandidateHash  common.Hash
	ValidatorIndex uint32
	// TODO: add oneshot sender
}

func (QueryChunk) Index() uint {
	return 2
}

type QueryChunkSize struct {
	CandidateHash common.Hash
	// TODO: add oneshot sender
}

func (QueryChunkSize) Index() uint {
	return 3
}

type QueryAllChunks struct {
	CandidateHash common.Hash
	// TODO: add oneshot sender
}

func (QueryAllChunks) Index() uint {
	return 4
}

type QueryChunkAvailability struct {
	CandidateHash  common.Hash
	ValidatorIndex uint32
	// TODO: add oneshot sender
}

func (QueryChunkAvailability) Index() uint {
	return 5
}

type StoreChunk struct {
	CandidateHash common.Hash
	Chunk         []byte
	// TODO: add oneshot sender
}

func (StoreChunk) Index() uint {
	return 6
}

type StoreAvailableData struct {
	CandidateHash       common.Hash
	NValidators         uint32
	AvailableData       AvailableData
	ExpectedErasureRoot common.Hash
	// TODO: add oneshot sender
}

func (StoreAvailableData) Index() uint {
	return 7
}

// NewCollationFetchingResponse returns a new collation fetching response varying data type
func NewAvailabilityStoreMessage() AvailabilityStoreMessage {
	vdt := scale.MustNewVaryingDataType(QueryAvailableData{}, QueryDataAvailability{}, QueryChunk{},
		QueryChunkSize{}, QueryAllChunks{}, QueryChunkAvailability{}, StoreChunk{}, StoreAvailableData{})
	return AvailabilityStoreMessage(vdt)
}

type AvailableData struct{} // Define your AvailableData type
