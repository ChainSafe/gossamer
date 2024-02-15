package db

import (
	"sync"

	statedb "github.com/ChainSafe/gossamer/internal/client/state-db"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// / Disk backend.
// /
// / Disk backend keeps data in a key-value store. In archive mode, trie nodes are kept from all
// / blocks. Otherwise, trie nodes are kept only from some recent blocks.
// pub struct Backend<Block: BlockT> {
type Backend[H runtime.Hash, N runtime.Number] struct {
	// storage: Arc<StorageDb<Block>>,
	// offchain_storage: offchain::LocalStorage,
	// blockchain: BlockchainDb<Block>,
	// canonicalization_delay: u64,
	// import_lock: Arc<RwLock<()>>,
	// is_archive: bool,
	// blocks_pruning: BlocksPruning,
	// io_stats: FrozenForDuration<(kvdb::IoStats, StateUsageInfo)>,
	// state_usage: Arc<StateUsageStats>,
	// genesis_state: RwLock<Option<Arc<DbGenesisStorage<Block>>>>,
	// shared_trie_cache: Option<sp_trie::cache::SharedTrieCache<HashFor<Block>>>,
}

type BlockchainDB[H runtime.Hash, N runtime.Number] struct {
	db      database.Database[hash.H256]
	meta    meta[N, H]
	metaMtx sync.RWMutex
}

type storageDB[H runtime.Hash] struct {
	DB database.Database[hash.H256]
	// TODO: use generic param for db key?
	StateDB    statedb.StateDB[H, hash.H256]
	prefixKeys bool
}

type Prefix struct {
	Key    []byte
	Padded *byte
}

func NewPrefixedKey[H runtime.Hash](key H, prefix Prefix) []byte {
	prefixedKey := prefix.Key
	if prefix.Padded != nil {
		prefixedKey = append(prefixedKey, *prefix.Padded)
	}
	prefixedKey = append(prefixedKey, key.Bytes()...)
	return prefixedKey
}

/// Derive a database key from hash value of the node (key) and  the node prefix.
// pub fn prefixed_key<H: KeyHasher>(key: &H::Out, prefix: Prefix) -> Vec<u8> {
// 	let mut prefixed_key = Vec::with_capacity(key.as_ref().len() + prefix.0.len() + 1);
// 	prefixed_key.extend_from_slice(prefix.0);
// 	if let Some(last) = prefix.1 {
// 		prefixed_key.push(last);
// 	}
// 	prefixed_key.extend_from_slice(key.as_ref());
// 	prefixed_key
// }

// func (sdb *storageDB[H, N]) get(key H, prefix Prefix) ([]byte, error) {
// 	if sdb.prefixKeys {
// 		key := NewPrefixedKey[H](key, prefix)
// 		val, err := sdb.StateDB.Get(key, sdb)

// 	}
// 	return nil, nil
// }
