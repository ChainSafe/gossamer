// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"math/big"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/metrics"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/ChainSafe/chaindb"
)

var logger = log.NewFromGlobal(
	log.AddContext("pkg", "state"),
)

type Service interface {
	UseMemDB()
	DB() chaindb.Database
	SetupBase() error
	Start() error
	Rewind(toBlock int64) error
	Stop() error
	Import(header *types.Header, t *trie.Trie, firstSlot uint64) error
	StorageEntries(root *common.Hash) (map[string][]byte, error)
	Initialise(gen *genesis.Genesis, header *types.Header, t *trie.Trie) error
	CreateGenesisRuntime(t *trie.Trie, gen *genesis.Genesis) (runtime.Instance, error)
	BlockState() *BlockState
	StorageState() *StorageState
	TransactionState() *TransactionState
	EpochState() *EpochState
	GrandpaState() *GrandpaState
	BaseState() *BaseState

	// This is required at the moment since we are modifying internal state of the attributes
	// in the service.  TODO: Find references to these functions and add these as params to constructor?
	SetBlockState(*BlockState)
	SetEpochState(*EpochState)
	SetTelemetryClient(telemetry.Client)
}

var _ Service = &defaultService{}

// Service is the struct that holds storage, block and network states
type defaultService struct {
	dbPath      string
	logLvl      log.Level
	db          chaindb.Database
	isMemDB     bool // set to true if using an in-memory database; only used for testing.
	Base        *BaseState
	Storage     *StorageState
	Block       *BlockState
	Transaction *TransactionState
	Epoch       *EpochState
	Grandpa     *GrandpaState
	closeCh     chan interface{}

	PrunerCfg pruner.Config
	Telemetry telemetry.Client

	// Below are for testing only.
	BabeThresholdNumerator   uint64
	BabeThresholdDenominator uint64
}

func (s *defaultService) BaseState() *BaseState {
	return s.Base
}
func (s *defaultService) BlockState() *BlockState {
	return s.Block
}
func (s *defaultService) StorageState() *StorageState {
	return s.Storage
}
func (s *defaultService) TransactionState() *TransactionState {
	return s.Transaction
}
func (s *defaultService) EpochState() *EpochState {
	return s.Epoch
}
func (s *defaultService) GrandpaState() *GrandpaState {
	return s.Grandpa
}
func (s *defaultService) SetTelemetryClient(client telemetry.Client) {
	s.Telemetry = client
}
func (s *defaultService) SetEpochState(epochState *EpochState) {
	s.Epoch = epochState
}
func (s *defaultService) SetBlockState(blockState *BlockState) {
	s.Block = blockState
}

// Config is the default configuration used by state service.
type Config struct {
	Path         string
	LogLevel     log.Level
	PrunerCfg    pruner.Config
	Telemetry    telemetry.Client
	Metrics      metrics.IntervalConfig
	StorageState *StorageState
	BlockState   *BlockState
	GrandpaState *GrandpaState
}

// NewService create a new instance of Service
func NewService(config Config) Service {
	logger.Patch(log.SetLevel(config.LogLevel))

	return &defaultService{
		dbPath:    config.Path,
		logLvl:    config.LogLevel,
		db:        nil,
		isMemDB:   false,
		Storage:   config.StorageState,
		Block:     config.BlockState,
		Grandpa:   config.GrandpaState,
		closeCh:   make(chan interface{}),
		PrunerCfg: config.PrunerCfg,
		Telemetry: config.Telemetry,
	}
}

func (s *defaultService) StorageEntries(root *common.Hash) (map[string][]byte, error) {
	return s.Storage.Entries(root)
}

// UseMemDB tells the service to use an in-memory key-value store instead of a persistent database.
// This should be called after NewService, and before Initialise.
// This should only be used for testing.
func (s *defaultService) UseMemDB() {
	s.isMemDB = true
}

// DB returns the Service's database
func (s *defaultService) DB() chaindb.Database {
	return s.db
}

// SetupBase intitializes state.Base property with
// the instance of a chain.NewBadger database
func (s *defaultService) SetupBase() error {
	if s.isMemDB {
		return nil
	}

	basepath, err := filepath.Abs(s.dbPath)
	if err != nil {
		return err
	}

	// initialise database
	db, err := utils.SetupDatabase(basepath, false)
	if err != nil {
		return err
	}

	s.db = db
	s.Base = NewBaseState(db)

	return nil
}

// Start initialises the Storage database and the Block database.
func (s *defaultService) Start() error {
	if !s.isMemDB && (s.Storage != nil || s.Block != nil || s.Epoch != nil || s.Grandpa != nil) {
		return nil
	}

	var err error
	// create block state
	s.Block, err = NewBlockState(s.db, s.Telemetry)
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

	pr, err := s.Base.loadPruningData()
	if err != nil {
		return err
	}

	// create storage state
	s.Storage, err = NewStorageState(s.db, s.Block, trie.NewEmptyTrie(), pr)
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

	s.Grandpa, err = NewGrandpaState(s.db)
	if err != nil {
		return fmt.Errorf("failed to create grandpa state: %w", err)
	}

	num, _ := s.Block.BestBlockNumber()
	logger.Info("created state service with head " +
		s.Block.BestBlockHash().String() +
		", highest number " + num.String() +
		" and genesis hash " + s.Block.genesisHash.String())

	// Start background goroutine to GC pruned keys.
	go s.Storage.pruneStorage(s.closeCh)

	return nil
}

// Rewind rewinds the chain to the given block number.
// If the given number of blocks is greater than the chain height, it will rewind to genesis.
func (s *defaultService) Rewind(toBlock int64) error {
	num, _ := s.Block.BestBlockNumber()
	if toBlock > num.Int64() {
		return fmt.Errorf("cannot rewind, given height is higher than our current height")
	}

	logger.Infof(
		"rewinding state from current height %s to desired height %d...",
		num, toBlock)

	root, err := s.Block.GetBlockByNumber(big.NewInt(toBlock))
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

	//return s.Base.StoreBestBlockHash(newHead)
	return nil
}

// Stop closes each state database
func (s *defaultService) Stop() error {
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
func (s *defaultService) Import(header *types.Header, t *trie.Trie, firstSlot uint64) error {
	var err error
	// initialise database using data directory
	s.db, err = utils.SetupDatabase(s.dbPath, s.isMemDB)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err)
	}

	block := &BlockState{
		db: chaindb.NewTable(s.db, blockPrefix),
	}

	storage := &StorageState{
		db: chaindb.NewTable(s.db, storagePrefix),
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

	root := t.MustHash()
	if root != header.StateRoot {
		return fmt.Errorf("trie state root does not equal header state root")
	}

	logger.Info("importing storage trie from base path " +
		s.dbPath + " with root " + root.String() + "...")

	if err := t.Store(storage.db); err != nil {
		return err
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
