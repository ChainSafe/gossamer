// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package full

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_storeDeletedNodeHashes(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		databaseBuilder   func(ctrl *gomock.Controller) Getter
		batchBuilder      func(ctrl *gomock.Controller) Putter
		blockNumber       uint32
		blockHash         common.Hash
		deletedNodeHashes map[common.Hash]struct{}
		errWrapped        error
		errMessage        string
	}{
		"get encoded journal keys error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(makeDeletedKey(common.Hash{3})).
					Return(nil, errTest)
				return database
			},
			batchBuilder:      func(ctrl *gomock.Controller) Putter { return nil },
			blockHash:         common.Hash{2},
			deletedNodeHashes: map[common.Hash]struct{}{{3}: {}},
			errWrapped:        errTest,
			errMessage: "getting journal keys for deleted node hash " +
				"from journal database: test error",
		},
		"decode journal keys error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(makeDeletedKey(common.Hash{3})).
					Return([]byte{99}, nil)
				return database
			},
			batchBuilder:      func(ctrl *gomock.Controller) Putter { return nil },
			blockHash:         common.Hash{2},
			deletedNodeHashes: map[common.Hash]struct{}{{3}: {}},
			errWrapped:        io.EOF,
			errMessage: "scale decoding journal keys for deleted node hash " +
				"from journal database: reading bytes: EOF",
		},
		"deleted node hash put error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(makeDeletedKey(common.Hash{3})).
					Return(nil, nil)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Putter {
				database := NewMockPutDeleter(ctrl)
				database.EXPECT().Put(
					makeDeletedKey(common.Hash{3}),
					scale.MustMarshal([]journalKey{{BlockNumber: 1, BlockHash: common.Hash{2}}}),
				).Return(errTest)
				return database
			},
			blockNumber:       1,
			blockHash:         common.Hash{2},
			deletedNodeHashes: map[common.Hash]struct{}{{3}: {}},
			errWrapped:        errTest,
			errMessage:        "putting journal keys in database batch: test error",
		},
		"encoded deleted node hashes put error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(makeDeletedKey(common.Hash{3})).
					Return(nil, nil)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Putter {
				database := NewMockPutDeleter(ctrl)
				database.EXPECT().Put(
					makeDeletedKey(common.Hash{3}),
					scale.MustMarshal([]journalKey{{BlockNumber: 1, BlockHash: common.Hash{2}}}),
				).Return(nil)
				database.EXPECT().Put(
					scaleEncodeJournalKey(1, common.Hash{2}),
					scale.MustMarshal([]common.Hash{{3}}),
				).Return(errTest)
				return database
			},
			blockNumber:       1,
			blockHash:         common.Hash{2},
			deletedNodeHashes: map[common.Hash]struct{}{{3}: {}},
			errWrapped:        errTest,
			errMessage:        "putting deleted node hashes in database batch: test error",
		},
		"success": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(makeDeletedKey(common.Hash{3})).
					Return(
						scale.MustMarshal([]journalKey{{BlockNumber: 5, BlockHash: common.Hash{5}}}),
						nil,
					)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Putter {
				database := NewMockPutDeleter(ctrl)
				database.EXPECT().Put(
					makeDeletedKey(common.Hash{3}),
					scale.MustMarshal([]journalKey{
						{BlockNumber: 5, BlockHash: common.Hash{5}},
						{BlockNumber: 1, BlockHash: common.Hash{2}},
					}),
				).Return(nil)
				database.EXPECT().Put(
					scaleEncodeJournalKey(1, common.Hash{2}),
					scale.MustMarshal([]common.Hash{{3}}),
				).Return(nil)
				return database
			},
			blockNumber:       1,
			blockHash:         common.Hash{2},
			deletedNodeHashes: map[common.Hash]struct{}{{3}: {}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			database := testCase.databaseBuilder(ctrl)
			batch := testCase.batchBuilder(ctrl)
			err := storeNodeHashesDeltas(database, batch, testCase.blockNumber,
				testCase.blockHash, testCase.deletedNodeHashes)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_getDeletedNodeHashes(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		databaseBuilder   func(ctrl *gomock.Controller) Getter
		blockNumber       uint32
		blockHash         common.Hash
		deletedNodeHashes []common.Hash
		errWrapped        error
		errMessage        string
	}{
		"get error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(1, common.Hash{2})).
					Return(nil, errTest)
				return database
			},
			blockNumber: 1,
			blockHash:   common.Hash{2},
			errWrapped:  errTest,
			errMessage:  "getting from database: test error",
		},
		"scale decoding error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(1, common.Hash{2})).
					Return([]byte{99}, nil)
				return database
			},
			blockNumber: 1,
			blockHash:   common.Hash{2},
			errWrapped:  io.EOF,
			errMessage:  "scale decoding deleted node hashes: reading bytes: EOF",
		},
		"success": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get(scaleEncodeJournalKey(1, common.Hash{2})).
					Return(scale.MustMarshal(
						[]common.Hash{{2}, {3}},
					), nil)
				return database
			},
			blockNumber:       1,
			blockHash:         common.Hash{2},
			deletedNodeHashes: []common.Hash{{2}, {3}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			database := testCase.databaseBuilder(ctrl)
			deletedNodeHashes, err := getDeletedNodeHashes(database,
				testCase.blockNumber, testCase.blockHash)

			assert.Equal(t, testCase.deletedNodeHashes, deletedNodeHashes)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_storeBlockNumberAtKey(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		batchBuilder func(ctrl *gomock.Controller) Putter
		key          []byte
		blockNumber  uint32
		errWrapped   error
		errMessage   string
	}{
		"put error": {
			batchBuilder: func(ctrl *gomock.Controller) Putter {
				database := NewMockPutDeleter(ctrl)
				database.EXPECT().Put([]byte("key"), scale.MustMarshal(uint32(1))).
					Return(errTest)
				return database
			},
			key:         []byte("key"),
			blockNumber: 1,
			errWrapped:  errTest,
			errMessage:  "putting block number 1: test error",
		},
		"success": {
			batchBuilder: func(ctrl *gomock.Controller) Putter {
				database := NewMockPutDeleter(ctrl)
				database.EXPECT().Put([]byte("key"), scale.MustMarshal(uint32(1))).
					Return(nil)
				return database
			},
			key:         []byte("key"),
			blockNumber: 1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			batch := testCase.batchBuilder(ctrl)
			err := storeBlockNumberAtKey(batch, testCase.key, testCase.blockNumber)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_getBlockNumberFromKey(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		databaseBuilder func(ctrl *gomock.Controller) Getter
		key             []byte
		blockNumber     uint32
		errWrapped      error
		errMessage      string
	}{
		"get error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				expectedKey := []byte("key")
				database.EXPECT().Get(expectedKey).Return(nil, errTest)
				return database
			},
			key:        []byte("key"),
			errWrapped: errTest,
			errMessage: "getting block number from database: test error",
		},
		"key not found": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				expectedKey := []byte("key")
				database.EXPECT().Get(expectedKey).Return(nil, chaindb.ErrKeyNotFound)
				return database
			},
			key:        []byte("key"),
			errWrapped: chaindb.ErrKeyNotFound,
			errMessage: "getting block number from database: Key not found",
		},
		"decoding error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				expectedKey := []byte("key")
				database.EXPECT().Get(expectedKey).Return([]byte{}, nil)
				return database
			},
			key:        []byte("key"),
			errWrapped: io.EOF,
			errMessage: "decoding block number: EOF",
		},
		"success": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				database.EXPECT().Get([]byte("key")).
					Return(scale.MustMarshal(uint32(1)), nil)
				return database
			},
			key:         []byte("key"),
			blockNumber: 1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			database := testCase.databaseBuilder(ctrl)
			blockNumber, err := getBlockNumberFromKey(database, testCase.key)

			assert.Equal(t, testCase.blockNumber, blockNumber)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_loadBlockHashes(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		databaseBuilder func(ctrl *gomock.Controller) Getter
		blockNumber     uint32
		blockHashes     []common.Hash
		errWrapped      error
		errMessage      string
	}{
		"get from database error": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				database.EXPECT().Get(databaseKey).Return(nil, errTest)
				return database
			},
			blockNumber: 10,
			errWrapped:  errTest,
			errMessage:  "getting block hashes for block number 10: test error",
		},
		"single block hash": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				database.EXPECT().Get(databaseKey).Return(common.Hash{2}.ToBytes(), nil)
				return database
			},
			blockNumber: 10,
			blockHashes: []common.Hash{{2}},
		},
		"multiple block hashes": {
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				databaseValue := bytes.Join([][]byte{
					common.Hash{2}.ToBytes(), common.Hash{3}.ToBytes(),
				}, nil)
				database.EXPECT().Get(databaseKey).
					Return(databaseValue, nil)
				return database
			},
			blockNumber: 10,
			blockHashes: []common.Hash{{2}, {3}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			database := testCase.databaseBuilder(ctrl)
			blockHashes, err := loadBlockHashes(testCase.blockNumber, database)

			assert.Equal(t, testCase.blockHashes, blockHashes)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_appendBlockHashes(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	testCases := map[string]struct {
		blockNumber     uint32
		blockHash       common.Hash
		databaseBuilder func(ctrl *gomock.Controller) Getter
		batchBuilder    func(ctrl *gomock.Controller) Putter
		errWrapped      error
		errMessage      string
	}{
		"get from database error": {
			blockNumber: 10,
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				database.EXPECT().Get(databaseKey).Return(nil, errTest)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Putter {
				return nil
			},
			errWrapped: errTest,
			errMessage: "getting block hashes for block number 10: test error",
		},
		"key not found": {
			blockNumber: 10,
			blockHash:   common.Hash{2},
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				database.EXPECT().Get(databaseKey).Return(nil, chaindb.ErrKeyNotFound)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Putter {
				batch := NewMockPutDeleter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				databaseValue := common.Hash{2}.ToBytes()
				batch.EXPECT().Put(databaseKey, databaseValue).Return(nil)
				return batch
			},
		},
		"put error": {
			blockNumber: 10,
			blockHash:   common.Hash{2},
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				database.EXPECT().Get(databaseKey).Return(nil, nil)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Putter {
				batch := NewMockPutDeleter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				databaseValue := common.Hash{2}.ToBytes()
				batch.EXPECT().Put(databaseKey, databaseValue).Return(errTest)
				return batch
			},
			errWrapped: errTest,
			errMessage: "putting block hashes for block number 10: test error",
		},
		"append to existing block hashes": {
			blockNumber: 10,
			blockHash:   common.Hash{2},
			databaseBuilder: func(ctrl *gomock.Controller) Getter {
				database := NewMockGetter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				databaseValue := bytes.Join([][]byte{
					common.Hash{1}.ToBytes(), common.Hash{3}.ToBytes(),
				}, nil)
				database.EXPECT().Get(databaseKey).Return(databaseValue, nil)
				return database
			},
			batchBuilder: func(ctrl *gomock.Controller) Putter {
				batch := NewMockPutDeleter(ctrl)
				databaseKey := []byte("block_number_to_hash_10")
				databaseValue := bytes.Join([][]byte{
					common.Hash{1}.ToBytes(), common.Hash{3}.ToBytes(), common.Hash{2}.ToBytes(),
				}, nil)
				batch.EXPECT().Put(databaseKey, databaseValue).Return(nil)
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
			err := appendBlockHash(testCase.blockNumber, testCase.blockHash, database, batch)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
