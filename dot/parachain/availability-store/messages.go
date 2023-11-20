// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"fmt"
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

// QueryDataAvailability query wether a `AvailableData` exists within the AV store
//
// This is useful in cases when existence
// matters, but we don't want to necessarily pass around multiple
// megabytes of data to get a single bit of information.
type QueryDataAvailability struct {
	CandidateHash common.Hash
	Sender        chan bool
}

// ErasureChunk a chunk of erasure-encoded block data
type ErasureChunk struct {
	Chunk []byte
	Index uint32
	Proof []byte
}

// QueryChunk query an `ErasureChunk` from the AV store by candidate hash and validator index
type QueryChunk struct {
	CandidateHash  common.Hash
	ValidatorIndex uint32
	Sender         chan ErasureChunk
}

// QueryChunkSize get the size of an `ErasureChunk` from the AV store by candidate hash
type QueryChunkSize struct {
	CandidateHash common.Hash
	Sender        chan uint32
}

// QueryAllChunks query all chunks that we have for the given candidate hash
type QueryAllChunks struct {
	CandidateHash common.Hash
	Sender        chan []ErasureChunk
}

// QueryChunkAvailability query wether a `ErasureChunk` exists within the AV store
//
// This is useful in cases when existence
// matters, but we don't want to necessarily pass around multiple
// megabytes of data to get a single bit of information.
type QueryChunkAvailability struct {
	CandidateHash  common.Hash
	ValidatorIndex uint32
	Sender         chan bool
}

// StoreChunk store an `ErasureChunk` in the AV store
type StoreChunk struct {
	CandidateHash common.Hash
	Chunk         ErasureChunk
	Sender        chan any
}

// StoreAvailableData store an `AvailableData` in the AV store
type StoreAvailableData struct {
	CandidateHash       common.Hash
	NValidators         uint32
	AvailableData       AvailableData
	ExpectedErasureRoot common.Hash
	Sender              chan any
}

// AvailableData is the data we keep available for each candidate included in the relay chain
type AvailableData struct {
	PoV            parachaintypes.PoV
	ValidationData parachaintypes.PersistedValidationData
}

// CanidataMeta information about a candidate
type CandidateMeta struct {
	State         State
	DataAvailable bool
	ChunksStored  []bool
}

type StateValues interface {
	Unavailable | Unfinalized | Finalized
}

// State is the state of candidate data
type State struct {
	inner any
}

func setState[Value StateValues](mvdt *State, value Value) {
	mvdt.inner = value
}

func (mvdt *State) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Unavailable:
		setState(mvdt, value)
		return

	case Unfinalized:
		setState(mvdt, value)
		return

	case Finalized:
		setState(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt State) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Unavailable:
		return 0, mvdt.inner, nil

	case Unfinalized:
		return 1, mvdt.inner, nil

	case Finalized:
		return 2, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt State) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt State) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Unavailable), nil

	case 1:
		return *new(Unfinalized), nil

	case 2:
		return *new(Finalized), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewState creates a new State
func NewState() State {
	// vdt := scale.MustNewVaryingDataType(Unavailable{}, Unfinalized{}, Finalized{})
	return State{}
}

// Unavailable candidate data was first observed at the given time but in not available in any black
type Unavailable struct {
	Timestamp time.Time
}

// Unfinalized the candidate was first observed at the given time and was included in the given list of
// unfinalized blocks, which may be empty. The timestamp here is not used for pruning. Either
// one of these blocks will be finalized or the state will regress to `State::Unavailable`, in
// which case the same timestamp will be reused. Blocks are sorted ascending first by block
// number and then hash. candidate data was first observed at the given time and is available in at least one block
type Unfinalized struct {
	Timestamp       time.Time
	BlockNumberHash []BlockNumberHash
}

// Finalized candidate data has appeared in a finalized block and did so at the given time
type Finalized struct {
	Timestamp time.Time
}

// BlockNumberHash is a block number and hash
type BlockNumberHash struct {
	blockNumber parachaintypes.BlockNumber //nolint:unused,structcheck
	blockHash   common.Hash                //nolint:unused,structcheck
}
