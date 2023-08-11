// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
)

// OfflinePruner is a tool to prune the stale state with the help of
// bloom filter, The workflow of Pruner is very simple:
// - iterate the storage state, reconstruct the relevant state tries
// - iterate the database, stream all the targeted keys to new DB
type OfflinePruner struct {
	inputDB        database.Database
	storageState   *StorageState
	blockState     *BlockState
	filterDatabase database.Database
	bestBlockHash  common.Hash
	retainBlockNum uint32

	inputDBPath string
}

// NewOfflinePruner creates an instance of OfflinePruner.
func NewOfflinePruner(inputDBPath string,
	retainBlockNum uint32) (pruner *OfflinePruner, err error) {
	db, err := utils.LoadChainDB(inputDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load DB %w", err)
	}

	tries := NewTries()
	tries.SetEmptyTrie()

	// create blockState state
	// NewBlockState on pruner execution does not use telemetry
	blockState, err := NewBlockState(db, tries, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create block state: %w", err)
	}

	bestHash, err := blockState.GetHighestFinalisedHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get best finalised hash: %w", err)
	}

	// Create temporary filter database to store database keys only
	filterDatabaseDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("creating filter database temp directory: %w", err)
	}
	defer func() {
		removeErr := os.RemoveAll(filterDatabaseDir)
		if err == nil {
			err = removeErr
		}
	}()

	filterDatabase, err := database.NewPebble(filterDatabaseDir, false)
	if err != nil {
		return nil, fmt.Errorf("creating badger filter database: %w", err)
	}

	// load storage state
	storageState, err := NewStorageState(db, blockState, tries)
	if err != nil {
		return nil, fmt.Errorf("failed to create new storage state %w", err)
	}

	return &OfflinePruner{
		inputDB:        db,
		storageState:   storageState,
		blockState:     blockState,
		filterDatabase: filterDatabase,
		bestBlockHash:  bestHash,
		retainBlockNum: retainBlockNum,
		inputDBPath:    inputDBPath,
	}, nil
}

// SetBloomFilter loads keys with storage prefix of last `retainBlockNum` blocks into the bloom filter
func (p *OfflinePruner) SetBloomFilter() (err error) {
	defer func() {
		closeErr := p.inputDB.Close()
		switch {
		case closeErr == nil:
			return
		case err == nil:
			err = fmt.Errorf("cannot close input database: %w", closeErr)
		default:
			logger.Errorf("cannot close input database: %s", err)
		}
	}()

	finalisedHash, err := p.blockState.GetHighestFinalisedHash()
	if err != nil {
		return fmt.Errorf("failed to get highest finalised hash: %w", err)
	}

	header, err := p.blockState.GetHeader(finalisedHash)
	if err != nil {
		return fmt.Errorf("failed to get highest finalised header: %w", err)
	}

	latestBlockNum := header.Number
	nodeHashes := make(map[common.Hash]struct{})

	logger.Infof("Latest block number is %d", latestBlockNum)

	if latestBlockNum-uint(p.retainBlockNum) <= 0 {
		return fmt.Errorf("not enough block to perform pruning")
	}

	// loop from latest to last `retainBlockNum` blocks
	for blockNum := header.Number; blockNum > 0 && blockNum >= latestBlockNum-uint(p.retainBlockNum); {
		var tr *trie.Trie
		tr, err = p.storageState.LoadFromDB(header.StateRoot)
		if err != nil {
			return err
		}

		trie.PopulateNodeHashes(tr.RootNode(), nodeHashes)

		// get parent header of current block
		header, err = p.blockState.GetHeader(header.ParentHash)
		if err != nil {
			return err
		}
		blockNum = header.Number
	}

	for key := range nodeHashes {
		err = p.filterDatabase.Put(key.ToBytes(), nil)
		if err != nil {
			return err
		}
	}

	logger.Infof("Total keys added in filter database: %d", len(nodeHashes))
	return nil
}

// Prune starts streaming the data from input db to the pruned db.
func (p *OfflinePruner) Prune() error {
	inputDB, err := utils.SetupDatabase(p.inputDBPath, false)
	if err != nil {
		return fmt.Errorf("failed to load DB %w", err)
	}

	defer func() {
		closeErr := inputDB.Close()
		switch {
		case closeErr == nil:
			return
		case err == nil:
			err = fmt.Errorf("cannot close input database: %w", closeErr)
		default:
			logger.Errorf("cannot close input database: %s", err)
		}
	}()

	storagePrefixBytes := []byte(storagePrefix)
	// Ignore non-storage keys
	inputDBIter := inputDB.NewPrefixIterator(storagePrefixBytes)
	defer inputDBIter.Release()

	writeBatch := inputDB.NewBatch()

	for inputDBIter.First(); inputDBIter.Valid(); inputDBIter.Next() {
		key := inputDBIter.Key()

		// Storage keys not found in filter database are deleted.
		nodeHash := bytes.TrimPrefix(key, storagePrefixBytes)
		_, err := p.filterDatabase.Get(nodeHash)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				continue
			}

			return fmt.Errorf("checking filter database: %w", err)
		}

		err = writeBatch.Del(key)
		if err != nil {
			return fmt.Errorf("inserting in the batch delete: %w", err)
		}
	}

	err = writeBatch.Flush()
	if err != nil {
		return fmt.Errorf("flushing write batch: %w", err)
	}

	return nil
}
