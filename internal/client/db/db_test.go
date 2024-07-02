// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/stretchr/testify/assert"
)

type noopExtrinsic struct{}

func (noopExtrinsic) IsSigned() *bool {
	return nil
}

// Check for interface fulfilment
var (
	_ blockchain.HeaderBackend[hash.H256, uint] = &blockchainDB[
		hash.H256, uint, noopExtrinsic, *generic.Header[uint, hash.H256, runtime.BlakeTwo256]]{}
	_ blockchain.HeaderMetadata[hash.H256, uint] = &blockchainDB[
		hash.H256, uint, noopExtrinsic, *generic.Header[uint, hash.H256, runtime.BlakeTwo256]]{}
	_ blockchain.Backend[hash.H256, uint] = &blockchainDB[
		hash.H256, uint, noopExtrinsic, *generic.Header[uint, hash.H256, runtime.BlakeTwo256]]{}
)

func TestNewBlockchainDB(t *testing.T) {
	db, err := newBlockchainDB[
		hash.H256,
		uint32,
		runtime.BlakeTwo256,
		noopExtrinsic,
		*generic.Header[uint32, hash.H256, runtime.BlakeTwo256],
	](database.NewMemDB[hash.H256]())
	assert.NoError(t, err)
	assert.NotNil(t, db)
}

func TestBlockchainDB_updateMeta(t *testing.T) {
	db, err := newBlockchainDB[
		hash.H256,
		uint32,
		runtime.BlakeTwo256,
		noopExtrinsic,
		*generic.Header[uint32, hash.H256, runtime.BlakeTwo256],
	](database.NewMemDB[hash.H256]())
	assert.NoError(t, err)
	assert.NotNil(t, db)

	expected := metaUpdate[hash.H256, uint32]{
		Hash:        hash.NewRandomH256(),
		Number:      1,
		IsBest:      true,
		IsFinalized: true,
		WithState:   true,
	}
	db.updateMeta(expected)

	assert.Equal(t, expected.Hash, db.meta.BestHash)
	assert.Equal(t, expected.Number, db.meta.BestNumber)
	assert.Equal(t, expected.Hash, db.meta.FinalizedHash)
	assert.Equal(t, expected.Number, db.meta.FinalizedNumber)
	assert.Equal(t, &finalizedState[hash.H256, uint32]{expected.Hash, expected.Number}, db.meta.FinalizedState)
}

func TestBlockchainDB_updateBlockGap(t *testing.T) {
	db, err := newBlockchainDB[
		hash.H256,
		uint32,
		runtime.BlakeTwo256,
		noopExtrinsic,
		*generic.Header[uint32, hash.H256, runtime.BlakeTwo256],
	](database.NewMemDB[hash.H256]())
	assert.NoError(t, err)
	assert.NotNil(t, db)

	expected := &[2]uint32{1, 1}
	db.updateBlockGap(expected)
	assert.Equal(t, &[2]uint32{1, 1}, expected)
}

func TestBlockchainDB_clearPinningCache(t *testing.T) {
	db, err := newBlockchainDB[
		hash.H256,
		uint32,
		runtime.BlakeTwo256,
		noopExtrinsic,
		*generic.Header[uint32, hash.H256, runtime.BlakeTwo256],
	](database.NewMemDB[hash.H256]())
	assert.NoError(t, err)
	assert.NotNil(t, db)

	db.clearPinningCache()
}

func TestBlockchainDB_insertJustificationsIfPinned(t *testing.T) {
	db, err := newBlockchainDB[
		hash.H256,
		uint32,
		runtime.BlakeTwo256,
		noopExtrinsic,
		*generic.Header[uint32, hash.H256, runtime.BlakeTwo256],
	](database.NewMemDB[hash.H256]())
	assert.NoError(t, err)
	assert.NotNil(t, db)

	someHash := hash.NewRandomH256()
	someJustification := runtime.Justification{
		ConsensusEngineID:    runtime.ConsensusEngineID{1, 1, 1, 1},
		EncodedJustification: runtime.EncodedJustification{1, 1, 1, 1},
	}
	db.insertJustifcationsIfPinned(someHash, someJustification)
	assert.False(t, db.pinnedBlocksCache.Contains(someHash))

	db.pinnedBlocksCache.Pin(someHash)
	db.insertJustifcationsIfPinned(someHash, someJustification)

	assert.True(t, db.pinnedBlocksCache.Contains(someHash))
}

func TestBlockchainDB_insertPersistedJustificationsIfPinned(t *testing.T) {
	memdb := database.NewMemDB[hash.H256]()
	db, err := newBlockchainDB[
		hash.H256,
		uint32,
		runtime.BlakeTwo256,
		noopExtrinsic,
		*generic.Header[uint32, hash.H256, runtime.BlakeTwo256],
	](memdb)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	someHash := hash.NewRandomH256()
	db.insertPersistedJustificationsIfPinned(someHash)
	assert.False(t, db.pinnedBlocksCache.Contains(someHash))

	// nothing in the db, but will pin `runtime.Justifications(nil)`
	db.pinnedBlocksCache.Pin(someHash)
	err = db.insertPersistedJustificationsIfPinned(someHash)
	assert.NoError(t, err)
	assert.True(t, db.pinnedBlocksCache.Contains(someHash))
}

func TestBlockchainDB_insertPersistedBodyIfPinned(t *testing.T) {
	memdb := database.NewMemDB[hash.H256]()
	db, err := newBlockchainDB[
		hash.H256,
		uint32,
		runtime.BlakeTwo256,
		noopExtrinsic,
		*generic.Header[uint32, hash.H256, runtime.BlakeTwo256],
	](memdb)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	someHash := hash.NewRandomH256()
	db.insertPersistedBodyIfPinned(someHash)
	assert.False(t, db.pinnedBlocksCache.Contains(someHash))

	// nothing in the db, but will pin `[]runtime.Extrinsic(nil)`
	db.pinnedBlocksCache.Pin(someHash)
	err = db.insertPersistedBodyIfPinned(someHash)
	assert.NoError(t, err)
	assert.True(t, db.pinnedBlocksCache.Contains(someHash))
}

func TestBlockchainDB_bumpRef(t *testing.T) {
	memdb := database.NewMemDB[hash.H256]()
	db, err := newBlockchainDB[
		hash.H256,
		uint32,
		runtime.BlakeTwo256,
		noopExtrinsic,
		*generic.Header[uint32, hash.H256, runtime.BlakeTwo256],
	](memdb)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	someHash := hash.NewRandomH256()
	db.bumpRef(someHash)
	assert.True(t, db.pinnedBlocksCache.Contains(someHash))
}

func TestBlockchainDB_unpin(t *testing.T) {
	memdb := database.NewMemDB[hash.H256]()
	db, err := newBlockchainDB[
		hash.H256,
		uint32,
		runtime.BlakeTwo256,
		noopExtrinsic,
		*generic.Header[uint32, hash.H256, runtime.BlakeTwo256],
	](memdb)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	someHash := hash.NewRandomH256()
	db.bumpRef(someHash)
	assert.True(t, db.pinnedBlocksCache.Contains(someHash))

	db.unpin(someHash)
	assert.False(t, db.pinnedBlocksCache.Contains(someHash))
}
