// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/ChainSafe/gossamer/internal/database/badger"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

// OfflinePruner is a tool to prune the stale state with the help of
// bloom filter, The workflow of Pruner is very simple:
// - iterate the storage state, reconstruct the relevant state tries
// - iterate the database, stream all the targeted keys to new DB
type OfflinePruner struct {
	inputDB        *badger.Database
	storageState   *StorageState
	blockState     *BlockState
	filterDatabase *badger.Database
	bestBlockHash  common.Hash
	retainBlockNum uint32

	inputDBPath string
}

// NewOfflinePruner creates an instance of OfflinePruner.
func NewOfflinePruner(inputDBPath string,
	retainBlockNum uint32) (pruner *OfflinePruner, err error) {
	var settings badger.Settings
	settings.WithPath(inputDBPath)
	db, err := badger.New(settings)
	if err != nil {
		return nil, fmt.Errorf("creating badger database: %w", err)
	}

	tries := NewTries()
	tries.SetEmptyTrie()

	// create blockState state
	// NewBlockState on pruner execution does not use telemetry
	blockStateDB := db.NewTable(blockPrefix)
	baseState := NewBaseState(db)
	blockState, err := NewBlockState(blockStateDB, baseState, tries, nil)
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

	var filterDatabaseSettings badger.Settings
	filterDatabaseSettings.WithPath(filterDatabaseDir)
	filterDatabase, err := badger.New(filterDatabaseSettings)
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
		err = p.filterDatabase.Set(key.ToBytes(), nil)
		if err != nil {
			return err
		}
	}

	logger.Infof("Total keys added in filter database: %d", len(nodeHashes))
	return nil
}

// Prune starts streaming the data from input db to the pruned db.
func (p *OfflinePruner) Prune() error {
	var settings badger.Settings
	settings.WithPath(p.inputDBPath)
	inputDB, err := badger.New(settings)
	if err != nil {
		return fmt.Errorf("opening input database: %w", err)
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

	writeBatch := inputDB.NewWriteBatch()
	storagePrefixBytes := []byte(storagePrefix)
	err = inputDB.Stream(context.TODO(), storagePrefixBytes,
		func(key []byte) bool {
			// Storage keys not found in filter database are deleted.
			nodeHash := bytes.TrimPrefix(key, storagePrefixBytes)
			_, err := p.filterDatabase.Get(nodeHash)
			return err == nil
		},
		func(key, _ []byte) error {
			return writeBatch.Delete(key)
		},
	)
	if err != nil {
		return fmt.Errorf("streaming database: %w", err)
	}

	err = writeBatch.Flush()
	if err != nil {
		return fmt.Errorf("flushing write batch: %w", err)
	}

	return nil
}
