// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availability_store

import (
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// QueryAvailableData query a AvailableData from the AV store
type QueryAvailableData struct {
	CandidateHash common.Hash
	Sender        chan AvailableData
}

type QueryDataAvailability struct {
	CandidateHash common.Hash
	Sender        chan bool
}

type ErasureChunk struct {
	Chunk []byte
	Index uint32
	Proof []byte
}
type QueryChunk struct {
	CandidateHash  common.Hash
	ValidatorIndex uint32
	Sender         chan ErasureChunk
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
	Chunk         ErasureChunk
	Sender        chan any
}

type StoreAvailableData struct {
	CandidateHash       common.Hash
	NValidators         uint32
	AvailableData       AvailableData
	ExpectedErasureRoot common.Hash
	Sender              chan any
}

type AvailableData struct {
	PoV            parachaintypes.PoV
	ValidationData parachaintypes.PersistedValidationData
}

type CandidateMeta struct {
	State         State
	DataAvailable bool
	ChunksStored  []byte
}

type State scale.VaryingDataType

func NewState() State {
	vdt := scale.MustNewVaryingDataType(Unavailable{}, Unfinalized{}, Finalized{})
	return State(vdt)
}

type Unavailable struct {
	Timestamp time.Time
}

func (Unavailable) Index() uint {
	return 0
}

type Unfinalized struct {
	Timestamp       time.Time
	BlockNumberHash []BlockNumberHash
}

func (Unfinalized) Index() uint {
	return 1
}

type Finalized struct {
	Timestamp time.Time
}

func (Finalized) Index() uint {
	return 2
}

type BlockNumberHash struct {
	blockNumber parachaintypes.BlockNumber
	blockHash   common.Hash
}
