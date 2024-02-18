package db

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/db/columns"
	"github.com/ChainSafe/gossamer/internal/client/db/metakeys"
	statedb "github.com/ChainSafe/gossamer/internal/client/state-db"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/li1234yun/gods-generic/maps/linkedhashmap"
)

// / An extrinsic entry in the database.
// #[derive(Debug, Encode, Decode)]
//
//	enum DbExtrinsic<B: BlockT> {
//		/// Extrinsic that contains indexed data.
//		Indexed {
//			/// Hash of the indexed part.
//			hash: DbHash,
//			/// Extrinsic header.
//			header: Vec<u8>,
//		},
//		/// Complete extrinsic data.
//		Full(B::Extrinsic),
//	}
type dbExtrinsic interface{}

// / Extrinsic that contains indexed data.
type dbExtrinsicIndexed struct {
	/// Hash of the indexed part.
	Hash hash.H256
	/// Extrinsic header.
	Header [][]byte
}
type dbExtrinsicFull[E runtime.Extrinsic] struct {
	Extrinsic E
}

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

type metaUpdate[H, N any] struct {
	Hash        H
	Number      N
	IsBest      bool
	IsFinalized bool
	WithState   bool
}

type BlockchainDB[H runtime.Hash, N runtime.Number] struct {
	db                   database.Database[hash.H256]
	meta                 meta[H, N]
	metaMtx              sync.RWMutex
	leaves               api.LeafSet[H, N]
	leavesMtx            sync.RWMutex
	headerMetadataCache  blockchain.HeaderMetadataCache[H, N]
	headerCache          linkedhashmap.Map[H, runtime.Header[N, H]]
	headerCacheMtx       sync.Mutex
	pinnedBlocksCache    pinnedBlocksCache[H]
	pinnedBlocksCacheMtx sync.RWMutex
}

func NewBlockchainDB[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](db database.Database[hash.H256]) (*BlockchainDB[H, N], error) {
	meta, err := readMeta[H, N, *generic.Header[N, H, Hasher]](db, uint32(columns.Header))
	if err != nil {
		return nil, err
	}
	leaves, err := api.NewLeafSetFromDB[H, N](db, uint32(columns.Meta), metakeys.LeafPrefix)
	if err != nil {
		return nil, err
	}
	return &BlockchainDB[H, N]{
		db:                  db,
		leaves:              leaves,
		meta:                meta,
		headerMetadataCache: blockchain.NewHeaderMetadataCache[H, N](),
		headerCache:         *linkedhashmap.New[H, runtime.Header[N, H]](),
		pinnedBlocksCache:   newPinnedBlocksCache[H](),
	}, nil
}

func (bdb *BlockchainDB[H, N]) updateMeta(update metaUpdate[H, N]) {
	bdb.metaMtx.Lock()
	defer bdb.metaMtx.Unlock()
	if update.Number == 0 {
		bdb.meta.GenesisHash = update.Hash
	}

	if update.IsBest {
		bdb.meta.BestNumber = update.Number
		bdb.meta.BestHash = update.Hash
	}

	if update.IsFinalized {
		if update.WithState {
			bdb.meta.FinalizedState = &struct {
				Hash   H
				Number N
			}{update.Hash, update.Number}
		}
		bdb.meta.FinalizedNumber = update.Number
		bdb.meta.FinalizedHash = update.Hash
	}
}

func (bdb *BlockchainDB[H, N]) updateBlockGap(gap *[2]N) {
	bdb.metaMtx.Lock()
	defer bdb.metaMtx.Unlock()
	bdb.meta.BlockGap = gap
}

// / Empty the cache of pinned items.
func (bdb *BlockchainDB[H, N]) clearPinningCache() {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	bdb.pinnedBlocksCache.Clear()
}

// / Load a justification into the cache of pinned items.
// / Reference count of the item will not be increased. Use this
// / to load values for items into the cache which have already been pinned.
func (bdb *BlockchainDB[H, N]) insertJustifcationsIfPinned(hash H, justification runtime.Justification) {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	if !bdb.pinnedBlocksCache.Contains(hash) {
		return
	}

	justifications := runtime.Justifications{justification}
	bdb.pinnedBlocksCache.InsertJustifications(hash, &justifications)
}

// / Load a justification from the db into the cache of pinned items.
// / Reference count of the item will not be increased. Use this
// / to load values for items into the cache which have already been pinned.
func (bdb *BlockchainDB[H, N]) insertPersistedJustificationsIfPinned(hash H) error {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	if !bdb.pinnedBlocksCache.Contains(hash) {
		return nil
	}

	justifications, err := bdb.justificationsUncached(hash)
	if err != nil {
		return err
	}
	bdb.pinnedBlocksCache.InsertJustifications(hash, justifications)
	return nil
}

// / Load a block body from the db into the cache of pinned items.
// / Reference count of the item will not be increased. Use this
// / to load values for items items into the cache which have already been pinned.
func (bdb *BlockchainDB[H, N]) insertPersistedBodyIfPinned(hash H) error {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	if !bdb.pinnedBlocksCache.Contains(hash) {
		return nil
	}

	body, err := bdb.bodyUncached(hash)
	if err != nil {
		return err
	}
	bdb.pinnedBlocksCache.InsertBody(hash, body)
	return nil
}

// / Bump reference count for pinned item.
func (bdb *BlockchainDB[H, N]) bumpRef(hash H) {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	bdb.pinnedBlocksCache.Pin(hash)
}

// / Decrease reference count for pinned item and remove if reference count is 0.
func (bdb *BlockchainDB[H, N]) unpin(hash H) {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	bdb.pinnedBlocksCache.Unpin(hash)
}

func (bdb *BlockchainDB[H, N]) justificationsUncached(hash H) (*runtime.Justifications, error) {
	blockID := generic.BlockID(generic.BlockIDHash[H]{Inner: hash})
	justificationsBytes, err := readDB[H, N](bdb.db, uint32(columns.KeyLookup), uint32(columns.Justifications), blockID)
	if err != nil {
		return nil, err
	}
	if justificationsBytes != nil {
		var justifications runtime.Justifications
		err := scale.Unmarshal(*justificationsBytes, &justifications)
		if err != nil {
			return nil, err
		}
		return &justifications, nil
	}
	return nil, nil
}

func (bdb *BlockchainDB[H, N]) bodyUncached(hash H) (*[]runtime.Extrinsic, error) {
	blockID := generic.BlockID(generic.BlockIDHash[H]{Inner: hash})
	bodyBytes, err := readDB[H, N](bdb.db, uint32(columns.KeyLookup), uint32(columns.Body), blockID)
	if err != nil {
		return nil, err
	}
	if bodyBytes != nil {
		var body []runtime.Extrinsic
		err := scale.Unmarshal(*bodyBytes, &body)
		if err != nil {
			return nil, err
		}
		return &body, nil
	}

	indexBytes, err := readDB[H, N](bdb.db, uint32(columns.KeyLookup), uint32(columns.BodyIndex), blockID)
	if err != nil {
		return nil, err
	}
	if indexBytes != nil {

	}
	panic("unimplemented")
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
