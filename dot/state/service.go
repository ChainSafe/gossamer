// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/metrics"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/pkg/trie"
	inmemory_trie "github.com/ChainSafe/gossamer/pkg/trie/inmemory"
)

var logger = log.NewFromGlobal(
	log.AddContext("pkg", "state"),
)

// Service is the struct that holds storage, block and network states
type Service struct {
	dbPath      string
	logLvl      log.Level
	db          database.Database
	isMemDB     bool // set to true if using an in-memory database; only used for testing.
	Base        *BaseState
	Storage     *InmemoryStorageState
	Block       *BlockState
	Transaction *TransactionState
	Epoch       *EpochState
	Grandpa     *GrandpaState
	Slot        *SlotState
	closeCh     chan interface{}

	PrunerCfg pruner.Config
	Telemetry Telemetry

	// Below are for testing only.
	BabeThresholdNumerator   uint64
	BabeThresholdDenominator uint64
}

// Pause Pauses the state service
func (s *Service) Pause() error {
	if !s.Block.IsPaused() {
		return s.Block.Pause()
	}
	return nil
}

// Config is the default configuration used by state service.
type Config struct {
	Path      string
	LogLevel  log.Level
	PrunerCfg pruner.Config
	Telemetry Telemetry
	Metrics   metrics.IntervalConfig
}

// NewService create a new instance of Service
func NewService(config Config) *Service {
	logger.Patch(log.SetLevel(config.LogLevel))

	return &Service{
		dbPath:    config.Path,
		logLvl:    config.LogLevel,
		db:        nil,
		isMemDB:   false,
		Storage:   nil,
		Block:     nil,
		closeCh:   make(chan interface{}),
		PrunerCfg: config.PrunerCfg,
		Telemetry: config.Telemetry,
	}
}

// UseMemDB tells the service to use an in-memory key-value store instead of a persistent database.
// This should be called after NewService, and before Initialise.
// This should only be used for testing.
func (s *Service) UseMemDB() {
	s.isMemDB = true
}

// DB returns the Service's database
func (s *Service) DB() database.Database {
	return s.db
}

// SetupBase intitializes state.Base property with
// the instance of a chain.NewBadger database
func (s *Service) SetupBase() error {
	if s.isMemDB {
		return nil
	}

	basepath, err := filepath.Abs(s.dbPath)
	if err != nil {
		return err
	}

	// initialise database
	db, err := database.LoadDatabase(basepath, false)
	if err != nil {
		return err
	}

	s.db = db
	s.Base = NewBaseState(db)

	return nil
}

// Start initialises the Storage database and the Block database.
func (s *Service) Start() (err error) {
	if !s.isMemDB && (s.Storage != nil || s.Block != nil || s.Epoch != nil || s.Grandpa != nil) {
		return nil
	}

	tries := NewTries()
	tries.SetEmptyTrie()

	// create block state
	s.Block, err = NewBlockState(s.db, tries, s.Telemetry)
	if err != nil {
		return fmt.Errorf("failed to create block state: %w", err)
	}

	// retrieve latest header
	bestHeader, err := s.Block.GetHighestFinalisedHeader()
	if err != nil {
		return fmt.Errorf("failed to get best block hash: %w", err)
	}

	stateRoot := bestHeader.StateRoot
	logger.Debugf("start with latest state root: %s", stateRoot)

	// create storage state
	s.Storage, err = NewStorageState(s.db, s.Block, tries)
	if err != nil {
		return fmt.Errorf("failed to create storage state: %w", err)
	}

	// load current storage state trie into memory
	_, err = s.Storage.LoadFromDB(stateRoot)
	if err != nil {
		return fmt.Errorf("failed to load storage trie from database: %w", err)
	}

	// create transaction queue
	s.Transaction = NewTransactionState(s.Telemetry)

	// create epoch state
	s.Epoch, err = NewEpochState(s.db, s.Block)
	if err != nil {
		return fmt.Errorf("failed to create epoch state: %w", err)
	}

	s.Grandpa = NewGrandpaState(s.db, s.Block, s.Telemetry)
	num, _ := s.Block.BestBlockNumber()
	logger.Infof(
		"created state service with head %s, highest number %d and genesis hash %s",
		s.Block.BestBlockHash(), num, s.Block.genesisHash.String())

	s.Slot = NewSlotState(s.db)
	return nil
}

// Rewind rewinds the chain to the given block number.
// If the given number of blocks is greater than the chain height, it will rewind to genesis.
func (s *Service) Rewind(toBlock uint) error {
	num, _ := s.Block.BestBlockNumber()
	if toBlock > num {
		return fmt.Errorf("cannot rewind, given height is higher than our current height")
	}

	logger.Infof(
		"rewinding state from current height %s to desired height %d...",
		num, toBlock)

	root, err := s.Block.GetBlockByNumber(toBlock)
	if err != nil {
		return err
	}

	s.Block.bt = blocktree.NewBlockTreeFromRoot(&root.Header)

	header, err := s.Block.BestBlockHeader()
	if err != nil {
		return err
	}

	s.Block.lastFinalised = header.Hash()
	logger.Infof(
		"rewinding state for new height %s and best block hash %s...",
		header.Number, header.Hash())

	epoch, err := s.Epoch.GetEpochForBlock(header)
	if err != nil {
		return err
	}

	err = s.Epoch.SetCurrentEpoch(epoch)
	if err != nil {
		return err
	}

	s.Block.lastFinalised = header.Hash()

	// TODO: this is broken, it needs to set the latest finalised header after
	// rewinding to some block number, but there is no reverse lookup function
	// for block -> (round, setID) where it was finalised (#1859)
	err = s.Block.SetFinalisedHash(header.Hash(), 0, 0)
	if err != nil {
		return err
	}

	// update the current grandpa set ID
	prevSetID, err := s.Grandpa.GetCurrentSetID()
	if err != nil {
		return err
	}

	newSetID, err := s.Grandpa.GetSetIDByBlockNumber(header.Number)
	if err != nil {
		return err
	}

	err = s.Grandpa.setCurrentSetID(newSetID)
	if err != nil {
		return err
	}

	// remove previously set grandpa changes, need to go up to prevSetID+1 in case of a scheduled change
	for i := newSetID + 1; i <= prevSetID+1; i++ {
		err = s.Grandpa.db.Del(setIDChangeKey(i))
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop closes each state database
func (s *Service) Stop() error {
	close(s.closeCh)

	hash, err := s.Block.GetHighestFinalisedHash()
	if err != nil {
		return err
	}

	logger.Debugf("stop with best finalised hash %s", hash)

	if err = s.db.Flush(); err != nil {
		return err
	}

	return s.db.Close()
}

// Import imports the given state corresponding to the given header and sets the head of the chain
// to it. Additionally, it uses the first slot to correctly set the epoch number of the block.
func (s *Service) Import(header *types.Header, t trie.Trie, stateTrieVersion trie.TrieLayout, firstSlot uint64) error {
	var err error
	// initialise database using data directory
	if !s.isMemDB {
		s.db, err = database.LoadDatabase(s.dbPath, s.isMemDB)
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
	}

	block := &BlockState{
		db: database.NewTable(s.db, blockPrefix),
	}

	storage := &InmemoryStorageState{
		db: database.NewTable(s.db, storagePrefix),
	}

	epoch, err := NewEpochState(s.db, block)
	if err != nil {
		return err
	}

	s.Base = NewBaseState(s.db)

	if err = s.Base.storeFirstSlot(firstSlot); err != nil {
		return err
	}

	blockEpoch, err := epoch.GetEpochForBlock(header)
	if err != nil {
		return err
	}

	skipTo := blockEpoch + 1

	if err := s.Base.storeSkipToEpoch(skipTo); err != nil {
		return err
	}
	logger.Debugf("skip BABE verification up to epoch %d", skipTo)

	if err := epoch.SetCurrentEpoch(blockEpoch); err != nil {
		return err
	}

	root := stateTrieVersion.MustHash(t)
	if root != header.StateRoot {
		return fmt.Errorf("trie state root does not equal header state root")
	}

	logger.Info("importing storage trie from base path " +
		s.dbPath + " with root " + root.String() + "...")

	// TODO: all trie related db operations should be done in pkg/trie
	if inmemoryTrie, ok := t.(*inmemory_trie.InMemoryTrie); ok {
		if err := inmemoryTrie.WriteDirty(storage.db); err != nil {
			return err
		}
	}

	hash := header.Hash()
	if err := block.SetHeader(header); err != nil {
		return err
	}

	// TODO: this is broken, need to know round and setID for the header as well
	if err := block.db.Put(finalisedHashKey(0, 0), hash[:]); err != nil {
		return err
	}
	if err := block.setHighestRoundAndSetID(0, 0); err != nil {
		return err
	}

	logger.Debugf(
		"Import best block hash %s with latest state root %s",
		hash, root)
	if err := s.db.Flush(); err != nil {
		return err
	}

	logger.Info("finished state import")
	if s.isMemDB {
		return nil
	}

	return s.db.Close()
}
