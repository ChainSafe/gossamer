// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package full

import (
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func Test_Pruner(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	database, err := chaindb.NewBadgerDB(&chaindb.Config{InMemory: true})
	require.NoError(t, err)
	journalDB := chaindb.NewTable(database, "journal_")
	storageDB := chaindb.NewTable(database, "storage_")
	retainBlocks := uint32(2)
	blockState := NewMockBlockState(ctrl)
	logger := NewMockLogger(ctrl)

	logger.EXPECT().Debugf("highest block number stored in journal: %d", uint32(0))
	logger.EXPECT().Debugf("next block number to prune: %d", uint32(0))
	pruner, err := New(journalDB, storageDB, retainBlocks,
		blockState, logger)
	require.NoError(t, err)

	keyValuePairs := keyValueMap{}
	assertDatabaseContent(t, database, keyValuePairs)

	// Block 0 hash 100
	setNodeHashesInStorageDB(t, storageDB, []common.Hash{{1}, {2}})
	err = pruner.RecordAndPrune(
		map[common.Hash]struct{}{},                 // first block has no deleted node hashes
		map[common.Hash]struct{}{{1}: {}, {2}: {}}, // inserted node hashes
		common.Hash{100},                           // block hash
		0,                                          // block number
	)
	require.NoError(t, err)
	keyValuePairs = keyValueMap{
		"storage_" + string(common.Hash{1}.ToBytes()): []byte{0x99},
		"storage_" + string(common.Hash{2}.ToBytes()): []byte{0x99},
	}
	assertDatabaseContent(t, database, keyValuePairs)

	// Block 1 hash 101
	setNodeHashesInStorageDB(t, storageDB, []common.Hash{{3}, {4}})
	logger.EXPECT().Debugf("journal data stored for block number %d and block hash %s",
		uint32(1), "0x65000000...00000000")
	err = pruner.RecordAndPrune(
		map[common.Hash]struct{}{{1}: {}},          // deleted node hashes
		map[common.Hash]struct{}{{3}: {}, {4}: {}}, // inserted node hashes
		common.Hash{101},                           // block hash
		1,                                          // block number
	)
	require.NoError(t, err)
	keyValuePairs = keyValueMap{
		"storage_" + string(common.Hash{1}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{2}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{3}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{4}.ToBytes()):                   []byte{0x99},
		"journal_highest_block_number":                                  scale.MustMarshal(uint32(1)),
		"journal_block_number_to_hash_1":                                common.Hash{101}.ToBytes(),
		"journal_" + string(scaleEncodeJournalKey(1, common.Hash{101})): scale.MustMarshal([]common.Hash{{1}}),
		"journal_deleted_" + string(common.Hash{1}.ToBytes()): scale.MustMarshal(
			[]journalKey{{1, common.Hash{101}}}),
	}
	assertDatabaseContent(t, database, keyValuePairs)

	// Block 1 hash 102
	setNodeHashesInStorageDB(t, storageDB, []common.Hash{{5}, {6}})
	logger.EXPECT().Debugf("journal data stored for block number %d and block hash %s",
		uint32(1), "0x66000000...00000000")
	err = pruner.RecordAndPrune(
		map[common.Hash]struct{}{{3}: {}},          // deleted node hashes
		map[common.Hash]struct{}{{5}: {}, {6}: {}}, // inserted node hashes
		common.Hash{102},                           // block hash
		1,                                          // block number
	)
	require.NoError(t, err)
	keyValuePairs = keyValueMap{
		"storage_" + string(common.Hash{1}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{2}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{3}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{4}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{5}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{6}.ToBytes()):                   []byte{0x99},
		"journal_highest_block_number":                                  scale.MustMarshal(uint32(1)),
		"journal_block_number_to_hash_1":                                concatHashes([]common.Hash{{101}, {102}}),
		"journal_" + string(scaleEncodeJournalKey(1, common.Hash{101})): scale.MustMarshal([]common.Hash{{1}}),
		"journal_" + string(scaleEncodeJournalKey(1, common.Hash{102})): scale.MustMarshal([]common.Hash{{3}}),
		"journal_deleted_" + string(common.Hash{1}.ToBytes()): scale.MustMarshal(
			[]journalKey{{1, common.Hash{101}}}),
		"journal_deleted_" + string(common.Hash{3}.ToBytes()): scale.MustMarshal(
			[]journalKey{{1, common.Hash{102}}}),
	}
	assertDatabaseContent(t, database, keyValuePairs)

	// Block 2 hash 103
	setNodeHashesInStorageDB(t, storageDB, []common.Hash{{7}, {8}})
	logger.EXPECT().Debugf("pruned block numbers [%d..%d]", uint32(0), uint32(0))
	logger.EXPECT().Debugf("journal data stored for block number %d and block hash %s",
		uint32(2), "0x67000000...00000000")
	err = pruner.RecordAndPrune(
		map[common.Hash]struct{}{{5}: {}},          // deleted node hashes
		map[common.Hash]struct{}{{7}: {}, {8}: {}}, // inserted node hashes
		common.Hash{103},                           // block hash
		2,                                          // block number
	)
	require.NoError(t, err)
	keyValuePairs = keyValueMap{
		"storage_" + string(common.Hash{1}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{2}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{3}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{4}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{5}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{6}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{7}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{8}.ToBytes()):                   []byte{0x99},
		"journal_highest_block_number":                                  scale.MustMarshal(uint32(2)),
		"journal_last_pruned":                                           scale.MustMarshal(uint32(0)),
		"journal_block_number_to_hash_1":                                concatHashes([]common.Hash{{101}, {102}}),
		"journal_block_number_to_hash_2":                                common.Hash{103}.ToBytes(),
		"journal_" + string(scaleEncodeJournalKey(1, common.Hash{101})): scale.MustMarshal([]common.Hash{{1}}),
		"journal_" + string(scaleEncodeJournalKey(1, common.Hash{102})): scale.MustMarshal([]common.Hash{{3}}),
		"journal_" + string(scaleEncodeJournalKey(2, common.Hash{103})): scale.MustMarshal([]common.Hash{{5}}),
		"journal_deleted_" + string(common.Hash{1}.ToBytes()): scale.MustMarshal(
			[]journalKey{{1, common.Hash{101}}}),
		"journal_deleted_" + string(common.Hash{3}.ToBytes()): scale.MustMarshal(
			[]journalKey{{1, common.Hash{102}}}),
		"journal_deleted_" + string(common.Hash{5}.ToBytes()): scale.MustMarshal(
			[]journalKey{{2, common.Hash{103}}}),
	}
	assertDatabaseContent(t, database, keyValuePairs)

	// Block 3 hash 104
	setNodeHashesInStorageDB(t, storageDB, []common.Hash{{9}, {10}})
	logger.EXPECT().Debugf("pruned block numbers [%d..%d]", uint32(1), uint32(1))
	logger.EXPECT().Debugf("journal data stored for block number %d and block hash %s",
		uint32(3), "0x68000000...00000000")
	err = pruner.RecordAndPrune(
		map[common.Hash]struct{}{{7}: {}},           // deleted node hashes
		map[common.Hash]struct{}{{9}: {}, {10}: {}}, // inserted node hashes
		common.Hash{104},                            // block hash
		3,                                           // block number
	)
	require.NoError(t, err)
	keyValuePairs = keyValueMap{
		"storage_" + string(common.Hash{2}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{4}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{5}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{6}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{7}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{8}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{9}.ToBytes()):                   []byte{0x99},
		"storage_" + string(common.Hash{10}.ToBytes()):                  []byte{0x99},
		"journal_highest_block_number":                                  scale.MustMarshal(uint32(3)),
		"journal_last_pruned":                                           scale.MustMarshal(uint32(1)),
		"journal_block_number_to_hash_2":                                common.Hash{103}.ToBytes(),
		"journal_block_number_to_hash_3":                                common.Hash{104}.ToBytes(),
		"journal_" + string(scaleEncodeJournalKey(2, common.Hash{103})): scale.MustMarshal([]common.Hash{{5}}),
		"journal_" + string(scaleEncodeJournalKey(3, common.Hash{104})): scale.MustMarshal([]common.Hash{{7}}),
		"journal_deleted_" + string(common.Hash{5}.ToBytes()): scale.MustMarshal(
			[]journalKey{{2, common.Hash{103}}}),
		"journal_deleted_" + string(common.Hash{7}.ToBytes()): scale.MustMarshal(
			[]journalKey{{3, common.Hash{104}}}),
	}
	assertDatabaseContent(t, database, keyValuePairs)
}
