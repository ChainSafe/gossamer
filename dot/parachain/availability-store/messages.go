// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitystore

import (
	"errors"
	"fmt"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var errInvalidErasureRoot = errors.New("Invalid erasure root")

// QueryAvailableData query a AvailableData from the AV store
type QueryAvailableData struct {
	CandidateHash parachaintypes.CandidateHash
	Sender        chan AvailableData
}

// QueryDataAvailability query wether a `AvailableData` exists within the AV store
//
// This is useful in cases when existence
// matters, but we don't want to necessarily pass around multiple
// megabytes of data to get a single bit of information.
type QueryDataAvailability struct {
	CandidateHash parachaintypes.CandidateHash
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
	CandidateHash  parachaintypes.CandidateHash
	ValidatorIndex uint32
	Sender         chan ErasureChunk
}

// QueryChunkSize get the size of an `ErasureChunk` from the AV store by candidate hash
type QueryChunkSize struct {
	CandidateHash parachaintypes.CandidateHash
	Sender        chan uint32
}

// QueryAllChunks query all chunks that we have for the given candidate hash
type QueryAllChunks struct {
	CandidateHash parachaintypes.CandidateHash
	Sender        chan []ErasureChunk
}

// QueryChunkAvailability query wether a `ErasureChunk` exists within the AV store
//
// This is useful in cases when existence
// matters, but we don't want to necessarily pass around multiple
// megabytes of data to get a single bit of information.
type QueryChunkAvailability struct {
	CandidateHash  parachaintypes.CandidateHash
	ValidatorIndex uint32
	Sender         chan bool
}

// StoreChunk store an `ErasureChunk` in the AV store
type StoreChunk struct {
	CandidateHash parachaintypes.CandidateHash
	Chunk         ErasureChunk
	Sender        chan any
}

// StoreAvailableData computes and checks the erasure root of `AvailableData`
// before storing its chunks in the AV store.
type StoreAvailableData struct {
	CandidateHash parachaintypes.CandidateHash
	NumValidators uint32
	AvailableData AvailableData
	// A hash of the candidate this `ASMStoreAvailableData` belongs to.
	CandidateHash parachaintypes.CandidateHash
	// The number of validators in the session.
	NumValidators uint32
	// The `AvailableData` itself.
	AvailableData AvailableData
	// Erasure root we expect to get after chunking.
	ExpectedErasureRoot common.Hash
	// channel to send result to.
	Sender chan error
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

// State is the state of candidate data
type State scale.VaryingDataType

// New will enable scale to create new instance when needed
func (State) New() State {
	return NewStateVDT()
}

// NewState creates a new State
func NewStateVDT() State {
	vdt := scale.MustNewVaryingDataType(Unavailable{}, Unfinalized{}, Finalized{})
	return State(vdt)
}

// Set will set VaryingDataTypeValue using underlying VaryingDataType
func (s *State) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*s)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value te varying data type: %w", err)
	}
	*s = State(vdt)
	return nil
}

// Unavailable candidate data was first observed at the given time but in not available in any black
type Unavailable struct {
	Timestamp Timestamp
}

// Index returns the index of the varying data type
func (Unavailable) Index() uint {
	return 0
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

// Index returns the index of the varying data type
func (Unfinalized) Index() uint {
	return 1
}

// Finalized candidate data has appeared in a finalized block and did so at the given time
type Finalized struct {
	Timestamp time.Time
}

// Index returns the index of the varying data type
func (Finalized) Index() uint {
	return 2
}

// BlockNumberHash is a block number and hash
type BlockNumberHash struct {
	blockNumber parachaintypes.BlockNumber //nolint:unused,structcheck
	blockHash   common.Hash                //nolint:unused,structcheck
}

type branches struct {
	trieStorage *trie.Trie
	root        common.Hash
	chunks      [][]byte
	currentPos  uint
}
