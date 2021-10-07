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
	"bytes"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/ChainSafe/chaindb"
	log "github.com/ChainSafe/log15"
)

const readyPoolTransactionsMetrics = "gossamer/ready/pool/transaction/metrics"
const readyPriorityQueueTransactions = "gossamer/ready/queue/transaction/metrics"

var logger = log.New("pkg", "state")

// Service is the struct that holds storage, block and network states
type Service struct {
	dbPath      string
	logLvl      log.Lvl
	db          chaindb.Database
	isMemDB     bool // set to true if using an in-memory database; only used for testing.
	Base        *BaseState
	Storage     *StorageState
	Block       *BlockState
	Transaction *TransactionState
	Epoch       *EpochState
	Grandpa     *GrandpaState
	closeCh     chan interface{}

	// Below are for testing only.
	BabeThresholdNumerator   uint64
	BabeThresholdDenominator uint64

	// Below are for state trie online pruner
	PrunerCfg pruner.Config
}

// Config is the default configuration used by state service.
type Config struct {
	Path      string
	LogLevel  log.Lvl
	PrunerCfg pruner.Config
}

// NewService create a new instance of Service
func NewService(config Config) *Service {
	handler := log.StreamHandler(os.Stdout, log.TerminalFormat())
	handler = log.CallerFileHandler(handler)
	logger.SetHandler(log.LvlFilterHandler(config.LogLevel, handler))

	return &Service{
		dbPath:    config.Path,
		logLvl:    config.LogLevel,
		db:        nil,
		isMemDB:   false,
		Storage:   nil,
		Block:     nil,
		closeCh:   make(chan interface{}),
		PrunerCfg: config.PrunerCfg,
	}
}

// UseMemDB tells the service to use an in-memory key-value store instead of a persistent database.
// This should be called after NewService, and before Initialise.
// This should only be used for testing.
func (s *Service) UseMemDB() {
	s.isMemDB = true
}

// DB returns the Service's database
func (s *Service) DB() chaindb.Database {
	return s.db
}

// Start initialises the Storage database and the Block database.
func (s *Service) Start() error {
	if !s.isMemDB && (s.Storage != nil || s.Block != nil || s.Epoch != nil || s.Grandpa != nil) {
		return nil
	}

	db := s.db

	if !s.isMemDB {
		basepath, err := filepath.Abs(s.dbPath)
		if err != nil {
			return err
		}

		// initialise database
		db, err = utils.SetupDatabase(basepath, false)
		if err != nil {
			return err
		}

		s.db = db
		s.Base = NewBaseState(db)
	}

	// retrieve latest header
	bestHash, err := s.Base.LoadBestBlockHash()
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

	// if blocktree head isn't "best hash", then the node shutdown abnormally.
	// restore state from last finalised hash.
	btHead := bt.DeepestBlockHash()
	if !bytes.Equal(btHead[:], bestHash[:]) {
		logger.Info("detected abnormal node shutdown, restoring from last finalised block")

		lastFinalised, err := s.Block.GetHighestFinalisedHeader() //nolint
		if err != nil {
			return fmt.Errorf("failed to get latest finalised block: %w", err)
		}

		s.Block.bt = blocktree.NewBlockTreeFromRoot(lastFinalised, db)
	}

	pr, err := s.Base.loadPruningData()
	if err != nil {
		return err
	}

	// create storage state
	s.Storage, err = NewStorageState(db, s.Block, trie.NewEmptyTrie(), pr)
	if err != nil {
		return fmt.Errorf("failed to create storage state: %w", err)
	}

	stateRoot, err := s.Base.LoadLatestStorageHash()
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
	s.Epoch, err = NewEpochState(db, s.Block)
	if err != nil {
		return fmt.Errorf("failed to create epoch state: %w", err)
	}

	s.Grandpa, err = NewGrandpaState(db)
	if err != nil {
		return fmt.Errorf("failed to create grandpa state: %w", err)
	}

	num, _ := s.Block.BestBlockNumber()
	logger.Info("created state service", "head", s.Block.BestBlockHash(), "highest number", num)

	// Start background goroutine to GC pruned keys.
	go s.Storage.pruneStorage(s.closeCh)
	return nil
}

// Rewind rewinds the chain to the given block number.
// If the given number of blocks is greater than the chain height, it will rewind to genesis.
func (s *Service) Rewind(toBlock int64) error {
	num, _ := s.Block.BestBlockNumber()
	if toBlock > num.Int64() {
		return fmt.Errorf("cannot rewind, given height is higher than our current height")
	}

	logger.Info("rewinding state...", "current height", num, "desired height", toBlock)

	root, err := s.Block.GetBlockByNumber(big.NewInt(toBlock))
	if err != nil {
		return err
	}

	s.Block.bt = blocktree.NewBlockTreeFromRoot(&root.Header, s.db)
	newHead := s.Block.BestBlockHash()

	header, _ := s.Block.BestBlockHeader()
	logger.Info("rewinding state...", "new height", header.Number, "best block hash", newHead)

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
	// for best block -> best finalised before that block
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

	return s.Base.StoreBestBlockHash(newHead)
}

// Stop closes each state database
func (s *Service) Stop() error {
	head, err := s.Block.BestBlockStateRoot()
	if err != nil {
		return err
	}

	st, has := s.Storage.tries.Load(head)
	if !has {
		return errTrieDoesNotExist(head)
	}

	t := st.(*trie.Trie)

	if err = s.Base.StoreLatestStorageHash(head); err != nil {
		return err
	}

	logger.Debug("storing latest storage trie", "root", head)

	if err = t.Store(s.Storage.db); err != nil {
		return err
	}

	if err = s.Block.bt.Store(); err != nil {
		return err
	}

	hash := s.Block.BestBlockHash()
	if err = s.Base.StoreBestBlockHash(hash); err != nil {
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

// Import imports the given state corresponding to the given header and sets the head of the chain
// to it. Additionally, it uses the first slot to correctly set the epoch number of the block.
func (s *Service) Import(header *types.Header, t *trie.Trie, firstSlot uint64) error {
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
	logger.Debug("skip BABE verification up to epoch", "epoch", skipTo)

	if err := epoch.SetCurrentEpoch(blockEpoch); err != nil {
		return err
	}

	root := t.MustHash()
	if root != header.StateRoot {
		return fmt.Errorf("trie state root does not equal header state root")
	}

	if err := s.Base.StoreLatestStorageHash(root); err != nil {
		return err
	}

	logger.Info("importing storage trie...", "basepath", s.dbPath, "root", root)

	if err := t.Store(storage.db); err != nil {
		return err
	}

	bt := blocktree.NewBlockTreeFromRoot(header, s.db)
	if err := bt.Store(); err != nil {
		return err
	}

	if err := s.Base.StoreBestBlockHash(header.Hash()); err != nil {
		return err
	}

	if err := block.SetHeader(header); err != nil {
		return err
	}

	logger.Debug("Import", "best block hash", header.Hash(), "latest state root", root)
	if err := s.db.Flush(); err != nil {
		return err
	}

	logger.Info("finished state import")
	if s.isMemDB {
		return nil
	}

	return s.db.Close()
}

// CollectGauge exports 2 metrics related to valid transaction pool and queue
func (s *Service) CollectGauge() map[string]int64 {
	return map[string]int64{
		readyPoolTransactionsMetrics:   int64(s.Transaction.pool.Len()),
		readyPriorityQueueTransactions: int64(s.Transaction.queue.Len()),
	}
}
