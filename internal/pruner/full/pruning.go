// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package full

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// It prunes all block numbers falling off the window of block numbers to keep,
// before inserting the new record. It is thread safe to call.
func (p *Pruner) Prune(finalisedBlockHash common.Hash,
	prunedBlockHashes []common.Hash) (err error) {
	// Do not prune any key inserted by a pruned block from an abandoned fork
	// if it is still needed on the canonical chain.
	keysInserted

	// Prune all journal records for the pruned blocks from an abandoned fork.

	// Prune deleted trie nodes outside the window of block numbers to keep,
	// and no longer needed in the blocks currently in this window.
	// Records in the journal database are also pruned at the same time.
	journalBatch := p.journalDatabase.NewBatch()
	err = p.pruneAll(journalBatch)
	if err != nil {
		return fmt.Errorf("pruning storage database: %w", err)
	}

	return nil
}

func (p *Pruner) pruneAll(journalDBBatch PutDeleter) (err error) {
	if p.highestBlockNumber-p.nextBlockNumberToPrune < p.retainBlocks {
		return nil
	}

	storageBatch := p.storageDatabase.NewBatch()
	blockNumberToPrune := p.nextBlockNumberToPrune
	for p.highestBlockNumber-blockNumberToPrune >= p.retainBlocks {
		err := prune(blockNumberToPrune, p.journalDatabase, journalDBBatch, storageBatch)
		if err != nil {
			storageBatch.Reset()
			return fmt.Errorf("pruning block number %d: %w", blockNumberToPrune, err)
		}
		blockNumberToPrune++
	}

	var lastBlockNumberPruned uint32
	if blockNumberToPrune > 0 {
		lastBlockNumberPruned = blockNumberToPrune - 1
	}

	err = storeBlockNumberAtKey(journalDBBatch, []byte(lastPrunedKey), lastBlockNumberPruned)
	if err != nil {
		storageBatch.Reset()
		return fmt.Errorf("writing last pruned block number to journal database batch: %w", err)
	}

	err = storageBatch.Flush()
	if err != nil {
		return fmt.Errorf("flushing storage database batch: %w", err)
	}

	p.logger.Debugf("pruned block numbers [%d..%d]", p.nextBlockNumberToPrune, lastBlockNumberPruned)
	p.nextBlockNumberToPrune = blockNumberToPrune

	return nil
}

func prune(blockNumberToPrune uint32, journalDB Getter, journalDBBatch Deleter,
	storageBatch Deleter) (err error) {
	if blockNumberToPrune == 0 {
		// There is no deletion in the first block, so nothing can be pruned.
		return nil
	}

	blockHashes, err := loadBlockHashes(blockNumberToPrune, journalDB)
	if err != nil {
		return fmt.Errorf("loading block hashes for block number to prune: %w", err)
	}

	err = pruneStorage(blockNumberToPrune, blockHashes,
		journalDB, storageBatch)
	if err != nil {
		return fmt.Errorf("pruning storage: %w", err)
	}

	err = pruneJournal(blockNumberToPrune, blockHashes,
		journalDB, journalDBBatch)
	if err != nil {
		return fmt.Errorf("pruning journal: %w", err)
	}

	return nil
}

func pruneStorage(blockNumber uint32, blockHashes []common.Hash,
	journalDB Getter, batch Deleter) (err error) {
	for _, blockHash := range blockHashes {
		deletedNodeHashes, err := getDeletedNodeHashes(journalDB, blockNumber, blockHash)
		if err != nil {
			return fmt.Errorf("getting deleted node hashes: %w", err)
		}

		for _, deletedNodeHash := range deletedNodeHashes {
			err = batch.Del(deletedNodeHash.ToBytes())
			if err != nil {
				return fmt.Errorf("deleting key from batch: %w", err)
			}
		}
	}
	return nil
}

func pruneJournal(blockNumber uint32, blockHashes []common.Hash,
	journalDatabase Getter, batch Deleter) (err error) {
	err = pruneBlockHashes(blockNumber, batch)
	if err != nil {
		return fmt.Errorf("pruning block hashes: %w", err)
	}

	for _, blockHash := range blockHashes {
		key := journalKey{
			BlockNumber: blockNumber,
			BlockHash:   blockHash,
		}
		encodedKey := scale.MustMarshal(key)
		encodedDeletedNodeHashes, err := journalDatabase.Get(encodedKey)
		if err != nil {
			return fmt.Errorf("getting deleted node hashes from database: %w", err)
		}

		var deletedNodeHashes []common.Hash
		err = scale.Unmarshal(encodedDeletedNodeHashes, &deletedNodeHashes)
		if err != nil {
			return fmt.Errorf("scale decoding deleted node hashes: %w", err)
		}

		for _, deletedNodeHash := range deletedNodeHashes {
			deletedJournalKey := makeDeletedKey(deletedNodeHash)
			err = batch.Del(deletedJournalKey)
			if err != nil {
				return fmt.Errorf("deleting deleted node hash key from batch: %w", err)
			}
		}

		err = batch.Del(encodedKey)
		if err != nil {
			return fmt.Errorf("deleting journal key from batch: %w", err)
		}
	}
	return nil
}

func pruneBlockHashes(blockNumber uint32, batch Deleter) (err error) {
	keyString := blockNumberToHashPrefix + fmt.Sprint(blockNumber)
	key := []byte(keyString)
	err = batch.Del(key)
	if err != nil {
		return fmt.Errorf("deleting block hashes for block number %d from database: %w",
			blockNumber, err)
	}
	return nil
}
