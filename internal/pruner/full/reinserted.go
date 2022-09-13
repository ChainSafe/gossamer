// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package full

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

func (p *Pruner) handleInsertedKeys(insertedNodeHashes map[common.Hash]struct{},
	blockNumber uint32, blockHash common.Hash, journalDBBatch Putter) (err error) {
	for insertedNodeHash := range insertedNodeHashes {
		err = p.handleInsertedKey(insertedNodeHash, blockNumber, blockHash, journalDBBatch)
		if err != nil {
			return fmt.Errorf("handling inserted key %s: %w",
				insertedNodeHash, err)
		}
	}

	return nil
}

func (p *Pruner) handleInsertedKey(insertedNodeHash common.Hash, blockNumber uint32,
	blockHash common.Hash, journalDBBatch Putter) (err error) {
	// Try to find if the node hash was deleted in another block before
	// since we no longer want to prune it, as it was re-inserted.
	deletedNodeHashKey := makeDeletedKey(insertedNodeHash)
	encodedJournalKeysDeletedAt, err := p.journalDatabase.Get(deletedNodeHashKey)
	nodeHashDeletedInAnotherBlock := !errors.Is(err, chaindb.ErrKeyNotFound)
	if !nodeHashDeletedInAnotherBlock {
		return nil
	} else if err != nil {
		return fmt.Errorf("getting journal keys for node hash from journal database: %w", err)
	}

	var journalKeysDeletedAt []journalKey
	err = scale.Unmarshal(encodedJournalKeysDeletedAt, &journalKeysDeletedAt)
	if err != nil {
		return fmt.Errorf("decoding journal keys: %w", err)
	}

	for _, journalKeyDeletedAt := range journalKeysDeletedAt {
		deletedInUncleBlock := journalKeyDeletedAt.BlockNumber >= blockNumber
		if deletedInUncleBlock {
			// do not remove the deleted node hash from the uncle block journal data
			continue
		}

		isDescendant, err := p.blockState.IsDescendantOf(journalKeyDeletedAt.BlockHash, blockHash)
		if err != nil {
			return fmt.Errorf("checking if block %s is descendant of block %s: %w",
				journalKeyDeletedAt.BlockHash, blockHash, err)
		}
		if !isDescendant {
			// do not remove the deleted node hash from the non-parent block journal data
			continue
		}

		// Remove node hash from the deleted node hashes of the ancestor block it was deleted in.
		err = handleReInsertedKey(insertedNodeHash, journalKeyDeletedAt, p.journalDatabase, journalDBBatch)
		if err != nil {
			return fmt.Errorf("handling re-inserted key %s: %w", insertedNodeHash, err)
		}
	}

	return nil
}

func handleReInsertedKey(reInsertedNodeHash common.Hash, journalKeyDeletedAt journalKey,
	journalDatabase Getter, journalDBBatch Putter) (err error) {
	encodedJournalKeyDeletedAt, err := scale.Marshal(journalKeyDeletedAt)
	if err != nil {
		return fmt.Errorf("encoding journal key: %w", err)
	}

	encodedDeletedNodeHashes, err := journalDatabase.Get(encodedJournalKeyDeletedAt)
	if err != nil {
		return fmt.Errorf("getting deleted node hashes from journal database: %w", err)
	}

	var deletedNodeHashes []common.Hash
	err = scale.Unmarshal(encodedDeletedNodeHashes, &deletedNodeHashes)
	if err != nil {
		return fmt.Errorf("decoding deleted node hashes: %w", err)
	}
	for i, deletedNodeHash := range deletedNodeHashes {
		if deletedNodeHash != reInsertedNodeHash {
			continue
		}
		lastIndex := len(deletedNodeHashes) - 1
		deletedNodeHashes[lastIndex], deletedNodeHashes[i] =
			deletedNodeHashes[i], deletedNodeHashes[lastIndex]
		deletedNodeHashes = deletedNodeHashes[:lastIndex]
		break
	}

	encodedDeletedNodeHashes, err = scale.Marshal(deletedNodeHashes)
	if err != nil {
		return fmt.Errorf("encoding updated deleted node hashes: %w", err)
	}

	err = journalDBBatch.Put(encodedJournalKeyDeletedAt, encodedDeletedNodeHashes)
	if err != nil {
		return fmt.Errorf("putting updated deleted node hashes in journal database batch: %w", err)
	}

	return nil
}
