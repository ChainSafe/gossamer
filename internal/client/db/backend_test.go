// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/db/columns"
	statedb "github.com/ChainSafe/gossamer/internal/client/state-db"
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	memorykvdb "github.com/ChainSafe/gossamer/internal/kvdb/memory-kvdb"
	memorydb "github.com/ChainSafe/gossamer/internal/memory-db"
	p_blockchain "github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	rt_testing "github.com/ChainSafe/gossamer/internal/primitives/runtime/testing"
	statemachinebackend "github.com/ChainSafe/gossamer/internal/primitives/state-machine/backend"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

var (
	_ api.BlockImportOperation[
		uint32, hash.H256, runtime.BlakeTwo256, *generic.Header[uint32, hash.H256, runtime.BlakeTwo256], noopExtrinsic,
	] = &BlockImportOperation[
		hash.H256, runtime.BlakeTwo256, uint32, *generic.Header[uint32, hash.H256, runtime.BlakeTwo256], noopExtrinsic,
	]{}
	_ api.Backend[
		hash.H256, uint32, runtime.BlakeTwo256, *generic.Header[uint32, hash.H256, runtime.BlakeTwo256], noopExtrinsic,
	] = &Backend[
		hash.H256, runtime.BlakeTwo256, uint32, noopExtrinsic, *generic.Header[uint32, hash.H256, runtime.BlakeTwo256],
	]{}
)

func NewTestBackend(t *testing.T,
	blocksPruning BlocksPruning, canonicalizationDelay uint64,
) *Backend[
	hash.H256,
	runtime.BlakeTwo256,
	uint64,
	rt_testing.ExtrinsicsWrapper[uint64],
	*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
] {
	t.Helper()

	kvdb := memorykvdb.New(13)
	var statePruning statedb.PruningMode
	switch blocksPruning := blocksPruning.(type) {
	case BlocksPruningKeepAll:
		statePruning = statedb.PruningModeArchiveAll{}
	case BlocksPruningKeepFinalized:
		statePruning = statedb.PruningModeArchiveCanonical{}
	case BlocksPruningSome:
		statePruning = statedb.NewPruningModeConstrained(uint32(blocksPruning))
	default:
		t.Fatalf("unreachable")
	}
	trieCacheMaxSize := uint(16 * 1024 * 1024)
	dbSetting := DatabaseConfig{
		TrieCacheMaximumSize: &trieCacheMaxSize,
		StatePruning:         statePruning,
		Source:               DatabaseSource{DB: database.NewDBAdapter[hash.H256](kvdb), RequireCreateFlag: true},
		BlocksPruning:        blocksPruning,
	}

	backend, err := NewBackend[
		hash.H256,
		uint64,
		rt_testing.ExtrinsicsWrapper[uint64],
		runtime.BlakeTwo256,
		*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	](dbSetting, canonicalizationDelay)
	if err != nil {
		panic(err)
	}
	return backend
}

func insertHeader(t *testing.T,
	backend *Backend[
		hash.H256,
		runtime.BlakeTwo256,
		uint64,
		rt_testing.ExtrinsicsWrapper[uint64],
		*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	],
	number uint64,
	parentHash hash.H256,
	changes []trie.KeyValue, //nolint:unparam
	extrinisicsRoot hash.H256,
) hash.H256 {
	t.Helper()
	hash, err := insertBlock(
		t,
		backend,
		number,
		parentHash,
		changes,
		extrinisicsRoot,
		make([]rt_testing.ExtrinsicsWrapper[uint64], 0),
		nil,
	)
	require.NoError(t, err)
	return hash
}

func insertBlock(t *testing.T,
	backend *Backend[
		hash.H256,
		runtime.BlakeTwo256,
		uint64,
		rt_testing.ExtrinsicsWrapper[uint64],
		*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	],
	number uint64,
	parentHash hash.H256,
	_changes []trie.KeyValue,
	extrinisicsRoot hash.H256,
	body []rt_testing.ExtrinsicsWrapper[uint64],
	transactionIndex []overlayedchanges.IndexOperation,
) (hash.H256, error) {
	t.Helper()
	var digest runtime.Digest
	header := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		number, extrinisicsRoot, hash.H256(""), parentHash, digest,
	)

	var blockHash hash.H256
	if number != 0 {
		blockHash = parentHash
	}

	op := backend.beginOperation()
	err := backend.BeginStateOperation(op, blockHash)
	require.NoError(t, err)
	if transactionIndex != nil {
		err = op.UpdateTransactionIndex(transactionIndex)
		require.NoError(t, err)
	}

	// Insert some fake data to ensure that the block can be found in the state column.
	root, overlay := op.oldState.state.StorageRoot(
		[]statemachinebackend.Delta{{Key: blockHash.Bytes(), Value: blockHash.Bytes()}},
		storage.StateVersionV1,
	)
	err = op.UpdateDBStorage(overlay)
	require.NoError(t, err)
	header.SetStateRoot(root)

	err = op.SetBlockData(header, body, nil, nil, api.NewBlockStateBest)
	require.NoError(t, err)

	err = backend.CommitOperation(op)
	return header.Hash(), err
}

func insertHeaderNoHead(t *testing.T,
	backend *Backend[
		hash.H256,
		runtime.BlakeTwo256,
		uint64,
		rt_testing.ExtrinsicsWrapper[uint64],
		*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	],
	number uint64,
	parentHash hash.H256,
	extrinisicsRoot hash.H256,
) hash.H256 {
	var digest runtime.Digest
	header := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		number, extrinisicsRoot, hash.H256(""), parentHash, digest,
	)
	op := backend.beginOperation()

	state, err := backend.StateAt(parentHash)
	if err != nil {
		if parentHash == hash.H256("") {
			tb := backend.emptyState().state.TrieBackend
			state = tb
		} else {
			t.Fail()
		}
	}
	root, _ := state.StorageRoot([]statemachinebackend.Delta{
		{Key: parentHash.Bytes(), Value: parentHash.Bytes()},
	}, storage.StateVersionV1)
	header.SetStateRoot(root)

	err = op.SetBlockData(header, nil, nil, nil, api.NewBlockStateNormal)
	require.NoError(t, err)

	err = backend.CommitOperation(op)
	require.NoError(t, err)

	return header.Hash()

}

func TestBackend(t *testing.T) {
	t.Run("block_hash_inserted_correctly", func(t *testing.T) {
		var backing database.Database[hash.H256]
		{
			db := NewTestBackend(t, BlocksPruningSome(1), 0)
			for i := uint64(0); i < 10; i++ {
				h, err := db.Blockchain().Hash(i)
				require.NoError(t, err)
				require.Nil(t, h)

				{
					var hash hash.H256
					if i != 0 {
						h, err := db.blockchain.Hash(i - 1)
						require.NoError(t, err)
						require.NotNil(t, h)
						hash = *h
					}

					op, err := db.BeginOperation()
					require.NoError(t, err)
					err = db.BeginStateOperation(op, hash)
					require.NoError(t, err)
					header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
						i, dbHash(""), dbHash(""), hash, runtime.Digest{},
					)

					err = op.SetBlockData(header, nil, nil, nil, api.NewBlockStateBest)
					require.NoError(t, err)
					err = db.CommitOperation(op)
					require.NoError(t, err)
				}

				h, err = db.Blockchain().Hash(i)
				require.NoError(t, err)
				require.NotNil(t, h)
			}
			backing = db.storage.db
		}

		trieCacheMaxSize := uint(16 * 1024 * 1024)
		backend, err := NewBackend[
			dbHash,
			uint64,
			noopExtrinsic,
			runtime.BlakeTwo256,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
		](
			DatabaseConfig{
				TrieCacheMaximumSize: &trieCacheMaxSize,
				StatePruning:         statedb.NewPruningModeConstrained(1),
				Source:               DatabaseSource{DB: backing, RequireCreateFlag: false},
				BlocksPruning:        BlocksPruningKeepFinalized{},
			},
			0,
		)
		require.NoError(t, err)
		require.Equal(t, uint64(9), backend.Blockchain().Info().BestNumber)
		for i := uint64(0); i < 10; i++ {
			hash, err := backend.Blockchain().Hash(i)
			require.NoError(t, err)
			require.NotNil(t, hash)
		}
	})

	t.Run("set_state_data", func(t *testing.T) {
		for i, stateVersion := range []storage.StateVersion{storage.StateVersionV0, storage.StateVersionV1} {
			t.Run(fmt.Sprintf("StateVersion%d", i), func(t *testing.T) {
				db := NewTestBackend(t, BlocksPruningSome(2), 0)
				var hash dbHash
				{
					op := db.beginOperation()
					header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
						0, dbHash(""), dbHash(""), dbHash(""), runtime.Digest{},
					)

					deltas := []statemachinebackend.Delta{
						{Key: []byte{1, 3, 5}, Value: []byte{2, 4, 6}},
						{Key: []byte{1, 2, 3}, Value: []byte{9, 9, 9}},
					}

					root, _ := op.oldState.state.StorageRoot(deltas, stateVersion)
					header.SetStateRoot(root)
					h := header.Hash()

					top := btree.NewMap[string, []byte](0)
					for _, delta := range deltas {
						top.Set(string(delta.Key), delta.Value)
					}
					_, err := op.ResetStorage(storage.Storage{
						Top: *top,
					}, stateVersion)
					require.NoError(t, err)
					err = op.SetBlockData(header, nil, nil, nil, api.NewBlockStateBest)
					require.NoError(t, err)

					err = db.CommitOperation(op)
					require.NoError(t, err)

					state, err := db.StateAt(h)
					require.NoError(t, err)

					val, err := state.Storage([]byte{1, 3, 5})
					require.NoError(t, err)
					require.Equal(t, []byte{2, 4, 6}, []byte(val))
					val, err = state.Storage([]byte{1, 2, 3})
					require.NoError(t, err)
					require.Equal(t, []byte{9, 9, 9}, []byte(val))
					val, err = state.Storage([]byte{5, 5, 5})
					require.NoError(t, err)
					require.Nil(t, val)

					hash = h
				}

				{
					op := db.beginOperation()
					err := db.BeginStateOperation(op, hash)
					require.NoError(t, err)
					header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
						1,
						dbHash(""),
						dbHash(""),
						hash,
						runtime.Digest{},
					)

					deltas := []statemachinebackend.Delta{
						{Key: []byte{1, 3, 5}, Value: nil},
						{Key: []byte{5, 5, 5}, Value: []byte{4, 5, 6}},
					}

					root, overlay := op.oldState.state.StorageRoot(deltas, stateVersion)
					err = op.UpdateDBStorage(overlay)
					require.NoError(t, err)
					header.SetStateRoot(root)

					copiedDeltas := make(statemachinebackend.StorageCollection, 0)
					for _, delta := range deltas {
						copiedDeltas = append(copiedDeltas, statemachinebackend.StorageKeyValue{
							StorageKey:   delta.Key,
							StorageValue: delta.Value,
						})
					}
					err = op.UpdateStorage(copiedDeltas, nil)
					require.NoError(t, err)
					err = op.SetBlockData(header, nil, nil, nil, api.NewBlockStateBest)
					require.NoError(t, err)

					err = db.CommitOperation(op)
					require.NoError(t, err)

					state, err := db.StateAt(header.Hash())
					require.NoError(t, err)

					val, err := state.Storage([]byte{1, 3, 5})
					require.NoError(t, err)
					require.Nil(t, val)
					val, err = state.Storage([]byte{1, 2, 3})
					require.NoError(t, err)
					require.Equal(t, []byte{9, 9, 9}, []byte(val))
					val, err = state.Storage([]byte{5, 5, 5})
					require.NoError(t, err)
					require.Equal(t, []byte{4, 5, 6}, []byte(val))
				}
			})
		}
	})

	t.Run("delete_only_when_negative_rc", func(t *testing.T) {
		stateVersion := storage.StateVersionV1
		var key dbHash
		backend := NewTestBackend(t, BlocksPruningSome(1), 0)

		var hash dbHash
		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, dbHash(""))
			require.NoError(t, err)
			header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
				0,
				dbHash(""),
				dbHash(""),
				dbHash(""),
				runtime.Digest{},
			)

			root, _ := op.oldState.state.StorageRoot(nil, stateVersion)
			header.SetStateRoot(root)
			h := header.Hash()

			_, err = op.ResetStorage(storage.Storage{Top: *btree.NewMap[string, []byte](0)}, stateVersion)
			require.NoError(t, err)

			key = op.dbUpdates.Insert(hashdb.EmptyPrefix, []byte("hello"))
			err = op.SetBlockData(header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateBest)
			require.NoError(t, err)

			err = backend.CommitOperation(op)
			require.NoError(t, err)

			val := backend.storage.db.Get(columns.State, memorydb.NewPrefixedKey(key, hashdb.EmptyPrefix))
			require.Equal(t, []byte("hello"), val)

			hash = h
		}

		var hash1 dbHash
		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, hash)
			require.NoError(t, err)
			header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
				1,
				dbHash(""),
				dbHash(""),
				hash,
				runtime.Digest{},
			)

			deltas := []statemachinebackend.Delta{}

			root, _ := op.oldState.state.StorageRoot(deltas, stateVersion)
			header.SetStateRoot(root)
			h := header.Hash()

			op.dbUpdates.Insert(hashdb.EmptyPrefix, []byte("hello"))
			op.dbUpdates.Remove(key, hashdb.EmptyPrefix)

			err = op.SetBlockData(header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateBest)
			require.NoError(t, err)

			err = backend.CommitOperation(op)
			require.NoError(t, err)

			val := backend.storage.db.Get(columns.State, memorydb.NewPrefixedKey(key, hashdb.EmptyPrefix))
			require.Equal(t, []byte("hello"), val)

			hash1 = h
		}

		var hash2 dbHash
		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, hash1)
			require.NoError(t, err)
			header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
				2,
				dbHash(""),
				dbHash(""),
				hash1,
				runtime.Digest{},
			)

			deltas := []statemachinebackend.Delta{}

			root, _ := op.oldState.state.StorageRoot(deltas, stateVersion)
			header.SetStateRoot(root)
			h := header.Hash()

			op.dbUpdates.Remove(key, hashdb.EmptyPrefix)
			err = op.SetBlockData(header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateBest)
			require.NoError(t, err)

			err = backend.CommitOperation(op)
			require.NoError(t, err)

			val := backend.storage.db.Get(columns.State, memorydb.NewPrefixedKey(key, hashdb.EmptyPrefix))
			require.Equal(t, []byte("hello"), val)

			hash2 = h
		}

		var hash3 dbHash
		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, hash2)
			require.NoError(t, err)
			header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
				3,
				dbHash(""),
				dbHash(""),
				hash2,
				runtime.Digest{},
			)

			deltas := []statemachinebackend.Delta{}

			root, _ := op.oldState.state.StorageRoot(deltas, stateVersion)
			header.SetStateRoot(root)
			h := header.Hash()

			err = op.SetBlockData(header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateBest)
			require.NoError(t, err)

			err = backend.CommitOperation(op)
			require.NoError(t, err)

			val := backend.storage.db.Get(columns.State, memorydb.NewPrefixedKey(key, hashdb.EmptyPrefix))
			require.Equal(t, []byte("hello"), val)

			hash3 = h
		}

		var hash4 dbHash
		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, hash3)
			require.NoError(t, err)
			header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
				4,
				dbHash(""),
				dbHash(""),
				hash3,
				runtime.Digest{},
			)

			deltas := []statemachinebackend.Delta{}

			root, _ := op.oldState.state.StorageRoot(deltas, stateVersion)
			header.SetStateRoot(root)
			h := header.Hash()

			err = op.SetBlockData(header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateBest)
			require.NoError(t, err)

			err = backend.CommitOperation(op)
			require.NoError(t, err)

			val := backend.storage.db.Get(columns.State, memorydb.NewPrefixedKey(key, hashdb.EmptyPrefix))
			require.Nil(t, val)

			hash4 = h
		}

		err := backend.FinalizeBlock(hash1, nil)
		require.NoError(t, err)
		err = backend.FinalizeBlock(hash2, nil)
		require.NoError(t, err)
		err = backend.FinalizeBlock(hash3, nil)
		require.NoError(t, err)
		err = backend.FinalizeBlock(hash4, nil)
		require.NoError(t, err)

		val := backend.storage.db.Get(columns.State, memorydb.NewPrefixedKey(key, hashdb.EmptyPrefix))
		require.Nil(t, val)
	})

	t.Run("tree_route_works", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(1000), 100)
		blockchain := backend.blockchain
		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))

		// fork from genesis: 3 prong.
		a1 := insertHeader(t, backend, 1, block0, nil, hash.H256(""))
		a2 := insertHeader(t, backend, 2, a1, nil, hash.H256(""))
		a3 := insertHeader(t, backend, 3, a2, nil, hash.H256(""))

		// fork from genesis: 2 prong.
		b1 := insertHeader(t, backend, 1, block0, nil, hash.H256(bytes.Repeat([]byte{1}, 32)))
		b2 := insertHeader(t, backend, 2, b1, nil, hash.H256(""))

		{
			treeRoute, err := p_blockchain.NewTreeRoute(blockchain, a1, a1)
			require.NoError(t, err)

			require.Equal(t, a1, treeRoute.CommonBlock().Hash)
			require.Empty(t, treeRoute.Retracted())
			require.Empty(t, treeRoute.Enacted())
		}

		{
			treeRoute, err := p_blockchain.NewTreeRoute(blockchain, a3, b2)
			require.NoError(t, err)

			require.Equal(t, block0, treeRoute.CommonBlock().Hash)
			var retractedHashes []hash.H256
			for _, hn := range treeRoute.Retracted() {
				retractedHashes = append(retractedHashes, hn.Hash)
			}
			require.Equal(t, []hash.H256{a3, a2, a1}, retractedHashes)
		}

		{
			treeRoute, err := p_blockchain.NewTreeRoute(blockchain, a3, a1)
			require.NoError(t, err)

			require.Equal(t, a1, treeRoute.CommonBlock().Hash)
			var retractedHashes []hash.H256
			for _, hn := range treeRoute.Retracted() {
				retractedHashes = append(retractedHashes, hn.Hash)
			}
			require.Equal(t, []hash.H256{a3, a2}, retractedHashes)
			require.Empty(t, treeRoute.Enacted())
		}

		{
			treeRoute, err := p_blockchain.NewTreeRoute(blockchain, a2, a2)
			require.NoError(t, err)

			require.Equal(t, a2, treeRoute.CommonBlock().Hash)
			require.Empty(t, treeRoute.Retracted())
			require.Empty(t, treeRoute.Enacted())
		}
	})

	t.Run("tree_route_child", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(1000), 100)
		blockchain := backend.blockchain

		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))
		block1 := insertHeader(t, backend, 1, block0, nil, hash.H256(""))

		{
			treeRoute, err := p_blockchain.NewTreeRoute(blockchain, block0, block1)
			require.NoError(t, err)

			require.Equal(t, block0, treeRoute.CommonBlock().Hash)
			require.Empty(t, treeRoute.Retracted())
			var enactedHashes []hash.H256
			for _, hn := range treeRoute.Enacted() {
				enactedHashes = append(enactedHashes, hn.Hash)
			}
			require.Equal(t, []hash.H256{block1}, enactedHashes)
		}
	})

	t.Run("lowest_common_ancestor", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(1000), 100)
		blockchain := backend.blockchain
		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))

		// fork from genesis: 3 prong.
		a1 := insertHeader(t, backend, 1, block0, nil, hash.H256(""))
		a2 := insertHeader(t, backend, 2, a1, nil, hash.H256(""))
		a3 := insertHeader(t, backend, 3, a2, nil, hash.H256(""))

		// fork from genesis: 2 prong.
		b1 := insertHeader(t, backend, 1, block0, nil, hash.H256(bytes.Repeat([]byte{1}, 32)))
		b2 := insertHeader(t, backend, 2, b1, nil, hash.H256(""))

		{
			lca, err := p_blockchain.LowestCommonAncestor(blockchain, a3, b2)
			require.NoError(t, err)

			require.Equal(t, block0, lca.Hash)
			require.Equal(t, uint64(0), lca.Number)
		}

		{
			lca, err := p_blockchain.LowestCommonAncestor(blockchain, a1, a3)
			require.NoError(t, err)

			require.Equal(t, a1, lca.Hash)
			require.Equal(t, uint64(1), lca.Number)
		}

		{
			lca, err := p_blockchain.LowestCommonAncestor(blockchain, a3, a1)
			require.NoError(t, err)

			require.Equal(t, a1, lca.Hash)
			require.Equal(t, uint64(1), lca.Number)
		}

		{
			lca, err := p_blockchain.LowestCommonAncestor(blockchain, a2, a3)
			require.NoError(t, err)

			require.Equal(t, a2, lca.Hash)
			require.Equal(t, uint64(2), lca.Number)
		}

		{
			lca, err := p_blockchain.LowestCommonAncestor(blockchain, a2, a1)
			require.NoError(t, err)

			require.Equal(t, a1, lca.Hash)
			require.Equal(t, uint64(1), lca.Number)
		}

		{
			lca, err := p_blockchain.LowestCommonAncestor(blockchain, a2, a2)
			require.NoError(t, err)

			require.Equal(t, a2, lca.Hash)
			require.Equal(t, uint64(2), lca.Number)
		}
	})

	t.Run("leaves_pruned_on_finality", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)
		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))

		block1a := insertHeader(t, backend, 1, block0, nil, hash.H256(""))
		block1b := insertHeader(t, backend, 1, block0, nil, hash.H256(bytes.Repeat([]byte{1}, 32)))
		block1c := insertHeader(t, backend, 1, block0, nil, hash.H256(bytes.Repeat([]byte{2}, 32)))

		leaves, err := backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{block1a, block1b, block1c}, leaves)

		block2a := insertHeader(t, backend, 2, block1a, nil, hash.H256(""))
		block2b := insertHeader(t, backend, 2, block1b, nil, hash.H256(""))
		block2c := insertHeader(t, backend, 2, block1b, nil, hash.H256(bytes.Repeat([]byte{1}, 32)))

		leaves, err = backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{block2a, block2b, block2c, block1c}, leaves)

		err = backend.FinalizeBlock(block1a, nil)
		require.NoError(t, err)

		// leaves at same height stay. Leaves at lower heights pruned.
		leaves, err = backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{block2a, block2b, block2c, block1c}, leaves)

		err = backend.FinalizeBlock(block2a, nil)
		require.NoError(t, err)

		// leaves at same height stay. Leaves at lower heights pruned.
		leaves, err = backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{block2a, block2b, block2c}, leaves)
	})

	t.Run("test_aux", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(0), 0)
		val, err := backend.GetAux([]byte("test"))
		require.NoError(t, err)
		require.Nil(t, val)

		err = backend.InsertAux([]api.KeyValue{{Key: []byte("test"), Value: []byte("hello")}}, nil)
		require.NoError(t, err)
		val, err = backend.GetAux([]byte("test"))
		require.NoError(t, err)
		require.Equal(t, []byte("hello"), val)

		err = backend.InsertAux(nil, [][]byte{[]byte("test")})
		require.NoError(t, err)
		val, err = backend.GetAux([]byte("test"))
		require.NoError(t, err)
		require.Nil(t, val)
	})

	var CON0EngineID [4]byte
	copy(CON0EngineID[:], "CON0")

	var CON1EngineID [4]byte
	copy(CON1EngineID[:], "CON1")

	t.Run("finalize_block_with_justification", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)

		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))
		block1 := insertHeader(t, backend, 1, block0, nil, hash.H256(""))

		justification := runtime.Justification{
			ConsensusEngineID:    CON0EngineID,
			EncodedJustification: runtime.EncodedJustification{1, 2, 3},
		}
		err := backend.FinalizeBlock(block1, &justification)
		require.NoError(t, err)

		justifications, err := backend.Blockchain().Justifications(block1)
		require.NoError(t, err)
		require.Equal(t, runtime.Justifications{justification}, justifications)
	})

	t.Run("append_justification_to_finalized_block", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)

		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))
		block1 := insertHeader(t, backend, 1, block0, nil, hash.H256(""))

		just0 := runtime.Justification{
			ConsensusEngineID:    CON0EngineID,
			EncodedJustification: runtime.EncodedJustification{1, 2, 3},
		}
		err := backend.FinalizeBlock(block1, &just0)
		require.NoError(t, err)

		just1 := runtime.Justification{
			ConsensusEngineID:    CON1EngineID,
			EncodedJustification: runtime.EncodedJustification{4, 5},
		}
		err = backend.AppendJustification(block1, just1)
		require.NoError(t, err)

		just2 := runtime.Justification{
			ConsensusEngineID:    CON1EngineID,
			EncodedJustification: runtime.EncodedJustification{6, 7},
		}
		err = backend.AppendJustification(block1, just2)
		require.ErrorIs(t, err, p_blockchain.ErrBadJustification)

		expected := runtime.Justifications{just0, just1}
		justifications, err := backend.Blockchain().Justifications(block1)
		require.NoError(t, err)
		require.Equal(t, expected, justifications)
	})

	t.Run("finalize_multiple_blocks_in_single_op", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)

		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))
		block1 := insertHeader(t, backend, 1, block0, nil, hash.H256(""))
		block2 := insertHeader(t, backend, 2, block1, nil, hash.H256(""))
		block3 := insertHeader(t, backend, 3, block2, nil, hash.H256(""))
		block4 := insertHeader(t, backend, 4, block3, nil, hash.H256(""))

		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, block0)
			require.NoError(t, err)
			err = op.MarkFinalized(block1, nil)
			require.NoError(t, err)
			err = op.MarkFinalized(block2, nil)
			require.NoError(t, err)
			err = backend.CommitOperation(op)
			require.NoError(t, err)
		}

		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, block2)
			require.NoError(t, err)
			err = op.MarkFinalized(block3, nil)
			require.NoError(t, err)
			err = op.MarkFinalized(block4, nil)
			require.NoError(t, err)
			err = backend.CommitOperation(op)
			require.NoError(t, err)
		}
	})

	t.Run("storage_hash_is_cached_correctly", func(t *testing.T) {
		stateVersion := storage.StateVersionV1
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)

		var hash0 dbHash
		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, dbHash(""))
			require.NoError(t, err)
			header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
				0,
				dbHash(""),
				dbHash(""),
				dbHash(""),
				runtime.Digest{},
			)

			deltas := []statemachinebackend.Delta{
				{Key: []byte("test"), Value: []byte("test")},
			}

			root, _ := op.oldState.state.StorageRoot(deltas, stateVersion)
			header.SetStateRoot(root)
			h := header.Hash()

			top := btree.NewMap[string, []byte](0)
			for _, delta := range deltas {
				top.Set(string(delta.Key), delta.Value)
			}
			_, err = op.ResetStorage(storage.Storage{
				Top: *top,
			}, stateVersion)
			require.NoError(t, err)

			err = op.SetBlockData(header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateBest)
			require.NoError(t, err)

			err = backend.CommitOperation(op)
			require.NoError(t, err)

			hash0 = h
		}

		state, err := backend.StateAt(hash0)
		require.NoError(t, err)
		block0Hash, err := state.StorageHash([]byte("test"))
		require.NoError(t, err)
		require.NotNil(t, block0Hash)

		var hash1 dbHash
		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, hash0)
			require.NoError(t, err)
			header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
				1,
				dbHash(""),
				dbHash(""),
				hash0,
				runtime.Digest{},
			)

			deltas := []statemachinebackend.Delta{
				{Key: []byte("test"), Value: []byte("test2")},
			}

			root, overlay := op.oldState.state.StorageRoot(deltas, stateVersion)
			err = op.UpdateDBStorage(overlay)
			require.NoError(t, err)
			header.SetStateRoot(root)
			h := header.Hash()

			copiedDeltas := make(statemachinebackend.StorageCollection, 0)
			for _, delta := range deltas {
				copiedDeltas = append(copiedDeltas, statemachinebackend.StorageKeyValue{
					StorageKey:   delta.Key,
					StorageValue: delta.Value,
				})
			}
			err = op.UpdateStorage(copiedDeltas, nil)
			require.NoError(t, err)
			err = op.SetBlockData(header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateNormal)
			require.NoError(t, err)

			err = backend.CommitOperation(op)
			require.NoError(t, err)

			hash1 = h
		}

		{
			header, err := backend.Blockchain().Header(hash1)
			require.NoError(t, err)
			require.NotNil(t, header)
			op, err := backend.BeginOperation()
			require.NoError(t, err)
			err = op.SetBlockData(*header, nil, nil, nil, api.NewBlockStateBest)
			require.NoError(t, err)
			err = backend.CommitOperation(op)
			require.NoError(t, err)
		}

		state, err = backend.StateAt(hash1)
		require.NoError(t, err)
		block1Hash, err := state.StorageHash([]byte("test"))
		require.NoError(t, err)
		require.NotNil(t, block1Hash)

		require.NotEqual(t, *block0Hash, *block1Hash)
	})

	t.Run("finalize_non_sequential", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)

		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))
		block1 := insertHeader(t, backend, 1, block0, nil, hash.H256(""))
		block2 := insertHeader(t, backend, 2, block1, nil, hash.H256(""))

		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, block0)
			require.NoError(t, err)
			err = op.MarkFinalized(block2, nil)
			require.NoError(t, err)
			err = backend.CommitOperation(op)
			require.Error(t, err)
		}
	})

	t.Run("prune_blocks_on_finalize", func(t *testing.T) {
		pruningModes := []BlocksPruning{BlocksPruningSome(2), BlocksPruningKeepFinalized{}, BlocksPruningKeepAll{}}

		for _, pruningMode := range pruningModes {
			backend := NewTestBackend(t, pruningMode, 0)
			var blocks []hash.H256
			var prevHash hash.H256
			for i := 0; i < 5; i++ {
				hash, err := insertBlock(t,
					backend,
					uint64(i),
					prevHash,
					nil,
					hash.H256(""),
					[]rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(i)}},
					nil,
				)
				require.NoError(t, err)
				blocks = append(blocks, hash)
				prevHash = hash
			}

			{
				op := backend.beginOperation()
				err := backend.BeginStateOperation(op, blocks[4])
				require.NoError(t, err)
				for i := 1; i < 5; i++ {
					err = op.MarkFinalized(blocks[i], nil)
					require.NoError(t, err)
				}
				err = backend.CommitOperation(op)
				require.NoError(t, err)
			}

			bc := backend.Blockchain()
			switch pruningMode.(type) {
			case BlocksPruningSome:
				body, err := bc.Body(blocks[0])
				require.NoError(t, err)
				require.Nil(t, body)
				body, err = bc.Body(blocks[1])
				require.NoError(t, err)
				require.Nil(t, body)
				body, err = bc.Body(blocks[2])
				require.NoError(t, err)
				require.Nil(t, body)
				body, err = bc.Body(blocks[3])
				require.NoError(t, err)
				require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(3)}}, body)
				body, err = bc.Body(blocks[4])
				require.NoError(t, err)
				require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(4)}}, body)
			default:
				for i := 0; i < 5; i++ {
					body, err := bc.Body(blocks[i])
					require.NoError(t, err)
					require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(i)}}, body)
				}
			}
		}
	})

	t.Run("prune_blocks_on_finalize_with_fork", func(t *testing.T) {
		pruningModes := []BlocksPruning{BlocksPruningSome(2), BlocksPruningKeepFinalized{}, BlocksPruningKeepAll{}}

		for _, pruningMode := range pruningModes {
			backend := NewTestBackend(t, pruningMode, 10)
			var blocks []hash.H256
			var prevHash hash.H256
			for i := 0; i < 5; i++ {
				hash, err := insertBlock(t,
					backend,
					uint64(i),
					prevHash,
					nil,
					hash.H256(""),
					[]rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(i)}},
					nil,
				)
				require.NoError(t, err)
				blocks = append(blocks, hash)
				prevHash = hash
			}

			// insert a fork at block 2
			forkHashRoot, err := insertBlock(t,
				backend,
				2,
				blocks[1],
				nil,
				hash.NewRandomH256(),
				[]rt_testing.ExtrinsicsWrapper[uint64]{{T: 2}},
				nil,
			)
			require.NoError(t, err)

			_, err = insertBlock(t,
				backend,
				3,
				forkHashRoot,
				nil,
				hash.NewRandomH256(),
				[]rt_testing.ExtrinsicsWrapper[uint64]{{T: 2}, {T: 11}},
				nil,
			)
			require.NoError(t, err)
			op, err := backend.BeginOperation()
			require.NoError(t, err)
			err = backend.BeginStateOperation(op, blocks[4])
			require.NoError(t, err)
			err = op.MarkHead(blocks[4])
			require.NoError(t, err)
			err = backend.CommitOperation(op)
			require.NoError(t, err)

			bc := backend.Blockchain()
			body, err := bc.Body(forkHashRoot)
			require.NoError(t, err)
			require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(2)}}, body)

			for i := 1; i < 5; i++ {
				op, err := backend.BeginOperation()
				require.NoError(t, err)
				err = backend.BeginStateOperation(op, blocks[4])
				require.NoError(t, err)
				err = op.MarkFinalized(blocks[i], nil)
				require.NoError(t, err)
				err = backend.CommitOperation(op)
				require.NoError(t, err)
			}

			switch pruningMode.(type) {
			case BlocksPruningSome:
				body, err := bc.Body(blocks[0])
				require.NoError(t, err)
				require.Nil(t, body)
				body, err = bc.Body(blocks[1])
				require.NoError(t, err)
				require.Nil(t, body)
				body, err = bc.Body(blocks[2])
				require.NoError(t, err)
				require.Nil(t, body)
				body, err = bc.Body(blocks[3])
				require.NoError(t, err)
				require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(3)}}, body)
				body, err = bc.Body(blocks[4])
				require.NoError(t, err)
				require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(4)}}, body)
			default:
				for i := 0; i < 5; i++ {
					body, err := bc.Body(blocks[i])
					require.NoError(t, err)
					require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(i)}}, body)
				}
			}

			switch pruningMode.(type) {
			case BlocksPruningKeepAll:
				body, err := bc.Body(forkHashRoot)
				require.NoError(t, err)
				require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(2)}}, body)
			default:
				body, err := bc.Body(forkHashRoot)
				require.NoError(t, err)
				require.Nil(t, body)
			}

			require.Equal(t, uint64(4), bc.Info().BestNumber)
			for i := 0; i < 5; i++ {
				hash, err := bc.Hash(uint64(i))
				require.NoError(t, err)
				require.NotNil(t, hash)
			}
		}
	})

	t.Run("prune_blocks_on_finalize_and_reorg", func(t *testing.T) {
		//	0 - 1b
		//	\ - 1a - 2a - 3a
		//	     \ - 2b
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)

		var makeBlock = func(index uint64, parent hash.H256, val uint64) hash.H256 {
			hash, err := insertBlock(t,
				backend,
				index,
				parent,
				nil,
				hash.NewRandomH256(),
				[]rt_testing.ExtrinsicsWrapper[uint64]{{T: val}},
				nil,
			)
			require.NoError(t, err)
			return hash
		}

		block0 := makeBlock(0, "", 0)
		block1a := makeBlock(1, block0, 0x1a)
		block1b := makeBlock(1, block0, 0x1b)
		block2a := makeBlock(2, block1a, 0x2a)
		block2b := makeBlock(2, block1a, 0x2b)
		block3a := makeBlock(3, block2a, 0x3a)

		// Make sure 1b is head
		op, err := backend.BeginOperation()
		require.NoError(t, err)
		err = backend.BeginStateOperation(op, block0)
		require.NoError(t, err)
		err = op.MarkHead(block1b)
		require.NoError(t, err)
		err = backend.CommitOperation(op)
		require.NoError(t, err)

		// Finalize 3a
		op, err = backend.BeginOperation()
		require.NoError(t, err)
		err = backend.BeginStateOperation(op, block0)
		require.NoError(t, err)
		err = op.MarkHead(block3a)
		require.NoError(t, err)
		err = op.MarkFinalized(block1a, nil)
		require.NoError(t, err)
		err = op.MarkFinalized(block2a, nil)
		require.NoError(t, err)
		err = op.MarkFinalized(block3a, nil)
		require.NoError(t, err)

		err = backend.CommitOperation(op)
		require.NoError(t, err)

		bc := backend.Blockchain()
		body, err := bc.Body(block1b)
		require.NoError(t, err)
		require.Nil(t, body)
		body, err = bc.Body(block2b)
		require.NoError(t, err)
		require.Nil(t, body)
		body, err = bc.Body(block0)
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(0x00)}}, body)
		body, err = bc.Body(block1a)
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(0x1a)}}, body)
		body, err = bc.Body(block2a)
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(0x2a)}}, body)
		body, err = bc.Body(block3a)
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(0x3a)}}, body)
	})

	t.Run("indexed_data_block_body", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(1), 10)

		x0 := scale.MustMarshal(rt_testing.ExtrinsicsWrapper[uint64]{T: uint64(0)})
		x1 := scale.MustMarshal(rt_testing.ExtrinsicsWrapper[uint64]{T: uint64(1)})
		x0Hash := runtime.BlakeTwo256{}.Hash(x0[1:])
		x1Hash := runtime.BlakeTwo256{}.Hash(x1[1:])
		index := []overlayedchanges.IndexOperation{
			overlayedchanges.IndexOperationInsert{
				Extrinsic: 0,
				Hash:      x0Hash.Bytes(),
				Size:      uint32(len(x0)) - 1,
			},
			overlayedchanges.IndexOperationInsert{
				Extrinsic: 1,
				Hash:      x1Hash.Bytes(),
				Size:      uint32(len(x1)) - 1,
			},
		}
		hash, err := insertBlock(t, backend,
			0,
			"",
			nil,
			"",
			[]rt_testing.ExtrinsicsWrapper[uint64]{{T: 0}, {T: 1}},
			index,
		)
		require.NoError(t, err)
		bc := backend.Blockchain()
		tx, err := bc.IndexedTransaction(x0Hash)
		require.NoError(t, err)
		require.NotNil(t, tx)
		require.Equal(t, x0[1:], tx)
		tx, err = bc.IndexedTransaction(x1Hash)
		require.NoError(t, err)
		require.NotNil(t, tx)
		require.Equal(t, x1[1:], tx)

		hash0 := bc.Info().GenesisHash
		// Push one more blocks and make sure block is pruned and transaction index is cleared.
		block1, err := insertBlock(t, backend,
			1,
			hash,
			nil,
			"",
			[]rt_testing.ExtrinsicsWrapper[uint64]{},
			nil,
		)
		require.NoError(t, err)
		err = backend.FinalizeBlock(block1, nil)
		require.NoError(t, err)
		body, err := bc.Body(hash0)
		require.NoError(t, err)
		require.Nil(t, body)
		tx, err = bc.IndexedTransaction(x0Hash)
		require.NoError(t, err)
		require.Nil(t, tx)
		tx, err = bc.IndexedTransaction(x1Hash)
		require.NoError(t, err)
		require.Nil(t, tx)
	})

	t.Run("index_invalid_size", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(1), 10)

		x0 := scale.MustMarshal(rt_testing.ExtrinsicsWrapper[uint64]{T: uint64(0)})
		x1 := scale.MustMarshal(rt_testing.ExtrinsicsWrapper[uint64]{T: uint64(1)})
		x0Hash := runtime.BlakeTwo256{}.Hash(x0)
		x1Hash := runtime.BlakeTwo256{}.Hash(x1)
		index := []overlayedchanges.IndexOperation{
			overlayedchanges.IndexOperationInsert{
				Extrinsic: 0,
				Hash:      x0Hash.Bytes(),
				Size:      uint32(len(x0)),
			},
			overlayedchanges.IndexOperationInsert{
				Extrinsic: 1,
				Hash:      x1Hash.Bytes(),
				Size:      uint32(len(x1)) + 1,
			},
		}
		_, err := insertBlock(t, backend,
			0,
			"",
			nil,
			"",
			[]rt_testing.ExtrinsicsWrapper[uint64]{{T: 0}, {T: 1}},
			index,
		)
		require.NoError(t, err)
		bc := backend.Blockchain()
		tx, err := bc.IndexedTransaction(x0Hash)
		require.NoError(t, err)
		require.NotNil(t, tx)
		require.Equal(t, x0, tx)
		tx, err = bc.IndexedTransaction(x1Hash)
		require.NoError(t, err)
		require.Nil(t, tx)
	})

	t.Run("renew_transaction_storage", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(2), 10)
		var blocks []hash.H256
		var prevHash hash.H256
		x1 := scale.MustMarshal(rt_testing.ExtrinsicsWrapper[uint64]{T: uint64(0)})
		x1Hash := runtime.BlakeTwo256{}.Hash(x1[1:])
		for i := 0; i < 10; i++ {
			index := []overlayedchanges.IndexOperation{}
			if i == 0 {
				index = append(index, overlayedchanges.IndexOperationInsert{
					Extrinsic: 0,
					Hash:      x1Hash.Bytes(),
					Size:      uint32(len(x1) - 1),
				})
			} else if i < 5 {
				// keep renewing 1st
				index = append(index, overlayedchanges.IndexOperationRenew{
					Extrinsic: 0,
					Hash:      x1Hash.Bytes(),
				})
			} // else stop removing

			hash, err := insertBlock(t, backend,
				uint64(i),
				prevHash,
				nil,
				"",
				[]rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(i)}},
				index,
			)
			require.NoError(t, err)
			blocks = append(blocks, hash)
			prevHash = hash
		}

		for i := 0; i < 10; i++ {
			op, err := backend.BeginOperation()
			require.NoError(t, err)
			err = backend.BeginStateOperation(op, blocks[4])
			require.NoError(t, err)
			err = op.MarkFinalized(blocks[i], nil)
			require.NoError(t, err)
			err = backend.CommitOperation(op)
			require.NoError(t, err)
			bc := backend.Blockchain()
			if i < 6 {
				tx, err := bc.IndexedTransaction(x1Hash)
				require.NoError(t, err)
				require.NotNil(t, tx)
			} else {
				tx, err := bc.IndexedTransaction(x1Hash)
				require.NoError(t, err)
				require.Nil(t, tx)
			}
		}
	})

	t.Run("remove_leaf_block", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(2), 10)
		var blocks []hash.H256
		var prevHash hash.H256
		for i := uint64(0); i < 2; i++ {
			hash, err := insertBlock(t, backend,
				i,
				prevHash,
				nil,
				"",
				[]rt_testing.ExtrinsicsWrapper[uint64]{{T: i}},
				nil,
			)
			require.NoError(t, err)
			blocks = append(blocks, hash)
			prevHash = hash
		}

		for i := uint64(0); i < 2; i++ {
			hash, err := insertBlock(t, backend,
				2,
				blocks[1],
				nil,
				hash.NewRandomH256(),
				[]rt_testing.ExtrinsicsWrapper[uint64]{{T: i}},
				nil,
			)
			require.NoError(t, err)
			blocks = append(blocks, hash)
		}

		// insert a fork at block 1, which becomes best block
		bestHash, err := insertBlock(t, backend,
			uint64(1),
			blocks[0],
			nil,
			hash.NewRandomH256(),
			[]rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(42)}},
			nil,
		)
		require.NoError(t, err)

		require.Equal(t, bestHash, backend.Blockchain().Info().BestHash)
		err = backend.RemoveLeafBlock(bestHash)
		require.Error(t, err)

		leaves, err := backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{blocks[2], blocks[3], bestHash}, leaves)
		children, err := backend.Blockchain().Children(blocks[1])
		require.NoError(t, err)
		require.Equal(t, []hash.H256{blocks[2], blocks[3]}, children)

		require.True(t, backend.HaveStateAt(blocks[3], 2))
		header, err := backend.Blockchain().Header(blocks[3])
		require.NoError(t, err)
		require.NotNil(t, header)
		err = backend.RemoveLeafBlock(blocks[3])
		require.NoError(t, err)
		require.False(t, backend.HaveStateAt(blocks[3], 2))
		header, err = backend.Blockchain().Header(blocks[3])
		require.NoError(t, err)
		require.Nil(t, header)
		leaves, err = backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{blocks[2], bestHash}, leaves)
		children, err = backend.Blockchain().Children(blocks[1])
		require.NoError(t, err)
		require.Equal(t, []hash.H256{blocks[2]}, children)

		require.True(t, backend.HaveStateAt(blocks[2], 2))
		header, err = backend.Blockchain().Header(blocks[2])
		require.NoError(t, err)
		require.NotNil(t, header)
		err = backend.RemoveLeafBlock(blocks[2])
		require.NoError(t, err)
		require.False(t, backend.HaveStateAt(blocks[2], 2))
		header, err = backend.Blockchain().Header(blocks[2])
		require.NoError(t, err)
		require.Nil(t, header)
		leaves, err = backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{bestHash, blocks[1]}, leaves)
		children, err = backend.Blockchain().Children(blocks[1])
		require.NoError(t, err)
		require.Nil(t, children)

		require.True(t, backend.HaveStateAt(blocks[1], 1))
		header, err = backend.Blockchain().Header(blocks[1])
		require.NoError(t, err)
		require.NotNil(t, header)
		err = backend.RemoveLeafBlock(blocks[1])
		require.NoError(t, err)
		require.False(t, backend.HaveStateAt(blocks[1], 1))
		header, err = backend.Blockchain().Header(blocks[1])
		require.NoError(t, err)
		require.Nil(t, header)
		leaves, err = backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{bestHash}, leaves)
		children, err = backend.Blockchain().Children(blocks[0])
		require.NoError(t, err)
		require.Equal(t, []hash.H256{bestHash}, children)
	})

	t.Run("import_existing_block_as_new_head", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(10), 3)

		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))
		block1 := insertHeader(t, backend, 1, block0, nil, hash.H256(""))
		block2 := insertHeader(t, backend, 2, block1, nil, hash.H256(""))
		block3 := insertHeader(t, backend, 3, block2, nil, hash.H256(""))
		block4 := insertHeader(t, backend, 4, block3, nil, hash.H256(""))
		block5 := insertHeader(t, backend, 5, block4, nil, hash.H256(""))
		require.Equal(t, block5, backend.Blockchain().Info().BestHash)

		// Insert 1 as best again. This should fail because canonicalization_delay == 3
		// and best == 5
		trie := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](
			trie.NewPrefixedMemoryDB[hash.H256, runtime.BlakeTwo256](),
		)
		trie.SetVersion(triedb.V1)
		header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
			1,
			dbHash(""),
			trie.MustHash(),
			block0,
			runtime.Digest{},
		)
		op, err := backend.BeginOperation()
		require.NoError(t, err)
		err = op.SetBlockData(header, nil, nil, nil, api.NewBlockStateBest)
		require.NoError(t, err)
		err = backend.CommitOperation(op)
		require.ErrorIs(t, err, p_blockchain.ErrSetHeadTooOld)

		// Insert 2 as best again
		header2, err := backend.Blockchain().Header(block2)
		require.NoError(t, err)
		require.NotNil(t, header2)
		op, err = backend.BeginOperation()
		require.NoError(t, err)
		err = op.SetBlockData(*header2, nil, nil, nil, api.NewBlockStateBest)
		require.NoError(t, err)
		err = backend.CommitOperation(op)
		require.NoError(t, err)
		require.Equal(t, block2, backend.Blockchain().Info().BestHash)
	})

	t.Run("impoort_existing_state_fails", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)

		genesis, err := insertBlock(t, backend, 0, "", nil, "", nil, nil)
		require.NoError(t, err)

		_, err = insertBlock(t, backend, 1, genesis, nil, "", nil, nil)
		require.NoError(t, err)

		_, err = insertBlock(t, backend, 1, genesis, nil, "", nil, nil)
		require.ErrorIs(t, err, p_blockchain.ErrStateDatabase)
	})

	t.Run("leaves_not_created_for_ancient_blocks", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)

		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))

		block1a := insertHeader(t, backend, 1, block0, nil, "")
		block2a := insertHeader(t, backend, 2, block1a, nil, "")
		err := backend.FinalizeBlock(block1a, nil)
		require.NoError(t, err)
		leaves, err := backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{block2a}, leaves)

		// Insert a fork prior to finalization point. Leave should not be created.
		insertHeaderNoHead(t, backend, 1, block0, hash.H256(bytes.Repeat([]byte{1}, 32)))
		leaves, err = backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{block2a}, leaves)
	})

	t.Run("revert_finalized_blocks", func(t *testing.T) {
		pruningModes := []BlocksPruning{
			BlocksPruningSome(10),
			BlocksPruningKeepAll{},
		}

		// we will create a chain with 11 blocks, finalize block #8 and then
		// attempt to revert 5 blocks.
		for _, pruningMode := range pruningModes {
			t.Run(fmt.Sprintf("%T", pruningMode), func(t *testing.T) {
				backend := NewTestBackend(t, pruningMode, 1)

				var parent hash.H256
				for i := uint64(0); i <= 10; i++ {
					var err error
					parent, err = insertBlock(t, backend, i, parent, nil, "", nil, nil)
					require.NoError(t, err)
				}

				require.Equal(t, uint64(10), backend.Blockchain().Info().BestNumber)

				block8, err := backend.Blockchain().Hash(8)
				require.NoError(t, err)
				require.NotNil(t, block8)
				err = backend.FinalizeBlock(*block8, nil)
				require.NoError(t, err)
				_, _, err = backend.Revert(5, true)
				require.NoError(t, err)

				_, ok := pruningMode.(BlocksPruningSome)
				if ok {
					// we can only revert to blocks for which we have state, if pruning is enabled
					// then the last state available will be that of the latest finalized block
					require.Equal(t, uint64(8), backend.Blockchain().Info().BestNumber)
				} else {
					// otherwise if we're not doing state pruning we can revert past finalized blocks
					require.Equal(t, uint64(5), backend.Blockchain().Info().BestNumber)
				}

			})
		}
	})

	t.Run("revert_non_best_blocks", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)

		genesis, err := insertBlock(t, backend, 0, "", nil, "", nil, nil)
		require.NoError(t, err)

		block1, err := insertBlock(t, backend, 1, genesis, nil, "", nil, nil)
		require.NoError(t, err)

		block2, err := insertBlock(t, backend, 2, block1, nil, "", nil, nil)
		require.NoError(t, err)

		var block3 hash.H256
		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, block1)
			require.NoError(t, err)
			trie := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](
				trie.NewPrefixedMemoryDB[hash.H256, runtime.BlakeTwo256](),
			)
			trie.SetVersion(triedb.V1)
			header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
				3,
				dbHash(""),
				trie.MustHash(),
				block2,
				runtime.Digest{},
			)

			err = op.SetBlockData(header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateNormal)
			require.NoError(t, err)

			err = backend.CommitOperation(op)
			require.NoError(t, err)

			block3 = header.Hash()
		}

		var block4 hash.H256
		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, block2)
			require.NoError(t, err)
			trie := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](
				trie.NewPrefixedMemoryDB[hash.H256, runtime.BlakeTwo256](),
			)
			trie.SetVersion(triedb.V1)
			header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
				4,
				"",
				trie.MustHash(),
				block3,
				runtime.Digest{},
			)

			err = op.SetBlockData(header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateNormal)
			require.NoError(t, err)

			err = backend.CommitOperation(op)
			require.NoError(t, err)

			block4 = header.Hash()
		}

		var block3Fork hash.H256
		{
			op := backend.beginOperation()
			err := backend.BeginStateOperation(op, block2)
			require.NoError(t, err)
			trie := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](
				trie.NewPrefixedMemoryDB[hash.H256, runtime.BlakeTwo256](),
			)
			trie.SetVersion(triedb.V1)
			header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
				3,
				hash.NewH256FromLowUint64BigEndian(42),
				trie.MustHash(),
				block2,
				runtime.Digest{},
			)

			err = op.SetBlockData(header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateNormal)
			require.NoError(t, err)

			err = backend.CommitOperation(op)
			require.NoError(t, err)

			block3Fork = header.Hash()
		}

		require.True(t, backend.HaveStateAt(block1, 1))
		require.True(t, backend.HaveStateAt(block2, 2))
		require.True(t, backend.HaveStateAt(block3, 3))
		require.True(t, backend.HaveStateAt(block4, 4))
		require.True(t, backend.HaveStateAt(block3Fork, 3))

		leaves, err := backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{block4, block3Fork}, leaves)

		leaf := backend.blockchain.leaves.HighestLeaf()
		require.NotNil(t, leaf)
		require.Equal(t, uint64(4), leaf.Number)

		number, _, err := backend.Revert(1, false)
		require.NoError(t, err)
		require.Equal(t, uint64(3), number)

		require.True(t, backend.HaveStateAt(block1, 1))
		require.False(t, backend.HaveStateAt(block2, 2))
		require.False(t, backend.HaveStateAt(block3, 3))
		require.False(t, backend.HaveStateAt(block4, 4))
		require.False(t, backend.HaveStateAt(block3Fork, 3))

		leaves, err = backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{block1}, leaves)

		leaf = backend.blockchain.leaves.HighestLeaf()
		require.NotNil(t, leaf)
		require.Equal(t, uint64(1), leaf.Number)
	})

	t.Run("no_duplicated_leaves_allowed", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(10), 10)

		block0 := insertHeader(t, backend, 0, hash.H256(""), nil, hash.H256(""))
		block1 := insertHeader(t, backend, 1, block0, nil, hash.H256(""))
		// Add block 2 not as the best block
		block2 := insertHeaderNoHead(t, backend, 2, block1, "")
		leaves, err := backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{block2}, leaves)
		require.Equal(t, block1, backend.Blockchain().Info().BestHash)

		// Add block 2 as the best block
		block2 = insertHeader(t, backend, 2, block1, nil, "")
		leaves, err = backend.Blockchain().Leaves()
		require.NoError(t, err)
		require.Equal(t, []hash.H256{block2}, leaves)
		require.Equal(t, block2, backend.Blockchain().Info().BestHash)
	})

	t.Run("force_delayed_canonicalize_waiting_for_blocks_to_be_finalized", func(t *testing.T) {
		pruningModes := []BlocksPruning{
			BlocksPruningSome(10),
			BlocksPruningKeepAll{},
			BlocksPruningKeepFinalized{},
		}

		for _, pruningMode := range pruningModes {
			t.Run(fmt.Sprintf("%T", pruningMode), func(t *testing.T) {
				backend := NewTestBackend(t, pruningMode, 1)

				genesis, err := insertBlock(t, backend, 0, "", nil, "", nil, nil)
				require.NoError(t, err)

				var block1 hash.H256
				{
					op := backend.beginOperation()
					err := backend.BeginStateOperation(op, genesis)
					require.NoError(t, err)
					header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
						1,
						"",
						"",
						genesis,
						runtime.Digest{},
					)

					deltas := []statemachinebackend.Delta{
						{Key: []byte{1, 3, 5}, Value: nil},
						{Key: []byte{5, 5, 5}, Value: []byte{4, 5, 6}},
					}

					root, overlay := op.oldState.state.StorageRoot(deltas, storage.StateVersionV1)
					err = op.UpdateDBStorage(overlay)
					require.NoError(t, err)
					header.SetStateRoot(root)
					h := header.Hash()

					copiedDeltas := make(statemachinebackend.StorageCollection, 0)
					for _, delta := range deltas {
						copiedDeltas = append(copiedDeltas, statemachinebackend.StorageKeyValue{
							StorageKey:   delta.Key,
							StorageValue: delta.Value,
						})
					}
					err = op.UpdateStorage(copiedDeltas, nil)
					require.NoError(t, err)
					err = op.SetBlockData(
						header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateNormal,
					)
					require.NoError(t, err)

					err = backend.CommitOperation(op)
					require.NoError(t, err)

					block1 = h
				}

				_, ok := pruningMode.(BlocksPruningSome)
				if ok {
					require.Equal(t, statedb.LastCanonicalizedBlock(0), backend.storage.StateDB.LastCanonicalized())
				}

				// This should not trigger any forced canonicalization as we didn't have imported any
				// best block by now.
				var block2 hash.H256
				{
					op := backend.beginOperation()
					err := backend.BeginStateOperation(op, block1)
					require.NoError(t, err)
					header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
						2,
						"",
						"",
						block1,
						runtime.Digest{},
					)

					deltas := []statemachinebackend.Delta{
						{Key: []byte{5, 5, 5}, Value: []byte{4, 5, 6, 2}},
					}

					root, overlay := op.oldState.state.StorageRoot(deltas, storage.StateVersionV1)
					err = op.UpdateDBStorage(overlay)
					require.NoError(t, err)
					header.SetStateRoot(root)
					h := header.Hash()

					copiedDeltas := make(statemachinebackend.StorageCollection, 0)
					for _, delta := range deltas {
						copiedDeltas = append(copiedDeltas, statemachinebackend.StorageKeyValue{
							StorageKey:   delta.Key,
							StorageValue: delta.Value,
						})
					}
					err = op.UpdateStorage(copiedDeltas, nil)
					require.NoError(t, err)
					err = op.SetBlockData(
						header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateNormal,
					)
					require.NoError(t, err)

					err = backend.CommitOperation(op)
					require.NoError(t, err)

					block2 = h
				}

				_, ok = pruningMode.(BlocksPruningSome)
				if ok {
					require.Equal(t, statedb.LastCanonicalizedBlock(0), backend.storage.StateDB.LastCanonicalized())
				}

				// This should also not trigger it yet, because we import a best block, but the best
				// block from the POV of the db is still at 0.
				var block3 hash.H256
				{
					op := backend.beginOperation()
					err := backend.BeginStateOperation(op, block2)
					require.NoError(t, err)
					header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
						3,
						"",
						"",
						block2,
						runtime.Digest{},
					)

					deltas := []statemachinebackend.Delta{
						{Key: []byte{5, 5, 5}, Value: []byte{4, 5, 6, 3}},
					}

					root, overlay := op.oldState.state.StorageRoot(deltas, storage.StateVersionV1)
					err = op.UpdateDBStorage(overlay)
					require.NoError(t, err)
					header.SetStateRoot(root)
					h := header.Hash()

					copiedDeltas := make(statemachinebackend.StorageCollection, 0)
					for _, delta := range deltas {
						copiedDeltas = append(copiedDeltas, statemachinebackend.StorageKeyValue{
							StorageKey:   delta.Key,
							StorageValue: delta.Value,
						})
					}
					err = op.UpdateStorage(copiedDeltas, nil)
					require.NoError(t, err)
					err = op.SetBlockData(
						header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateBest,
					)
					require.NoError(t, err)

					err = backend.CommitOperation(op)
					require.NoError(t, err)

					block3 = h
				}

				_, ok = pruningMode.(BlocksPruningSome)
				if ok {
					require.Equal(t, statedb.LastCanonicalizedBlock(0), backend.storage.StateDB.LastCanonicalized())
				}

				// Now it should kick in.
				var block4 hash.H256
				{
					op := backend.beginOperation()
					err := backend.BeginStateOperation(op, block3)
					require.NoError(t, err)
					header := generic.NewHeader[uint64, dbHash, runtime.BlakeTwo256](
						4,
						"",
						"",
						block3,
						runtime.Digest{},
					)

					deltas := []statemachinebackend.Delta{
						{Key: []byte{5, 5, 5}, Value: []byte{4, 5, 6, 4}},
					}

					root, overlay := op.oldState.state.StorageRoot(deltas, storage.StateVersionV1)
					err = op.UpdateDBStorage(overlay)
					require.NoError(t, err)
					header.SetStateRoot(root)
					h := header.Hash()

					copiedDeltas := make(statemachinebackend.StorageCollection, 0)
					for _, delta := range deltas {
						copiedDeltas = append(copiedDeltas, statemachinebackend.StorageKeyValue{
							StorageKey:   delta.Key,
							StorageValue: delta.Value,
						})
					}
					err = op.UpdateStorage(copiedDeltas, nil)
					require.NoError(t, err)
					err = op.SetBlockData(
						header, []rt_testing.ExtrinsicsWrapper[uint64]{}, nil, nil, api.NewBlockStateBest,
					)
					require.NoError(t, err)

					err = backend.CommitOperation(op)
					require.NoError(t, err)

					block4 = h
				}

				_, ok = pruningMode.(BlocksPruningSome)
				if ok {
					require.Equal(t, statedb.LastCanonicalizedBlock(2), backend.storage.StateDB.LastCanonicalized())
				}

				hash, err := backend.Blockchain().Hash(1)
				require.NoError(t, err)
				require.NotNil(t, hash)
				require.Equal(t, block1, *hash)
				hash, err = backend.Blockchain().Hash(2)
				require.NoError(t, err)
				require.NotNil(t, hash)
				require.Equal(t, block2, *hash)
				hash, err = backend.Blockchain().Hash(3)
				require.NoError(t, err)
				require.NotNil(t, hash)
				require.Equal(t, block3, *hash)
				hash, err = backend.Blockchain().Hash(4)
				require.NoError(t, err)
				require.NotNil(t, hash)
				require.Equal(t, block4, *hash)
			})
		}
	})

	t.Run("pinned_blocks_on_finalize", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(1), 10)
		var blocks []hash.H256
		var prevHash hash.H256

		var buildJustification = func(i uint64) *runtime.Justification {
			return &runtime.Justification{
				ConsensusEngineID:    [4]byte{},
				EncodedJustification: []byte{uint8(i)},
			}
		}

		// Block tree:
		//   0 -> 1 -> 2 -> 3 -> 4
		for i := 0; i < 5; i++ {
			hash, err := insertBlock(t,
				backend,
				uint64(i),
				prevHash,
				nil,
				hash.H256(""),
				[]rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(i)}},
				nil,
			)
			require.NoError(t, err)
			blocks = append(blocks, hash)
			// Avoid block pruning.
			err = backend.PinBlock(blocks[i])
			require.NoError(t, err)

			prevHash = hash
		}

		bc := backend.Blockchain()

		// Check that we can properly access values when there is reference count
		// but no value.
		body, err := bc.Body(blocks[1])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(1)}}, body)

		// Block 1 gets pinned three times
		err = backend.PinBlock(blocks[1])
		require.NoError(t, err)
		err = backend.PinBlock(blocks[1])
		require.NoError(t, err)

		// Finalize all blocks. This will trigger pruning.
		op := backend.beginOperation()
		err = backend.BeginStateOperation(op, blocks[4])
		require.NoError(t, err)
		for i := 1; i < 5; i++ {
			err := op.MarkFinalized(blocks[i], buildJustification(uint64(i)))
			require.NoError(t, err)
		}
		err = backend.CommitOperation(op)
		require.NoError(t, err)

		// Block 0, 1, 2, 3 are pinned, so all values should be cached.
		// Block 4 is inside the pruning window, its value is in db.
		body, err = bc.Body(blocks[0])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(0)}}, body)

		body, err = bc.Body(blocks[1])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(1)}}, body)
		justifications, err := bc.Justifications(blocks[1])
		require.NoError(t, err)
		require.Equal(t, runtime.Justifications{*buildJustification(1)}, justifications)

		body, err = bc.Body(blocks[2])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(2)}}, body)
		justifications, err = bc.Justifications(blocks[2])
		require.NoError(t, err)
		require.Equal(t, runtime.Justifications{*buildJustification(2)}, justifications)

		body, err = bc.Body(blocks[3])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(3)}}, body)
		justifications, err = bc.Justifications(blocks[3])
		require.NoError(t, err)
		require.Equal(t, runtime.Justifications{*buildJustification(3)}, justifications)

		body, err = bc.Body(blocks[4])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(4)}}, body)
		justifications, err = bc.Justifications(blocks[4])
		require.NoError(t, err)
		require.Equal(t, runtime.Justifications{*buildJustification(4)}, justifications)

		// Unpin all blocks. Values should be removed from cache.
		for _, block := range blocks {
			backend.UnpinBlock(block)
		}

		body, err = bc.Body(blocks[0])
		require.NoError(t, err)
		require.Nil(t, body)
		// Block 1 was pinned twice, we expect it to be still cached
		body, err = bc.Body(blocks[1])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(1)}}, body)
		justifications, err = bc.Justifications(blocks[1])
		require.NoError(t, err)
		require.Equal(t, runtime.Justifications{*buildJustification(1)}, justifications)
		// Headers should also be available while pinned
		header, err := bc.Header(blocks[1])
		require.NoError(t, err)
		require.NotNil(t, header)
		body, err = bc.Body(blocks[2])
		require.NoError(t, err)
		require.Nil(t, body)
		justifications, err = bc.Justifications(blocks[2])
		require.NoError(t, err)
		require.Nil(t, justifications)
		body, err = bc.Body(blocks[3])
		require.NoError(t, err)
		require.Nil(t, body)
		justifications, err = bc.Justifications(blocks[3])
		require.NoError(t, err)
		require.Nil(t, justifications)

		// After these unpins, block 1 should also be removed
		backend.UnpinBlock(blocks[1])
		body, err = bc.Body(blocks[1])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(1)}}, body)
		justifications, err = bc.Justifications(blocks[1])
		require.NoError(t, err)
		require.Equal(t, runtime.Justifications{*buildJustification(1)}, justifications)
		backend.UnpinBlock(blocks[1])
		body, err = bc.Body(blocks[1])
		require.NoError(t, err)
		require.Nil(t, body)
		justifications, err = bc.Justifications(blocks[1])
		require.NoError(t, err)
		require.Nil(t, justifications)

		// Block 4 is inside the pruning window and still kept
		body, err = bc.Body(blocks[4])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(4)}}, body)
		justifications, err = bc.Justifications(blocks[4])
		require.NoError(t, err)
		require.Equal(t, runtime.Justifications{*buildJustification(4)}, justifications)

		// Block tree:
		//   0 -> 1 -> 2 -> 3 -> 4 -> 5
		hash, err := insertBlock(
			t, backend, 5, prevHash, nil, "", []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(5)}}, nil,
		)
		require.NoError(t, err)
		blocks = append(blocks, hash)

		err = backend.PinBlock(blocks[4])
		require.NoError(t, err)
		// Mark block 5 as finalized.
		op = backend.beginOperation()
		err = backend.BeginStateOperation(op, blocks[5])
		require.NoError(t, err)
		err = op.MarkFinalized(blocks[5], buildJustification(uint64(5)))
		require.NoError(t, err)
		err = backend.CommitOperation(op)
		require.NoError(t, err)

		body, err = bc.Body(blocks[0])
		require.NoError(t, err)
		require.Nil(t, body)
		body, err = bc.Body(blocks[1])
		require.NoError(t, err)
		require.Nil(t, body)
		body, err = bc.Body(blocks[2])
		require.NoError(t, err)
		require.Nil(t, body)
		body, err = bc.Body(blocks[3])
		require.NoError(t, err)
		require.Nil(t, body)

		body, err = bc.Body(blocks[4])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(4)}}, body)
		justifications, err = bc.Justifications(blocks[4])
		require.NoError(t, err)
		require.Equal(t, runtime.Justifications{*buildJustification(4)}, justifications)
		body, err = bc.Body(blocks[5])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(5)}}, body)
		header, err = bc.Header(blocks[5])
		require.NoError(t, err)
		require.NotNil(t, header)

		backend.UnpinBlock(blocks[4])
		body, err = bc.Body(blocks[4])
		require.NoError(t, err)
		require.Nil(t, body)
		justifications, err = bc.Justifications(blocks[4])
		require.NoError(t, err)
		require.Nil(t, justifications)

		// Append a justification to block 5.
		backend.AppendJustification(blocks[5], runtime.Justification{
			ConsensusEngineID:    [4]byte{0, 0, 0, 1},
			EncodedJustification: []byte{42},
		})

		hash, err = insertBlock(
			t, backend, 6, blocks[5], nil, "", []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(6)}}, nil,
		)
		require.NoError(t, err)
		blocks = append(blocks, hash)

		// Pin block 5 so it gets loaded into the cache on prune
		err = backend.PinBlock(blocks[5])
		require.NoError(t, err)

		// Finalize block 6 so block 5 gets pruned. Since it is pinned both justifications should be
		// in memory.
		op = backend.beginOperation()
		err = backend.BeginStateOperation(op, blocks[6])
		require.NoError(t, err)
		err = op.MarkFinalized(blocks[6], nil)
		require.NoError(t, err)
		err = backend.CommitOperation(op)
		require.NoError(t, err)

		body, err = bc.Body(blocks[5])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(5)}}, body)
		expected := runtime.Justifications{
			*buildJustification(5),
			runtime.Justification{
				ConsensusEngineID:    [4]byte{0, 0, 0, 1},
				EncodedJustification: []byte{42},
			},
		}
		justifications, err = bc.Justifications(blocks[5])
		require.NoError(t, err)
		require.Equal(t, expected, justifications)
	})

	t.Run("pinned_blocks_on_finalize_with_fork", func(t *testing.T) {
		backend := NewTestBackend(t, BlocksPruningSome(1), 10)
		var blocks []hash.H256
		var prevHash hash.H256

		// Block tree:
		//   0 -> 1 -> 2 -> 3 -> 4
		for i := 0; i < 5; i++ {
			hash, err := insertBlock(t,
				backend,
				uint64(i),
				prevHash,
				nil,
				hash.H256(""),
				[]rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(i)}},
				nil,
			)
			require.NoError(t, err)
			blocks = append(blocks, hash)

			// Avoid block pruning.
			err = backend.PinBlock(blocks[i])
			require.NoError(t, err)

			prevHash = hash
		}

		// Insert a fork at the second block.
		// Block tree:
		//   0 -> 1 -> 2 -> 3 -> 4
		//        \ -> 2 -> 3
		forkHashRoot, err := insertBlock(t,
			backend,
			2,
			blocks[1],
			nil,
			hash.NewRandomH256(),
			[]rt_testing.ExtrinsicsWrapper[uint64]{{T: 2}},
			nil,
		)
		require.NoError(t, err)
		forkHash3, err := insertBlock(t,
			backend,
			3,
			forkHashRoot,
			nil,
			hash.NewRandomH256(),
			[]rt_testing.ExtrinsicsWrapper[uint64]{{T: 3}, {T: 11}},
			nil,
		)
		require.NoError(t, err)

		// Do not prune the fork hash.
		err = backend.PinBlock(forkHash3)
		require.NoError(t, err)

		op, err := backend.BeginOperation()
		require.NoError(t, err)
		err = backend.BeginStateOperation(op, blocks[4])
		require.NoError(t, err)
		err = op.MarkHead(blocks[4])
		require.NoError(t, err)
		err = backend.CommitOperation(op)
		require.NoError(t, err)

		for i := 1; i < 5; i++ {
			op, err := backend.BeginOperation()
			require.NoError(t, err)
			err = backend.BeginStateOperation(op, blocks[4])
			require.NoError(t, err)
			err = op.MarkFinalized(blocks[i], nil)
			require.NoError(t, err)
			err = backend.CommitOperation(op)
			require.NoError(t, err)
		}

		bc := backend.Blockchain()
		body, err := bc.Body(blocks[0])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(0)}}, body)
		body, err = bc.Body(blocks[1])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(1)}}, body)
		body, err = bc.Body(blocks[2])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(2)}}, body)
		body, err = bc.Body(blocks[3])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(3)}}, body)
		body, err = bc.Body(blocks[4])
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{{T: uint64(4)}}, body)
		// Check the fork hashes.
		body, err = bc.Body(forkHashRoot)
		require.NoError(t, err)
		require.Nil(t, body)
		body, err = bc.Body(forkHash3)
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{
			{T: 3},
			{T: 11},
		}, body)

		// Unpine all blocks, except the forked one
		for _, block := range blocks {
			backend.UnpinBlock(block)
		}

		body, err = bc.Body(blocks[0])
		require.NoError(t, err)
		require.Nil(t, body)
		body, err = bc.Body(blocks[1])
		require.NoError(t, err)
		require.Nil(t, body)
		body, err = bc.Body(blocks[2])
		require.NoError(t, err)
		require.Nil(t, body)
		body, err = bc.Body(blocks[3])
		require.NoError(t, err)
		require.Nil(t, body)

		body, err = bc.Body(forkHash3)
		require.NoError(t, err)
		require.Equal(t, []rt_testing.ExtrinsicsWrapper[uint64]{
			{T: 3},
			{T: 11},
		}, body)
		backend.UnpinBlock(forkHash3)
		body, err = bc.Body(forkHash3)
		require.NoError(t, err)
		require.Nil(t, body)
	})
}
