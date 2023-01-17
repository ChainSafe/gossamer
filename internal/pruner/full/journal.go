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

type journalRecord struct {
	deletedNodeHashes  []common.Hash
	insertedNodeHashes []common.Hash
}

func storeNodeHashesDeltas(journalDatabase Getter, batch Putter,
	blockNumber uint32, blockHash common.Hash,
	deletedNodeHashes, insertedNodeHashes map[common.Hash]struct{}) (err error) {
	key := journalKey{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
	encodedKey, err := scale.Marshal(key)
	if err != nil {
		return fmt.Errorf("scale encoding journal key: %w", err)
	}

	record := journalRecord{
		deletedNodeHashes:  make([]common.Hash, 0, len(deletedNodeHashes)),
		insertedNodeHashes: make([]common.Hash, 0, len(insertedNodeHashes)),
	}
	for deletedNodeHash := range deletedNodeHashes {
		record.deletedNodeHashes = append(record.deletedNodeHashes, deletedNodeHash)
	}
	for insertedNodeHash := range insertedNodeHashes {
		record.insertedNodeHashes = append(record.insertedNodeHashes, insertedNodeHash)
	}
	// Sort the node hashes to have a deterministic encoding for tests
	sort.Slice(record.deletedNodeHashes, func(i, j int) bool {
		return bytes.Compare(record.deletedNodeHashes[i][:], record.deletedNodeHashes[j][:]) < 0
	})
	sort.Slice(record.insertedNodeHashes, func(i, j int) bool {
		return bytes.Compare(record.insertedNodeHashes[i][:], record.insertedNodeHashes[j][:]) < 0
	})
	encodedRecord, err := scale.Marshal(record)
	if err != nil {
		return fmt.Errorf("scale encoding journal record: %w", err)
	}

	err = batch.Put(encodedKey, encodedRecord)
	if err != nil {
		return fmt.Errorf("putting journal record in database batch: %w", err)
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
