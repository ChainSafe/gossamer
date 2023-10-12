// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// QueryAvailableData query a AvailableData from the AV store
type QueryAvailableData struct {
	CandidateHash common.Hash
	AvailableData AvailableData
}

type QueryDataAvailability struct {
	CandidateHash common.Hash
	Sender        chan AvailableData
}

type QueryChunk struct {
	CandidateHash  common.Hash
	ValidatorIndex uint32
	Sender         chan []byte
}

type QueryChunkSize struct {
	CandidateHash common.Hash
	Sender        chan uint32
}

type QueryAllChunks struct {
	CandidateHash common.Hash
	Sender        chan []byte
}

type QueryChunkAvailability struct {
	CandidateHash  common.Hash
	ValidatorIndex uint32
	Sender         chan bool
}

type StoreChunk struct {
	CandidateHash common.Hash
	Chunk         []byte
	Sender        chan any
}

type StoreAvailableData struct {
	CandidateHash       common.Hash
	NValidators         uint32
	AvailableData       AvailableData
	ExpectedErasureRoot common.Hash
	Sender              chan any
}

type AvailableData struct{} // Define your AvailableData type
