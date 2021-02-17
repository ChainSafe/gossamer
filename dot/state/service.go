// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/genesis"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/ChainSafe/chaindb"
	log "github.com/ChainSafe/log15"
)

var logger = log.New("pkg", "state")

// Service is the struct that holds storage, block and network states
type Service struct {
	dbPath      string
	logLvl      log.Lvl
	db          chaindb.Database
	isMemDB     bool // set to true if using an in-memory database; only used for testing.
	Storage     *StorageState
	Block       *BlockState
	Transaction *TransactionState
	Epoch       *EpochState
	closeCh     chan interface{}

	// Below are for testing only.
	BabeThresholdNumerator   uint64
	BabeThresholdDenominator uint64
}

// NewService create a new instance of Service
func NewService(path string, lvl log.Lvl) *Service {
	handler := log.StreamHandler(os.Stdout, log.TerminalFormat())
	handler = log.CallerFileHandler(handler)
	logger.SetHandler(log.LvlFilterHandler(lvl, handler))

	return &Service{
		dbPath:  path,
		logLvl:  lvl,
		db:      nil,
		isMemDB: false,
		Storage: nil,
		Block:   nil,
		closeCh: make(chan interface{}),
	}
}

// UseMemDB tells the service to use an in-memory key-value store instead of a persistent database.
// This should be called after NewService, and before Initialize.
// This should only be used for testing.
func (s *Service) UseMemDB() {
	s.isMemDB = true
}

// DB returns the Service's database
func (s *Service) DB() chaindb.Database {
	return s.db
}

// Initialize initializes the genesis state of the DB using the given storage trie. The trie should be loaded with the genesis storage state.
// This only needs to be called during genesis initialization of the node; it doesn't need to be called during normal startup.
func (s *Service) Initialize(gen *genesis.Genesis, header *types.Header, t *trie.Trie) error {
	var db chaindb.Database
	cfg := &chaindb.Config{}

	// check database type
	if s.isMemDB {
		cfg.InMemory = true
	}

	// get data directory from service
	basepath, err := filepath.Abs(s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to read basepath: %s", err)
	}

	cfg.DataDir = basepath

	// initialize database using data directory
	db, err = chaindb.NewBadgerDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err)
	}

	if err = db.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear database: %s", err)
	}

	if err = t.Store(chaindb.NewTable(db, storagePrefix)); err != nil {
		return fmt.Errorf("failed to write genesis trie to database: %w", err)
	}

	babeCfg, err := s.loadBabeConfigurationFromRuntime(t, gen)
	if err != nil {
		return err
	}

	// write initial genesis values to database
	if err = s.storeInitialValues(db, gen.GenesisData(), header, t); err != nil {
		return fmt.Errorf("failed to write genesis values to database: %s", err)
	}

	// create and store blockree from genesis block
	bt := blocktree.NewBlockTreeFromGenesis(header, db)
	err = bt.Store()
	if err != nil {
		return fmt.Errorf("failed to write blocktree to database: %s", err)
	}

	// create block state from genesis block
	blockState, err := NewBlockStateFromGenesis(db, header)
	if err != nil {
		return fmt.Errorf("failed to create block state from genesis: %s", err)
	}

	// create storage state from genesis trie
	storageState, err := NewStorageState(db, blockState, t)
	if err != nil {
		return fmt.Errorf("failed to create storage state from trie: %s", err)
	}

	epochState, err := NewEpochStateFromGenesis(db, babeCfg)
	if err != nil {
		return fmt.Errorf("failed to create epoch state: %s", err)
	}

	// check database type
	if s.isMemDB {
		// append memory database to state service
		s.db = db

		// append storage state and block state to state service
		s.Storage = storageState
		s.Block = blockState
		s.Epoch = epochState
	} else {
		// close database
		if err = db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %s", err)
		}
	}

	logger.Info("state", "genesis hash", blockState.genesisHash)
	return nil
}

func (s *Service) loadBabeConfigurationFromRuntime(t *trie.Trie, gen *genesis.Genesis) (*types.BabeConfiguration, error) {
	// load genesis state into database
	genTrie, err := rtstorage.NewTrieState(t)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate TrieState: %w", err)
	}

	// create genesis runtime
	rtCfg := &wasmer.Config{}
	rtCfg.Storage = genTrie
	rtCfg.LogLvl = s.logLvl

	r, err := wasmer.NewRuntimeFromGenesis(gen, rtCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis runtime: %w", err)
	}

	// load and store initial BABE epoch configuration
	babeCfg, err := r.BabeConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch genesis babe configuration: %w", err)
	}

	r.Stop()

	if s.BabeThresholdDenominator != 0 {
		babeCfg.C1 = s.BabeThresholdNumerator
		babeCfg.C2 = s.BabeThresholdDenominator
	}

	return babeCfg, nil
}

// storeInitialValues writes initial genesis values to the state database
func (s *Service) storeInitialValues(db chaindb.Database, data *genesis.Data, header *types.Header, t *trie.Trie) error {
	// write genesis trie to database
	if err := StoreTrie(chaindb.NewTable(db, storagePrefix), t); err != nil {
		return fmt.Errorf("failed to write trie to database: %s", err)
	}

	// write storage hash to database
	if err := StoreLatestStorageHash(db, t.MustHash()); err != nil {
		return fmt.Errorf("failed to write storage hash to database: %s", err)
	}

	// write best block hash to state database
	if err := StoreBestBlockHash(db, header.Hash()); err != nil {
		return fmt.Errorf("failed to write best block hash to database: %s", err)
	}

	// write genesis data to state database
	if err := StoreGenesisData(db, data); err != nil {
		return fmt.Errorf("failed to write genesis data to database: %s", err)
	}

	return nil
}

// Start initializes the Storage database and the Block database.
func (s *Service) Start() error {
	if !s.isMemDB && (s.Storage != nil || s.Block != nil || s.Epoch != nil) {
		return nil
	}

	db := s.db
	if !s.isMemDB {
		basepath, err := filepath.Abs(s.dbPath)
		if err != nil {
			return err
		}

		cfg := &chaindb.Config{
			DataDir: basepath,
		}

		// initialize database
		db, err = chaindb.NewBadgerDB(cfg)
		if err != nil {
			return err
		}

		s.db = db
	}

	// retrieve latest header
	bestHash, err := LoadBestBlockHash(db)
	if err != nil {
		return fmt.Errorf("failed to get best block hash: %w", err)
	}

	logger.Trace("start", "best block hash", bestHash)

	// load blocktree
	bt := blocktree.NewEmptyBlockTree(db)
	if err = bt.Load(); err != nil {
		return fmt.Errorf("failed to load blocktree: %w", err)
	}

	// create block state
	s.Block, err = NewBlockState(db, bt)
	if err != nil {
		return fmt.Errorf("failed to create block state: %w", err)
	}

	// create storage state
	s.Storage, err = NewStorageState(db, s.Block, trie.NewEmptyTrie())
	if err != nil {
		return fmt.Errorf("failed to create storage state: %w", err)
	}

	stateRoot, err := LoadLatestStorageHash(s.db)
	if err != nil {
		return fmt.Errorf("cannot load latest storage root: %w", err)
	}

	logger.Debug("start", "latest state root", stateRoot)

	// load current storage state
	_, err = s.Storage.LoadFromDB(stateRoot)
	if err != nil {
		return fmt.Errorf("failed to load storage trie from database: %w", err)
	}

	// create transaction queue
	s.Transaction = NewTransactionState()

	// create epoch state
	s.Epoch, err = NewEpochState(db)
	if err != nil {
		return fmt.Errorf("failed to create epoch state: %w", err)
	}

	num, _ := s.Block.BestBlockNumber()
	logger.Info("created state service", "head", s.Block.BestBlockHash(), "highest number", num)
	// Start background goroutine to GC pruned keys.
	go s.Storage.pruneStorage(s.closeCh)
	return nil
}

// Stop closes each state database
func (s *Service) Stop() error {
	head, err := s.Block.BestBlockStateRoot()
	if err != nil {
		return err
	}

	s.Storage.lock.RLock()
	t := s.Storage.tries[head]
	s.Storage.lock.RUnlock()

	if t == nil {
		return errTrieDoesNotExist(head)
	}

	if err = StoreLatestStorageHash(s.db, head); err != nil {
		return err
	}

	logger.Debug("storing latest storage trie", "root", head)

	if err = StoreTrie(s.Storage.db, t); err != nil {
		return err
	}

	if err = s.Block.bt.Store(); err != nil {
		return err
	}

	hash := s.Block.BestBlockHash()
	if err = StoreBestBlockHash(s.db, hash); err != nil {
		return err
	}

	thash, err := t.Hash()
	if err != nil {
		return err
	}
	close(s.closeCh)

	logger.Debug("stop", "best block hash", hash, "latest state root", thash)

	if err = s.db.Flush(); err != nil {
		return err
	}

	return s.db.Close()
}
