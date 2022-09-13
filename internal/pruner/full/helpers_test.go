// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package full

import (
	"bytes"
	"sort"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keyValueMap is a map from stringed key to bytes value.
// It is a map to deduplicate keys.
type keyValueMap map[string][]byte

type keyValuePair struct {
	key   []byte
	value []byte
}

func (kvm keyValueMap) toComparable() (comparable []keyValuePair) {
	comparable = make([]keyValuePair, 0, len(kvm))
	for keyString, value := range kvm {
		keyValue := keyValuePair{
			key:   []byte(keyString),
			value: value,
		}
		comparable = append(comparable, keyValue)
	}

	sort.Slice(comparable, func(i, j int) bool {
		return bytes.Compare(comparable[i].key, comparable[j].key) < 0
	})

	return comparable
}

func assertDatabaseContent(t *testing.T, database any, keyValues keyValueMap) {
	t.Helper()

	implementation, ok := database.(*chaindb.BadgerDB)
	require.Truef(t, ok, "database is %T and is not a memory database", database)

	iterator := implementation.NewIterator()
	var allKeys [][]byte
	for iterator.Next() {
		allKeys = append(allKeys, iterator.Key())
	}
	storedKeyValues := make(keyValueMap, len(allKeys))
	for _, key := range allKeys {
		value, err := implementation.Get(key)
		require.NoError(t, err)
		storedKeyValues[string(key)] = value
	}

	expected := keyValues.toComparable()
	actual := storedKeyValues.toComparable()
	assert.ElementsMatch(t, expected, actual)
}

func setNodeHashesInStorageDB(t *testing.T, storageDB Putter, nodeHashes []common.Hash) {
	t.Helper()

	for _, nodeHash := range nodeHashes {
		key := nodeHash.ToBytes()
		value := []byte{0x99}
		err := storageDB.Put(key, value)
		require.NoError(t, err)
	}
}

func scaleEncodeJournalKey(blockNumber uint32, blockHash common.Hash) (encoded []byte) {
	key := journalKey{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
	return scale.MustMarshal(key)
}

func concatHashes(hashes []common.Hash) (bytes []byte) {
	bytes = make([]byte, 0, common.HashLength*len(hashes))
	for _, hash := range hashes {
		bytes = append(bytes, hash.ToBytes()...)
	}
	return bytes
}
