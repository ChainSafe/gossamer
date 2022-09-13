// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package full

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type journalKey struct {
	BlockNumber uint32
	BlockHash   common.Hash
}

func storeDeletedNodeHashes(journalDatabase Getter, batch Putter,
	blockNumber uint32, blockHash common.Hash,
	deletedNodeHashes map[common.Hash]struct{}) (err error) {
	key := journalKey{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}

	deletedNodeHashesSlice := make([]common.Hash, 0, len(deletedNodeHashes))
	for deletedNodeHash := range deletedNodeHashes {
		deletedNodeHashesSlice = append(deletedNodeHashesSlice, deletedNodeHash)

		// We store each block hash + block number for each deleted node hash
		// so a node hash can quickly be checked for from the journal database
		// when running `handleInsertedKey`.
		databaseKey := makeDeletedKey(deletedNodeHash)

		encodedJournalKeys, err := journalDatabase.Get(databaseKey)
		if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
			return fmt.Errorf("getting journal keys for deleted node hash "+
				"from journal database: %w", err)
		}

		var keys []journalKey
		if len(encodedJournalKeys) > 0 {
			// one or more other blocks deleted the same node hash the current
			// block deleted as well.
			err = scale.Unmarshal(encodedJournalKeys, &keys)
			if err != nil {
				return fmt.Errorf("scale decoding journal keys for deleted node hash "+
					"from journal database: %w", err)
			}
		}
		keys = append(keys, key)

		encodedKeys, err := scale.Marshal(keys)
		if err != nil {
			return fmt.Errorf("scale encoding journal keys: %w", err)
		}

		err = batch.Put(databaseKey, encodedKeys)
		if err != nil {
			return fmt.Errorf("putting journal keys in database batch: %w", err)
		}
	}

	// Sort the deleted node hashes to have a deterministic encoding for tests
	sort.Slice(deletedNodeHashesSlice, func(i, j int) bool {
		return bytes.Compare(deletedNodeHashesSlice[i][:], deletedNodeHashesSlice[j][:]) < 0
	})
	encodedDeletedNodeHashes, err := scale.Marshal(deletedNodeHashesSlice)
	if err != nil {
		return fmt.Errorf("scale encoding deleted node hashes: %w", err)
	}

	// We store the deleted node hashes in the journal database
	// at the key (block hash + block number)
	encodedKey, err := scale.Marshal(key)
	if err != nil {
		return fmt.Errorf("scale encoding journal key: %w", err)
	}
	err = batch.Put(encodedKey, encodedDeletedNodeHashes)
	if err != nil {
		return fmt.Errorf("putting deleted node hashes in database batch: %w", err)
	}

	return nil
}

func getDeletedNodeHashes(database Getter, blockNumber uint32,
	blockHash common.Hash) (deletedNodeHashes []common.Hash, err error) {
	key := journalKey{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
	encodedKey, err := scale.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("scale encoding key: %w", err)
	}

	encodedNodeHashes, err := database.Get(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("getting from database: %w", err)
	}

	err = scale.Unmarshal(encodedNodeHashes, &deletedNodeHashes)
	if err != nil {
		return nil, fmt.Errorf("scale decoding deleted node hashes: %w", err)
	}

	return deletedNodeHashes, nil
}

func storeBlockNumberAtKey(batch Putter, key []byte, blockNumber uint32) error {
	encodedBlockNumber, err := scale.Marshal(blockNumber)
	if err != nil {
		return fmt.Errorf("encoding block number: %w", err)
	}

	err = batch.Put(key, encodedBlockNumber)
	if err != nil {
		return fmt.Errorf("putting block number %d: %w", blockNumber, err)
	}

	return nil
}

// getBlockNumberFromKey obtains the block number from the database at the given key.
// If the key is not found, the block number `0` is returned without error.
func getBlockNumberFromKey(database Getter, key []byte) (blockNumber uint32, err error) {
	encodedBlockNumber, err := database.Get(key)
	if err != nil {
		return 0, fmt.Errorf("getting block number from database: %w", err)
	}

	err = scale.Unmarshal(encodedBlockNumber, &blockNumber)
	if err != nil {
		return 0, fmt.Errorf("decoding block number: %w", err)
	}

	return blockNumber, nil
}

func loadBlockHashes(blockNumber uint32, journalDB Getter) (blockHashes []common.Hash, err error) {
	keyString := blockNumberToHashPrefix + fmt.Sprint(blockNumber)
	key := []byte(keyString)
	encodedBlockHashes, err := journalDB.Get(key)
	if err != nil {
		return nil, fmt.Errorf("getting block hashes for block number %d: %w", blockNumber, err)
	}

	// Note the reason we don't use scale is to append a hash to existing hashes without
	// having to scale decode and scale encode.
	numberOfBlockHashes := len(encodedBlockHashes) / common.HashLength
	blockHashes = make([]common.Hash, numberOfBlockHashes)
	for i := 0; i < numberOfBlockHashes; i++ {
		startIndex := i * common.HashLength
		endIndex := startIndex + common.HashLength
		blockHashes[i] = common.NewHash(encodedBlockHashes[startIndex:endIndex])
	}

	return blockHashes, nil
}

func appendBlockHash(blockNumber uint32, blockHash common.Hash, journalDB Getter,
	batch Putter) (err error) {
	keyString := blockNumberToHashPrefix + fmt.Sprint(blockNumber)
	key := []byte(keyString)
	encodedBlockHashes, err := journalDB.Get(key)
	if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
		return fmt.Errorf("getting block hashes for block number %d: %w", blockNumber, err)
	}

	encodedBlockHashes = append(encodedBlockHashes, blockHash.ToBytes()...)

	err = batch.Put(key, encodedBlockHashes)
	if err != nil {
		return fmt.Errorf("putting block hashes for block number %d: %w", blockNumber, err)
	}

	return nil
}

func makeDeletedKey(hash common.Hash) (key []byte) {
	key = make([]byte, 0, len(deletedNodeHashKeyPrefix)+common.HashLength)
	key = append(key, []byte(deletedNodeHashKeyPrefix)...)
	key = append(key, hash.ToBytes()...)
	return key
}
