// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package full

import (
	"errors"
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_Pruner_pruneAll(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		pruner                 *Pruner
		journalDatabaseBuilder func(ctrl *gomock.Controller) JournalDatabase
		storageDatabaseBuilder func(ctrl *gomock.Controller) ChainDBNewBatcher
		loggerBuilder          func(ctrl *gomock.Controller) Logger
		journalBatchBuilder    func(ctrl *gomock.Controller) PutDeleter
		errWrapped             error
		errMessage             string
		expectedPruner         *Pruner
	}{
		"not enough blocks to prune": {
			pruner: &Pruner{
				retainBlocks:           3,
				nextBlockNumberToPrune: 1,
				highestBlockNumber:     3,
			},
			expectedPruner: &Pruner{
				retainBlocks:           3,
				nextBlockNumberToPrune: 1,
				highestBlockNumber:     3,
			},
		},
		"prune block error": {
			pruner: &Pruner{
				retainBlocks:           2,
				nextBlockNumberToPrune: 1,
				highestBlockNumber:     3,
			},
			journalDatabaseBuilder: func(ctrl *gomock.Controller) JournalDatabase {
				journalDatabase := NewMockJournalDatabase(ctrl)
				journalDatabase.EXPECT().Get([]byte("block_number_to_hash_1")).Return(nil, errTest)
				return journalDatabase
			},
			storageDatabaseBuilder: func(ctrl *gomock.Controller) ChainDBNewBatcher {
				storageDatabase := NewMockChainDBNewBatcher(ctrl)
				batch := NewMockBatch(ctrl)
				storageDatabase.EXPECT().NewBatch().Return(batch)
				batch.EXPECT().Reset()
				return storageDatabase
			},
			expectedPruner: &Pruner{
				retainBlocks:           2,
				nextBlockNumberToPrune: 1,
				highestBlockNumber:     3,
			},
			errWrapped: errTest,
			errMessage: "pruning block number 1: " +
				"loading block hashes for block number to prune: " +
				"getting block hashes for block number 1: test error",
		},
		"store last block number pruned error": {
			pruner: &Pruner{
				retainBlocks:           2,
				nextBlockNumberToPrune: 1,
				highestBlockNumber:     3,
			},
			journalDatabaseBuilder: func(ctrl *gomock.Controller) JournalDatabase {
				database := NewMockJournalDatabase(ctrl)
				blockHashes := common.Hash{2}.ToBytes()
				database.EXPECT().Get([]byte("block_number_to_hash_1")).Return(blockHashes, nil)

				database.EXPECT().Get(scaleEncodeJournalKey(1, common.Hash{2})).
					Return(scale.MustMarshal([]common.Hash{{3}}), nil).
					Times(2)

				return database
			},
			storageDatabaseBuilder: func(ctrl *gomock.Controller) ChainDBNewBatcher {
				storageDatabase := NewMockChainDBNewBatcher(ctrl)
				batch := NewMockBatch(ctrl)
				storageDatabase.EXPECT().NewBatch().Return(batch)
				batch.EXPECT().Del(common.Hash{3}.ToBytes()).Return(nil)
				batch.EXPECT().Reset()
				return storageDatabase
			},
			journalBatchBuilder: func(ctrl *gomock.Controller) PutDeleter {
				batch := NewMockPutDeleter(ctrl)

				batch.EXPECT().Del([]byte("block_number_to_hash_1")).Return(nil)

				batch.EXPECT().Del(scaleEncodeJournalKey(1, common.Hash{2})).Return(nil)
				batch.EXPECT().Del(makeDeletedKey(common.Hash{3})).Return(nil)

				batch.EXPECT().Put([]byte("last_pruned"), scale.MustMarshal(uint32(1))).
					Return(errTest)
				return batch
			},
			expectedPruner: &Pruner{
				retainBlocks:           2,
				nextBlockNumberToPrune: 1,
				highestBlockNumber:     3,
			},
			errWrapped: errTest,
			errMessage: "writing last pruned block number to journal database batch: " +
				"putting block number 1: test error",
		},
		"storage batch flush error": {
			pruner: &Pruner{
				retainBlocks:           2,
				nextBlockNumberToPrune: 1,
				highestBlockNumber:     3,
			},
			journalDatabaseBuilder: func(ctrl *gomock.Controller) JournalDatabase {
				database := NewMockJournalDatabase(ctrl)
				blockHashes := common.Hash{2}.ToBytes()
				database.EXPECT().Get([]byte("block_number_to_hash_1")).Return(blockHashes, nil)
				database.EXPECT().Get(scaleEncodeJournalKey(1, common.Hash{2})).
					Return(scale.MustMarshal([]common.Hash{{3}}), nil).Times(2)

				return database
			},
			storageDatabaseBuilder: func(ctrl *gomock.Controller) ChainDBNewBatcher {
				storageDatabase := NewMockChainDBNewBatcher(ctrl)
				batch := NewMockBatch(ctrl)
				storageDatabase.EXPECT().NewBatch().Return(batch)
				batch.EXPECT().Del(common.Hash{3}.ToBytes()).Return(nil)
				batch.EXPECT().Flush().Return(errTest)
				return storageDatabase
			},
			journalBatchBuilder: func(ctrl *gomock.Controller) PutDeleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del([]byte("block_number_to_hash_1")).Return(nil)
				batch.EXPECT().Del(scaleEncodeJournalKey(1, common.Hash{2})).Return(nil)
				batch.EXPECT().Del(makeDeletedKey(common.Hash{3})).Return(nil)
				batch.EXPECT().Put([]byte("last_pruned"), scale.MustMarshal(uint32(1))).
					Return(nil)
				return batch
			},
			expectedPruner: &Pruner{
				retainBlocks:           2,
				nextBlockNumberToPrune: 1,
				highestBlockNumber:     3,
			},
			errWrapped: errTest,
			errMessage: "flushing storage database batch: test error",
		},
		"success": {
			pruner: &Pruner{
				retainBlocks:           2,
				nextBlockNumberToPrune: 1,
				highestBlockNumber:     3,
			},
			journalDatabaseBuilder: func(ctrl *gomock.Controller) JournalDatabase {
				database := NewMockJournalDatabase(ctrl)
				database.EXPECT().Get([]byte("block_number_to_hash_1")).
					Return(common.Hash{2}.ToBytes(), nil)
				database.EXPECT().Get(scaleEncodeJournalKey(1, common.Hash{2})).
					Return(scale.MustMarshal([]common.Hash{{3}}), nil).
					Times(2)
				return database
			},
			storageDatabaseBuilder: func(ctrl *gomock.Controller) ChainDBNewBatcher {
				storageDatabase := NewMockChainDBNewBatcher(ctrl)
				batch := NewMockBatch(ctrl)
				storageDatabase.EXPECT().NewBatch().Return(batch)
				batch.EXPECT().Del(common.Hash{3}.ToBytes()).Return(nil)
				batch.EXPECT().Flush().Return(nil)
				return storageDatabase
			},
			loggerBuilder: func(ctrl *gomock.Controller) Logger {
				logger := NewMockLogger(ctrl)
				logger.EXPECT().Debugf("pruned block numbers [%d..%d]", uint32(1), uint32(1))
				return logger
			},
			journalBatchBuilder: func(ctrl *gomock.Controller) PutDeleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del([]byte("block_number_to_hash_1")).Return(nil)
				batch.EXPECT().Del(scaleEncodeJournalKey(1, common.Hash{2})).Return(nil)
				batch.EXPECT().Del(makeDeletedKey(common.Hash{3})).Return(nil)
				batch.EXPECT().Put([]byte("last_pruned"), scale.MustMarshal(uint32(1))).
					Return(nil)
				return batch
			},
			expectedPruner: &Pruner{
				retainBlocks:           2,
				nextBlockNumberToPrune: 2,
				highestBlockNumber:     3,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			if testCase.journalDatabaseBuilder != nil {
				testCase.pruner.journalDatabase = testCase.journalDatabaseBuilder(ctrl)
				testCase.expectedPruner.journalDatabase = testCase.pruner.journalDatabase
			}

			if testCase.storageDatabaseBuilder != nil {
				testCase.pruner.storageDatabase = testCase.storageDatabaseBuilder(ctrl)
				testCase.expectedPruner.storageDatabase = testCase.pruner.storageDatabase
			}

			if testCase.loggerBuilder != nil {
				testCase.pruner.logger = testCase.loggerBuilder(ctrl)
				testCase.expectedPruner.logger = testCase.pruner.logger
			}

			var journalBatch PutDeleter
			if testCase.journalBatchBuilder != nil {
				journalBatch = testCase.journalBatchBuilder(ctrl)
			}

			err := testCase.pruner.pruneAll(journalBatch)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.expectedPruner, testCase.pruner)
		})
	}
}

func Test_prune(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		blockNumberToPrune  uint32
		journalDBBuilder    func(ctrl *gomock.Controller) Getter
		journalBatchBuilder func(ctrl *gomock.Controller) Deleter
		storageBatchBuilder func(ctrl *gomock.Controller) Deleter
		errWrapped          error
		errMessage          string
	}{
		"nothing to do for block number 0": {
			blockNumberToPrune:  0,
			journalDBBuilder:    func(ctrl *gomock.Controller) Getter { return nil },
			journalBatchBuilder: func(_ *gomock.Controller) Deleter { return nil },
			storageBatchBuilder: func(_ *gomock.Controller) Deleter { return nil },
		},
		"load block hashes error": {
			blockNumberToPrune: 1,
			journalDBBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get([]byte("block_number_to_hash_1")).Return(nil, errTest)
				return database
			},
			journalBatchBuilder: func(_ *gomock.Controller) Deleter { return nil },
			storageBatchBuilder: func(_ *gomock.Controller) Deleter { return nil },
			errWrapped:          errTest,
			errMessage: "loading block hashes for block number to prune: " +
				"getting block hashes for block number 1: test error",
		},
		"prune storage error": {
			blockNumberToPrune: 1,
			journalDBBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				blockHashes := common.Hash{2}.ToBytes()
				database.EXPECT().Get([]byte("block_number_to_hash_1")).
					Return(blockHashes, nil)
				database.EXPECT().Get(scaleEncodeJournalKey(1, common.Hash{2})).
					Return(nil, errTest)
				return database
			},
			journalBatchBuilder: func(_ *gomock.Controller) Deleter { return nil },
			storageBatchBuilder: func(_ *gomock.Controller) Deleter { return nil },
			errWrapped:          errTest,
			errMessage: "pruning storage: getting deleted node hashes: " +
				"getting from database: test error",
		},
		"prune journal error": {
			blockNumberToPrune: 1,
			journalDBBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				blockHashes := common.Hash{2}.ToBytes()
				database.EXPECT().Get([]byte("block_number_to_hash_1")).Return(blockHashes, nil)
				database.EXPECT().Get(scaleEncodeJournalKey(1, common.Hash{2})).
					Return(scale.MustMarshal([]common.Hash{{3}}), nil)
				return database
			},
			journalBatchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del([]byte("block_number_to_hash_1")).Return(errTest)
				return batch
			},
			storageBatchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del(common.Hash{3}.ToBytes()).Return(nil)
				return batch
			},
			errWrapped: errTest,
			errMessage: "pruning journal: pruning block hashes: " +
				"deleting block hashes for block number 1 from database: " +
				"test error",
		},
		"success": {
			blockNumberToPrune: 1,
			journalDBBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				blockHashes := common.Hash{2}.ToBytes()
				database.EXPECT().Get([]byte("block_number_to_hash_1")).Return(blockHashes, nil)
				database.EXPECT().Get(scaleEncodeJournalKey(1, common.Hash{2})).
					Return(scale.MustMarshal([]common.Hash{{3}}), nil).
					Times(2) // pruneStorage + pruneJournal
				return database
			},
			journalBatchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del([]byte("block_number_to_hash_1")).Return(nil)
				batch.EXPECT().Del(scaleEncodeJournalKey(1, common.Hash{2})).Return(nil)
				batch.EXPECT().Del(makeDeletedKey(common.Hash{3})).Return(nil)
				return batch
			},
			storageBatchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del(common.Hash{3}.ToBytes()).Return(nil)
				return batch
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			journalDB := testCase.journalDBBuilder(ctrl)
			journalBatch := testCase.journalBatchBuilder(ctrl)
			storageBatch := testCase.storageBatchBuilder(ctrl)
			err := prune(testCase.blockNumberToPrune, journalDB, journalBatch, storageBatch)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_pruneStorage(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		blockNumber     uint32
		blockHashes     []common.Hash
		databaseBuilder func(ctrl *gomock.Controller) Getter
		batchBuilder    func(ctrl *gomock.Controller) Deleter
		errWrapped      error
		errMessage      string
	}{
		"get deleted node hashes error": {
			blockNumber: 10,
			blockHashes: []common.Hash{{1}},
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(10, common.Hash{1})).
					Return(nil, errTest)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Deleter { return nil },
			errWrapped:   errTest,
			errMessage:   "getting deleted node hashes: getting from database: test error",
		},
		"node hash deletion error": {
			blockNumber: 10,
			blockHashes: []common.Hash{{1}},
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(10, common.Hash{1})).
					Return(scale.MustMarshal([]common.Hash{{3}}), nil)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del(common.Hash{3}.ToBytes()).Return(errTest)
				return batch
			},
			errWrapped: errTest,
			errMessage: "deleting key from batch: test error",
		},
		"success": {
			blockNumber: 10,
			blockHashes: []common.Hash{{1}, {2}},
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(10, common.Hash{1})).
					Return(scale.MustMarshal([]common.Hash{{11}, {12}}), nil)
				database.EXPECT().Get(scaleEncodeJournalKey(10, common.Hash{2})).
					Return(scale.MustMarshal([]common.Hash{{13}}), nil)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del(common.Hash{11}.ToBytes()).Return(nil)
				batch.EXPECT().Del(common.Hash{12}.ToBytes()).Return(nil)
				batch.EXPECT().Del(common.Hash{13}.ToBytes()).Return(nil)
				return batch
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			database := testCase.databaseBuilder(ctrl)
			batch := testCase.batchBuilder(ctrl)
			err := pruneStorage(testCase.blockNumber,
				testCase.blockHashes, database, batch)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_pruneJournal(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		blockNumber     uint32
		blockHashes     []common.Hash
		databaseBuilder func(ctrl *gomock.Controller) Getter
		batchBuilder    func(ctrl *gomock.Controller) Deleter
		errWrapped      error
		errMessage      string
	}{
		"prune block hashes error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter { return nil },
			batchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del([]byte("block_number_to_hash_10")).Return(errTest)
				return batch
			},
			blockNumber: 10,
			errWrapped:  errTest,
			errMessage: "pruning block hashes: " +
				"deleting block hashes for block number 10 from database: " +
				"test error",
		},
		"get deleted node hashes error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(10, common.Hash{1})).
					Return(nil, errTest)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del([]byte("block_number_to_hash_10")).Return(nil)
				return batch
			},
			blockNumber: 10,
			blockHashes: []common.Hash{{1}},
			errWrapped:  errTest,
			errMessage: "getting deleted node hashes from database: " +
				"test error",
		},
		"decode deleted node hashes error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(10, common.Hash{1})).
					Return([]byte{99}, nil)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del([]byte("block_number_to_hash_10")).Return(nil)
				return batch
			},
			blockNumber: 10,
			blockHashes: []common.Hash{{1}},
			errWrapped:  io.EOF,
			errMessage:  "scale decoding deleted node hashes: reading bytes: EOF",
		},
		"delete deleted journal key error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(10, common.Hash{1})).
					Return(scale.MustMarshal([]common.Hash{{2}}), nil)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del([]byte("block_number_to_hash_10")).Return(nil)
				batch.EXPECT().Del(makeDeletedKey(common.Hash{2})).Return(errTest)
				return batch
			},
			blockNumber: 10,
			blockHashes: []common.Hash{{1}},
			errWrapped:  errTest,
			errMessage:  "deleting deleted node hash key from batch: test error",
		},
		"delete journal key error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(10, common.Hash{1})).
					Return(scale.MustMarshal([]common.Hash{{2}}), nil)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del([]byte("block_number_to_hash_10")).Return(nil)
				batch.EXPECT().Del(makeDeletedKey(common.Hash{2})).Return(nil)
				batch.EXPECT().Del(scaleEncodeJournalKey(10, common.Hash{1})).Return(errTest)
				return batch
			},
			blockNumber: 10,
			blockHashes: []common.Hash{{1}},
			errWrapped:  errTest,
			errMessage:  "deleting journal key from batch: test error",
		},
		"success": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(10, common.Hash{1})).
					Return(scale.MustMarshal([]common.Hash{{5}, {6}}), nil)
				database.EXPECT().Get(scaleEncodeJournalKey(10, common.Hash{2})).
					Return(scale.MustMarshal([]common.Hash{{7}}), nil)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				batch.EXPECT().Del([]byte("block_number_to_hash_10")).Return(nil)
				batch.EXPECT().Del(makeDeletedKey(common.Hash{5})).Return(nil)
				batch.EXPECT().Del(makeDeletedKey(common.Hash{6})).Return(nil)
				batch.EXPECT().Del(makeDeletedKey(common.Hash{7})).Return(nil)
				batch.EXPECT().Del(scaleEncodeJournalKey(10, common.Hash{1})).Return(nil)
				batch.EXPECT().Del(scaleEncodeJournalKey(10, common.Hash{2})).Return(nil)
				return batch
			},
			blockNumber: 10,
			blockHashes: []common.Hash{{1}, {2}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			database := testCase.databaseBuilder(ctrl)
			batch := testCase.batchBuilder(ctrl)
			err := pruneJournal(testCase.blockNumber,
				testCase.blockHashes, database, batch)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_pruneBlockHashes(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		blockNumber  uint32
		batchBuilder func(ctrl *gomock.Controller) Deleter
		errWrapped   error
		errMessage   string
	}{
		"delete from batch error": {
			blockNumber: 10,
			batchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				batch.EXPECT().Del(databaseKey).Return(errTest)
				return batch
			},
			errWrapped: errTest,
			errMessage: "deleting block hashes for block number 10 from database: test error",
		},
		"success": {
			blockNumber: 10,
			batchBuilder: func(ctrl *gomock.Controller) Deleter {
				batch := NewMockPutDeleter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				batch.EXPECT().Del(databaseKey).Return(nil)
				return batch
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			batch := testCase.batchBuilder(ctrl)
			err := pruneBlockHashes(testCase.blockNumber, batch)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
