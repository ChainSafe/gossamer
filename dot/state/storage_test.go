package state

import (
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	rt "github.com/ChainSafe/gossamer/lib/runtime"
	runtime "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	log "github.com/ChainSafe/log15"

	"github.com/stretchr/testify/require"
)

func newTestStorageState(t *testing.T) *StorageState {
	db := NewInMemoryDB(t)
	bs := newTestBlockState(t, testGenesisHeader)

	s, err := NewStorageState(db, bs, trie.NewEmptyTrie(), pruner.Config{})
	require.NoError(t, err)
	return s
}

//TODO: optimize this method to bare minimum body for the StorageState instance
func newFileDbTestStorageState(t *testing.T) *StorageState {
	config := Config{
		Path:     utils.NewTestBasePath(t, "flie_db"),
		LogLevel: log.LvlInfo,
	}
	stateSrvc := NewService(config)

	gen, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)

	err := stateSrvc.Initialise(gen, genesisHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	rt, err := stateSrvc.CreateGenesisRuntime(genTrie, gen)
	require.NoError(t, err)

	err = loadTestBlocks(t, genesisHeader.Hash(), stateSrvc.Block, rt)
	require.NoError(t, err)

	t.Cleanup(func() {
		stateSrvc.Stop()
	})
	return stateSrvc.Storage
}

func loadTestBlocks(t *testing.T, gh common.Hash, bs *BlockState, rt rt.Instance) error {
	// Create header
	header0 := &types.Header{
		Number:     big.NewInt(0),
		Digest:     types.NewDigest(),
		ParentHash: gh,
		StateRoot:  trie.EmptyHash,
	}
	// Create blockHash
	sampleBodyBytes := *types.NewBody([]types.Extrinsic{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}})
	blockHash0 := header0.Hash()
	block0 := &types.Block{
		Header: *header0,
		Body:   sampleBodyBytes,
	}

	err := bs.AddBlock(block0)
	if err != nil {
		return err
	}

	bs.StoreRuntime(block0.Header.Hash(), rt)

	// Create header & blockData for block 1
	digest := types.NewDigest()
	err = digest.Add(*types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest())
	require.NoError(t, err)
	header1 := &types.Header{
		Number:     big.NewInt(1),
		Digest:     digest,
		ParentHash: blockHash0,
		StateRoot:  trie.EmptyHash,
	}

	block1 := &types.Block{
		Header: *header1,
		Body:   sampleBodyBytes,
	}

	// Add the block1 to the DB
	err = bs.AddBlock(block1)
	if err != nil {
		return err
	}

	bs.StoreRuntime(block1.Header.Hash(), rt)

	return nil
}

func TestStorage_StoreAndLoadTrie(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	root, err := ts.Root()
	require.NoError(t, err)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 100)

	trie, err := storage.LoadFromDB(root)
	require.NoError(t, err)
	ts2, err := runtime.NewTrieState(trie)
	require.NoError(t, err)
	new := ts2.Snapshot()
	require.Equal(t, ts.Trie(), new)
}

func TestStorage_StorageChild(t *testing.T) {
	storage := newFileDbTestStorageState(t)
	tr, err := storage.TrieState(nil)
	require.NoError(t, err)

	childTr := trie.NewEmptyTrie()
	childTr.Put([]byte(":child_first"), []byte(":child_first_value"))
	childTr.Put([]byte(":child_second"), []byte(":child_second_value"))
	err = tr.SetChild([]byte(":child_storage_key"), childTr)
	require.NoError(t, err)

	err = storage.StoreTrie(tr, nil)
	require.NoError(t, err)

	stateRoot, err := tr.Root()
	require.NoError(t, err)

	bb, err := storage.blockState.BestBlock()
	require.NoError(t, err)

	//update the database header
	b := &types.Block{
		Header: types.Header{
			ParentHash: bb.Header.Hash(),
			Number:     big.NewInt(0).Add(big.NewInt(1), bb.Header.Number),
			StateRoot:  stateRoot,
		},
		Body: types.Body{},
	}
	err = storage.blockState.AddBlock(b)
	require.NoError(t, err)

	hash := storage.blockState.BestBlockHash()
	tests := []struct {
		expect   uint64
		keyChild []byte
		entry    []byte
	}{
		{
			expect:   uint64(len([]byte(":child_first_value"))),
			entry:    []byte(":child_first"),
			keyChild: []byte(":child_storage_key"),
		},
		{
			expect:   uint64(len([]byte(":child_second_value"))),
			entry:    []byte(":child_second"),
			keyChild: []byte(":child_storage_key"),
		},
	}

	for _, test := range tests {
		var res uint64

		stateRoot, err := storage.GetStateRootFromBlock(&hash)
		require.NoError(t, err)

		item, err := storage.GetStorageFromChild(stateRoot, test.keyChild, test.entry)
		require.NoError(t, err)

		if item != nil {
			res = uint64(len(item))
		}
		require.Equal(t, test.expect, res)
	}
}

func TestStorage_GetStorageByBlockHash(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	key := []byte("testkey")
	value := []byte("testvalue")
	ts.Set(key, value)

	root, err := ts.Root()
	require.NoError(t, err)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	body, err := types.NewBodyFromBytes([]byte{})
	require.NoError(t, err)

	block := &types.Block{
		Header: types.Header{
			ParentHash: testGenesisHeader.Hash(),
			Number:     big.NewInt(1),
			StateRoot:  root,
		},
		Body: *body,
	}
	err = storage.blockState.AddBlock(block)
	require.NoError(t, err)

	hash := block.Header.Hash()
	res, err := storage.GetStorageByBlockHash(&hash, key)
	require.NoError(t, err)
	require.Equal(t, value, res)
}

func TestStorage_TrieState(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)
	ts.Set([]byte("noot"), []byte("washere"))

	root, err := ts.Root()
	require.NoError(t, err)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 100)

	// get trie from db
	storage.tries.Delete(root)
	ts3, err := storage.TrieState(&root)
	require.NoError(t, err)
	require.Equal(t, ts.Trie().MustHash(), ts3.Trie().MustHash())
}

func TestStorage_LoadFromDB(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	trieKV := []struct {
		key   []byte
		value []byte
	}{{},
		{[]byte("key1"), []byte("value1")},
		{[]byte("key2"), []byte("value2")},
		{[]byte("xyzKey1"), []byte("xyzValue1")},
	}

	for _, kv := range trieKV {
		ts.Set(kv.key, kv.value)
	}

	root, err := ts.Root()
	require.NoError(t, err)

	// Write trie to disk.
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	// Clear trie from cache and fetch data from disk.
	storage.tries.Delete(root)

	data, err := storage.GetStorage(&root, trieKV[0].key)
	require.NoError(t, err)
	require.Equal(t, trieKV[0].value, data)

	storage.tries.Delete(root)

	prefixKeys, err := storage.GetKeysWithPrefix(&root, []byte("ke"))
	require.NoError(t, err)
	require.Equal(t, 2, len(prefixKeys))

	storage.tries.Delete(root)

	entries, err := storage.Entries(&root)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
}

func syncMapLen(m *sync.Map) int {
	l := 0
	m.Range(func(_, _ interface{}) bool {
		l++
		return true
	})
	return l
}

func TestStorage_StoreTrie_Syncing(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	key := []byte("testkey")
	value := []byte("testvalue")
	ts.Set(key, value)

	storage.SetSyncing(true)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)
	require.Equal(t, 1, syncMapLen(storage.tries))
}

func TestStorage_StoreTrie_NotSyncing(t *testing.T) {
	storage := newTestStorageState(t)
	ts, err := storage.TrieState(&trie.EmptyHash)
	require.NoError(t, err)

	key := []byte("testkey")
	value := []byte("testvalue")
	ts.Set(key, value)

	storage.SetSyncing(false)
	err = storage.StoreTrie(ts, nil)
	require.NoError(t, err)
	require.Equal(t, 2, syncMapLen(storage.tries))
}
