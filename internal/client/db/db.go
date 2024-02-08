package db

import (
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

type storageDB[H runtime.Hash, N runtime.Number] struct {
	DB database.Database[H]
}
