// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package full

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
)

// Pruner prunes unneeded database keys for blocks older than the current
// block minus the number of blocks to retain specified.
// It keeps track through a journal database of the trie changes for every block
// in order to determine what can be pruned and what should be kept.
type Pruner struct {
	// Configuration
	retainBlocks uint32

	// Dependency injected
	logger          Logger
	storageDatabase ChainDBNewBatcher
	journalDatabase JournalDatabase
	blockState      BlockState

	// Internal state
	// nextBlockNumberToPrune is the next block number to prune.
	// It is updated on disk but cached in memory as this field.
	nextBlockNumberToPrune uint32
	// highestBlockNumber is the highest block number stored in the journal.
	// It is updated on disk but cached in memory as this field.
	highestBlockNumber uint32
	// mutex protects the in memory data members since RecordAndPrune
	// is called in lib/babe `epochHandler`'s `run` method which is run
	// in its own goroutine.
	mutex sync.RWMutex
}

// New creates a full node pruner.
func New(journalDB JournalDatabase, storageDB ChainDBNewBatcher, retainBlocks uint32,
	blockState BlockState, logger Logger) (pruner *Pruner, err error) {
	highestBlockNumber, err := getBlockNumberFromKey(journalDB, []byte(highestBlockNumberKey))
	if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
		return nil, fmt.Errorf("getting highest block number: %w", err)
	}
	logger.Debugf("highest block number stored in journal: %d", highestBlockNumber)

	var nextBlockNumberToPrune uint32
	lastPrunedBlockNumber, err := getBlockNumberFromKey(journalDB, []byte(lastPrunedKey))
	if errors.Is(err, chaindb.ErrKeyNotFound) {
		nextBlockNumberToPrune = 0
	} else if err != nil {
		return nil, fmt.Errorf("getting last pruned block number: %w", err)
	} else {
		// if the error is database.ErrKeyNotFound it means we have not pruned
		// any block number yet, so leave the next block number to prune as 0.
		nextBlockNumberToPrune = lastPrunedBlockNumber + 1
	}
	logger.Debugf("next block number to prune: %d", nextBlockNumberToPrune)

	pruner = &Pruner{
		storageDatabase:        storageDB,
		journalDatabase:        journalDB,
		blockState:             blockState,
		retainBlocks:           retainBlocks,
		nextBlockNumberToPrune: nextBlockNumberToPrune,
		highestBlockNumber:     highestBlockNumber,
		logger:                 logger,
	}

	// Prune all block numbers necessary, if for example the
	// user lowers the retainBlocks parameter.
	journalDBBatch := journalDB.NewBatch()
	err = pruner.pruneAll(journalDBBatch)
	if err != nil {
		journalDBBatch.Reset()
		return nil, fmt.Errorf("pruning: %w", err)
	}
	err = journalDBBatch.Flush()
	if err != nil {
		return nil, fmt.Errorf("flushing journal database batch: %w", err)
	}

	return pruner, nil
}

// RecordAndPrune stores the trie deltas impacting the storage database for a particular
// block hash. It prunes all block numbers falling off the window of block numbers to keep,
// before inserting the new record. It is thread safe to call.
func (p *Pruner) RecordAndPrune(deletedNodeHashes, insertedNodeHashes map[common.Hash]struct{},
	blockHash common.Hash, blockNumber uint32) (err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	blockIsAlreadyPruned := blockNumber < p.nextBlockNumberToPrune
	if blockIsAlreadyPruned {
		panic(fmt.Sprintf("block number %d is already pruned, last block number pruned was %d",
			blockNumber, p.nextBlockNumberToPrune))
	}

	if blockNumber == 0 {
		// The genesis block has no node hash deletion, and no re-inserted node hashes.
		// There is no node hashes to be pruned either.
		return nil
	}

	// Delist re-inserted keys from being pruned.
	// WARNING: this must be before the pruning to avoid
	// pruning still needed database keys.
	journalDBBatch := p.journalDatabase.NewBatch()
	err = p.handleInsertedKeys(insertedNodeHashes, blockNumber,
		blockHash, journalDBBatch)
	if err != nil {
		journalDBBatch.Reset()
		return fmt.Errorf("handling inserted keys: %w", err)
	}

	err = journalDBBatch.Flush()
	if err != nil {
		return fmt.Errorf("flushing re-inserted keys updates to journal database: %w", err)
	}

	journalDBBatch = p.journalDatabase.NewBatch()

	// Update highest block number in memory and on disk so `pruneAll` can use it,
	// prune and flush the deletions in the journal and storage databases.
	if blockNumber > p.highestBlockNumber {
		p.highestBlockNumber = blockNumber
		err = storeBlockNumberAtKey(journalDBBatch, []byte(highestBlockNumberKey), blockNumber)
		if err != nil {
			journalDBBatch.Reset()
			return fmt.Errorf("storing highest block number in journal database: %w", err)
		}
	}

	// Prune before inserting new journal data
	err = p.pruneAll(journalDBBatch)
	if err != nil {
		journalDBBatch.Reset()
		return fmt.Errorf("pruning database: %w", err)
	}

	// Note we store block number <-> block hashes in the database
	// so we can pick up the block hashes after a program restart
	// using the stored last pruned block number and stored highest
	// block number encountered.
	err = appendBlockHash(blockNumber, blockHash, p.journalDatabase, journalDBBatch)
	if err != nil {
		journalDBBatch.Reset()
		return fmt.Errorf("recording block hash in journal database: %w", err)
	}

	err = storeDeletedNodeHashes(p.journalDatabase, journalDBBatch, blockNumber,
		blockHash, deletedNodeHashes)
	if err != nil {
		journalDBBatch.Reset()
		return fmt.Errorf("storing deleted node hashes for block number %d: %w", blockNumber, err)
	}

	err = journalDBBatch.Flush()
	if err != nil {
		return fmt.Errorf("flushing journal database batch: %w", err)
	}

	p.logger.Debugf("journal data stored for block number %d and block hash %s", blockNumber, blockHash.Short())
	return nil
}
