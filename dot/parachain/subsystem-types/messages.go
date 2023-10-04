// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package subsystem_types

import (
	"github.com/ChainSafe/gossamer/dot/parachain"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"time"
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

// NewCollationFetchingResponse returns a new collation fetching response varying data type
func NewAvailabilityStoreMessage() AvailabilityStoreMessage {
	vdt := scale.MustNewVaryingDataType(QueryAvailableData{})
	return AvailabilityStoreMessage(vdt)
}

type AvailableData struct{} // Define your AvailableData type

type State scale.VaryingDataType

func NewState() State {
	vdt := scale.MustNewVaryingDataType(Finalized{}, Unavailable{}, Unfinalized{})
	return State(vdt)
}

type Finalized struct {
	time.Time
}

// Index returns the index of varying data type
func (Finalized) Index() uint {
	return 0
}

type Unavailable struct {
	time.Time
}

// Index returns the index of varying data type
func (Unavailable) Index() uint {
	return 1
}

type Unfinalized struct {
	time.Time
}

// Index returns the index of varying data type
func (Unfinalized) Index() uint {
	return 2
}

type CandidateMeta struct {
	state         State
	dataAvailable bool
	chunksStored  parachain.Bitfield
}

type DBTransaction struct{}
type Config struct{}

type AvailabilityStoreSubsystem struct {
	db            interface{} // Define the actual database type
	config        Config      // Define the actual config type
	pruningConfig PruningConfig
	clock         Clock
	metrics       Metrics
}

type PruningConfig struct {
	KeepFinalizedFor   time.Duration
	KeepUnavailableFor time.Duration
}

type Clock struct {
	Now func() (time.Time, error)
}

type Metrics struct{}

func (m *Metrics) TimeGetChunk() func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		// Do something with elapsed time
		_ = elapsed
	}
}

func (m *Metrics) TimeStoreChunk() func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		// Do something with elapsed time
		_ = elapsed
	}
}

func (m *Metrics) TimeStoreAvailableData() func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		// Do something with elapsed time
		_ = elapsed
	}
}

func (m *Metrics) OnChunksReceived(count int) {
	// Do something with the received count
	_ = count
}

type PruningKey struct {
	Key    common.Hash
	Expiry time.Time
}

/*
type AvailabilityStoreSubsystemInterface interface {
	LoadAvailableData(db interface{}, config Config, candidate CandidateHash) (AvailableData, error)
	LoadMeta(db interface{}, config Config, candidate CandidateHash) (CandidateMeta, error)
	LoadChunk(db interface{}, config Config, candidate CandidateHash, validatorIndex ValidatorIndex) (ErasureChunk, error)
	StoreChunk(db interface{}, config Config, candidateHash CandidateHash, chunk ErasureChunk) (bool, error)
	SendMessage(msg AvailabilityStoreMessage) error
}

func LoadAvailableData(db interface{}, config Config, candidate CandidateHash) (AvailableData, error) {
	// Implement the logic to load available data
	return AvailableData{}, nil
}

func LoadMeta(db interface{}, config Config, candidate CandidateHash) (CandidateMeta, error) {
	// Implement the logic to load meta data
	return CandidateMeta{}, nil
}

func LoadChunk(db interface{}, config Config, candidate CandidateHash, validatorIndex ValidatorIndex) (ErasureChunk, error) {
	// Implement the logic to load a chunk
	return ErasureChunk{}, nil
}

func StoreChunk(db interface{}, config Config, candidateHash CandidateHash, chunk ErasureChunk) (bool, error) {
	// Implement the logic to store a chunk
	return true, nil
}

func SendMessage(msg AvailabilityStoreMessage) error {
	// Implement the logic to send a message
	return nil
}

//
//func UpdateBlocksAtFinalizedHeight(subsystem AvailabilityStoreSubsystemInterface, dbTransaction *DBTransaction, candidates []CandidateUpdate, blockNumber int, now time.Duration) error {
//	for _, candidateUpdate := range candidates {
//		candidateHash := candidateUpdate.CandidateHash
//		isFinalized := candidateUpdate.IsFinalized
//
//		meta, err := subsystem.LoadMeta(subsystem, subsystem.config, candidateHash)
//		if err != nil {
//			return err
//		}
//
//		if isFinalized {
//			switch meta.State.(type) {
//			case StateFinalized:
//				continue // sanity
//			case StateUnavailable:
//				deletePruningKey(dbTransaction, subsystem.config, now, candidateHash)
//			case StateUnfinalized:
//				blocks := meta.State.(StateUnfinalized).Blocks
//				for _, block := range blocks {
//					if block.Number != blockNumber {
//						deleteUnfinalizedInclusion(dbTransaction, subsystem.config, block.Number, block.Hash, candidateHash)
//					}
//				}
//			}
//
//			meta.State = StateFinalized{now}
//
//			writeMeta(dbTransaction, subsystem.config, candidateHash, meta)
//			writePruningKey(dbTransaction, subsystem.config, now+subsystem.pruningConfig.KeepFinalizedFor, candidateHash)
//		} else {
//			switch meta.State.(type) {
//			case StateFinalized:
//				continue // sanity
//			case StateUnavailable:
//				continue // sanity
//			case StateUnfinalized:
//				blocks := meta.State.(StateUnfinalized).Blocks
//				filteredBlocks := make([]Block, 0, len(blocks))
//				for _, block := range blocks {
//					if block.Number != blockNumber {
//						filteredBlocks = append(filteredBlocks, block)
//					}
//				}
//				meta.State = StateUnfinalized{now, filteredBlocks}
//
//				writeMeta(dbTransaction, subsystem.config, candidateHash, meta)
//			}
//		}
//	}
//
//	return nil
//}

func ProcessMessage(subsystem AvailabilityStoreSubsystemInterface, msg AvailabilityStoreMessage) error {
	switch msg := msg.(type) {
	case QueryAvailableData:
		data, err := subsystem.LoadAvailableData(subsystem, subsystem.config, msg.CandidateHash)
		if err != nil {
			return err
		}
		msg.Tx <- data
	case QueryDataAvailability:
		meta, err := subsystem.LoadMeta(subsystem, subsystem.config, msg.CandidateHash)
		if err != nil {
			return err
		}
		msg.Tx <- meta.DataAvailable
	case QueryChunk:
		defer msg.Timer() // Use the provided timer function
		chunk, err := subsystem.LoadChunk(subsystem, msg.CandidateHash, msg.ValidatorIndex)
		if err != nil {
			return err
		}
		msg.Tx <- chunk
	case QueryChunkSize:
		meta, err := subsystem.LoadMeta(subsystem, subsystem.config, msg.CandidateHash)
		if err != nil {
			return err
		}
		var validatorIndex ValidatorIndex
		if meta != nil {
			validatorIndex = meta.ChunksStored.FirstOne()
		}
		var maybeChunkSize *int
		if validatorIndex != nil {
			chunk, err := subsystem.LoadChunk(subsystem, msg.CandidateHash, validatorIndex)
			if err != nil {
				return err
			}
			size := len(chunk)
			maybeChunkSize = &size
		}
		msg.Tx <- maybeChunkSize
	case QueryAllChunks:
		meta, err := subsystem.LoadMeta(subsystem, subsystem.config, msg.CandidateHash)
		if err != nil {
			return err
		}
		var chunks []ErasureChunk
		if meta != nil {
			for index, stored := range meta.ChunksStored {
				if stored {
					defer msg.Timer() // Use the provided timer function
					chunk, err := subsystem.LoadChunk(subsystem, msg.CandidateHash, ValidatorIndex(index))
					if err != nil {
						return err
					}
					chunks = append(chunks, chunk)
				}
			}
		}
		msg.Tx <- chunks
	case QueryChunkAvailability:
		meta, err := subsystem.LoadMeta(subsystem, subsystem.config, msg.CandidateHash)
		if err != nil {
			return err
		}
		var availability bool
		if meta != nil && msg.ValidatorIndex < len(meta.ChunksStored) {
			availability = meta.ChunksStored[msg.ValidatorIndex]
		}
		msg.Tx <- availability
	case StoreChunk:
		subsystem.Metrics.OnChunksReceived(1)
		defer msg.Timer() // Use the provided timer function
		stored, err := subsystem.StoreChunk(subsystem, msg.CandidateHash, msg.Chunk)
		if err != nil {
			return err
		}
		if stored {
			msg.Tx <- nil
		} else {
			msg.Tx <- errors.New("StoreChunk failed")
		}
	case StoreAvailableData:
		subsystem.Metrics.OnChunksReceived(msg.NValidators)
		defer msg.Timer() // Use the provided timer function
		err := subsystem.StoreAvailableData(subsystem, msg.CandidateHash, msg.NValidators, msg.AvailableData, msg.ExpectedErasureRoot)
		if err == nil {
			msg.Tx <- nil
		} else if err == ErrorInvalidErasureRoot {
			msg.Tx <- StoreAvailableDataErrorInvalidErasureRoot
		} else {
			// We do not bubble up internal errors to caller subsystems, instead the
			// tx channel is dropped and that error is caught by the caller subsystem.
			//
			// We bubble up the specific error here so `av-store` logs still tell what
			// happened.
			return err
		}
	}
	return nil
}
*/
