package db

import (
	"errors"
	"fmt"
	"log"
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
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/li1234yun/gods-generic/maps/linkedhashmap"
)

// / An extrinsic entry in the database.
type dbExtrinsic[E runtime.Extrinsic] struct {
	inner any
}

type dbExtrinsicValues[E runtime.Extrinsic] interface {
	dbExtrinsicIndexed | dbExtrinsicFull[E]
}

func setVote[E runtime.Extrinsic, Value dbExtrinsicValues[E]](mvdt *dbExtrinsic[E], value Value) {
	mvdt.inner = value
}

func (mvdt *dbExtrinsic[E]) SetValue(value any) (err error) {
	switch value := value.(type) {
	case dbExtrinsicIndexed:
		setVote[E](mvdt, value)
		return
	case dbExtrinsicFull[E]:
		setVote[E](mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt dbExtrinsic[E]) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case dbExtrinsicIndexed:
		return 0, mvdt.inner, nil
	case dbExtrinsicFull[E]:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt dbExtrinsic[E]) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt dbExtrinsic[E]) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(dbExtrinsicIndexed), nil
	case 1:
		return *new(dbExtrinsicFull[E]), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// / Extrinsic that contains indexed data.
type dbExtrinsicIndexed struct {
	/// Hash of the indexed part.
	Hash hash.H256
	/// Extrinsic header.
	Header []byte
}
type dbExtrinsicFull[E runtime.Extrinsic] struct {
	Extrinsic E
}

// / DB-backed patricia trie state, transaction type is an overlay of changes to commit.
// pub type DbState<B> =
//
//	sp_state_machine::TrieBackend<Arc<dyn sp_state_machine::Storage<HashingFor<B>>>, HashingFor<B>>;
type DBState = statemachine.TrieBackend

// / A reference tracking state.
// /
// / It makes sure that the hash we are using stays pinned in storage
// / until this structure is dropped.
// pub struct RefTrackingState<Block: BlockT> {
type RefTrackingState struct {
	// state: DbState<Block>,
	state DBState
	// storage: Arc<StorageDb<Block>>,
	// parent_hash: Option<Block::Hash>,
}

// / Database settings.
type DatabaseSettings struct {
	/// The maximum trie cache size in bytes.
	///
	/// If `None` is given, the cache is disabled.
	TrieCacheMaximumSize *uint
	/// Requested state pruning mode.
	StatePruning statedb.PruningMode
	/// Where to find the database.
	Source DatabaseSource
	/// Block pruning mode.
	///
	/// NOTE: only finalized blocks are subject for removal!
	BlocksPruning BlocksPruning
}

// / Block pruning settings.
type BlocksPruning any
type BlocksPruningValues interface {
	BlocksPruningKeepAll | BlocksPruningKeepFinalized | BlocksPruningSome
}

// / Keep full block history, of every block that was ever imported.
type BlocksPruningKeepAll struct{}

// / Keep full finalized block history.
type BlocksPruningKeepFinalized struct{}

// / Keep N recent finalized blocks.
type BlocksPruningSome uint32

// / Where to find the database..
// / NOTE: only uses a custom already-open database.
type DatabaseSource struct {
	/// the handle to the custom storage
	DB database.Database[hash.H256]
	/// if set, the `create` flag will be required to open such datasource
	RequireCreateFlag bool
}

// wrapper that implements trait required for state_db
type stateMetaDB struct {
	db database.Database[hash.H256]
}

func (smdb stateMetaDB) GetMeta(key []byte) (*statedb.DBValue, error) {
	val := smdb.db.Get(database.ColumnID(columns.StateMeta), key)
	if val == nil {
		return nil, nil
	}
	dbVal := statedb.DBValue(*val)
	return &dbVal, nil
}

type metaUpdate[H, N any] struct {
	Hash        H
	Number      N
	IsBest      bool
	IsFinalized bool
	WithState   bool
}

type BlockchainDB[H runtime.Hash, N runtime.Number, E runtime.Extrinsic, Header runtime.Header[N, H]] struct {
	db                   database.Database[hash.H256]
	meta                 meta[H, N]
	metaMtx              sync.RWMutex
	leaves               api.LeafSet[H, N]
	leavesMtx            sync.RWMutex
	headerMetadataCache  blockchain.HeaderMetadataCache[H, N]
	headerCache          linkedhashmap.Map[H, *runtime.Header[N, H]]
	headerCacheMtx       sync.Mutex
	pinnedBlocksCache    pinnedBlocksCache[H]
	pinnedBlocksCacheMtx sync.RWMutex
}

func NewBlockchainDB[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H], E runtime.Extrinsic, Header runtime.Header[N, H]](
	db database.Database[hash.H256],
) (*BlockchainDB[H, N, E, Header], error) {
	meta, err := readMeta[H, N, *generic.Header[N, H, Hasher]](db, uint32(columns.Header))
	if err != nil {
		return nil, err
	}
	leaves, err := api.NewLeafSetFromDB[H, N](db, uint32(columns.Meta), metakeys.LeafPrefix)
	if err != nil {
		return nil, err
	}
	return &BlockchainDB[H, N, E, Header]{
		db:                  db,
		leaves:              leaves,
		meta:                meta,
		headerMetadataCache: blockchain.NewHeaderMetadataCache[H, N](),
		headerCache:         *linkedhashmap.New[H, *runtime.Header[N, H]](),
		pinnedBlocksCache:   newPinnedBlocksCache[H](),
	}, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) updateMeta(update metaUpdate[H, N]) {
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

func (bdb *BlockchainDB[H, N, E, Header]) updateBlockGap(gap *[2]N) {
	bdb.metaMtx.Lock()
	defer bdb.metaMtx.Unlock()
	bdb.meta.BlockGap = gap
}

// / Empty the cache of pinned items.
func (bdb *BlockchainDB[H, N, E, Header]) clearPinningCache() {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	bdb.pinnedBlocksCache.Clear()
}

// / Load a justification into the cache of pinned items.
// / Reference count of the item will not be increased. Use this
// / to load values for items into the cache which have already been pinned.
func (bdb *BlockchainDB[H, N, E, Header]) insertJustifcationsIfPinned(hash H, justification runtime.Justification) {
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
func (bdb *BlockchainDB[H, N, E, Header]) insertPersistedJustificationsIfPinned(hash H) error {
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
func (bdb *BlockchainDB[H, N, E, Header]) insertPersistedBodyIfPinned(hash H) error {
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
func (bdb *BlockchainDB[H, N, E, Header]) bumpRef(hash H) {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	bdb.pinnedBlocksCache.Pin(hash)
}

// / Decrease reference count for pinned item and remove if reference count is 0.
func (bdb *BlockchainDB[H, N, E, Header]) unpin(hash H) {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	bdb.pinnedBlocksCache.Unpin(hash)
}

func (bdb *BlockchainDB[H, N, E, Header]) justificationsUncached(hash H) (*runtime.Justifications, error) {
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

func (bdb *BlockchainDB[H, N, E, Header]) bodyUncached(hash H) (*[]runtime.Extrinsic, error) {
	blockID := generic.BlockID(generic.BlockIDHash[H]{Inner: hash})
	bodyBytes, err := readDB[H, N](bdb.db, uint32(columns.KeyLookup), uint32(columns.Body), blockID)
	if err != nil {
		return nil, err
	}
	if bodyBytes != nil {
		var extrinsics []E
		err := scale.Unmarshal(*bodyBytes, &extrinsics)
		if err != nil {
			return nil, err
		}
		var body []runtime.Extrinsic
		for _, e := range extrinsics {
			body = append(body, e)
		}
		return &body, nil
	}

	indexBytes, err := readDB[H, N](bdb.db, uint32(columns.KeyLookup), uint32(columns.BodyIndex), blockID)
	if err != nil {
		return nil, err
	}
	if indexBytes == nil {
		return nil, nil
	}
	var index []dbExtrinsic[E]
	err = scale.Unmarshal(*indexBytes, index)
	if err != nil {
		return nil, err
	}
	var body []runtime.Extrinsic
	for _, ex := range index {
		dbex, err := ex.Value()
		if err != nil {
			return nil, err
		}
		switch dbex := dbex.(type) {
		case dbExtrinsicIndexed:
			t := bdb.db.Get(database.ColumnID(columns.Transaction), hash.Bytes())
			if t != nil {
				input := newJoinInput(dbex.Header, *t)
				var ex E
				err := scale.Unmarshal(input.Bytes(), &ex)
				if err != nil {
					return nil, fmt.Errorf("Error decoding indexed extrinsic: %w", err)
				}
				body = append(body, ex)
			} else {
				return nil, fmt.Errorf("Missing indexed transaction %v", hash)
			}
		case dbExtrinsicFull[E]:
			body = append(body, dbex.Extrinsic)
		}
	}
	return &body, nil
}

const cacheHeaders = 8

func (bdb *BlockchainDB[H, N, E, Header]) cacheHeader(hash H, header *runtime.Header[N, H]) {
	bdb.headerCache.Put(hash, header)
	for bdb.headerCache.Size() > cacheHeaders {
		iterator := bdb.headerCache.Iterator()
		if !iterator.First() {
			panic("huh?")
		}
		bdb.headerCache.Remove(iterator.Key())
	}
}

func (bdb *BlockchainDB[H, N, E, Header]) Header(hash H) (*runtime.Header[N, H], error) {
	bdb.headerCacheMtx.Lock()
	defer bdb.headerCacheMtx.Unlock()
	val, ok := bdb.headerCache.Get(hash)
	if ok {
		// TODO: create issue to fork linkedhashmap, and add cache.get_refresh(&hash)
		bdb.headerCache.Remove(hash)
		bdb.headerCache.Put(hash, val)
		return val, nil
	}
	header, err := readHeader[H, N, Header](
		bdb.db,
		uint32(columns.KeyLookup),
		uint32(columns.Header),
		generic.BlockIDHash[H]{Inner: hash},
	)
	if err != nil {
		return nil, err
	}
	bdb.cacheHeader(hash, header)
	return header, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) Info() blockchain.Info[H, N] {
	bdb.metaMtx.RLock()
	defer bdb.metaMtx.RUnlock()
	return blockchain.Info[H, N]{
		BestHash:        bdb.meta.BestHash,
		BestNumber:      bdb.meta.BestNumber,
		GenesisHash:     bdb.meta.GenesisHash,
		FinalizedHash:   bdb.meta.FinalizedHash,
		FinalizedNumber: bdb.meta.FinalizedNumber,
		FinalizedState:  bdb.meta.FinalizedState,
		NumberLeaves:    bdb.leaves.Count(),
		BlockGap:        bdb.meta.BlockGap,
	}
}

func (bdb *BlockchainDB[H, N, E, Header]) Status(hash H) (blockchain.BlockStatus, error) {
	header, err := bdb.Header(hash)
	if err != nil {
		return 0, err
	}
	if header != nil {
		return blockchain.BlockStatusInChain, nil
	}
	return blockchain.BlockStatusUnknown, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) Number(hash H) (*N, error) {
	meta, err := bdb.HeaderMetadata(hash)
	if err != nil {
		return nil, err
	}
	return &meta.Number, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) Hash(number N) (*H, error) {
	header, err := readHeader[H, N, Header](
		bdb.db,
		uint32(columns.KeyLookup),
		uint32(columns.Header),
		generic.BlockIDNumber[N]{Inner: number},
	)
	if err != nil {
		return nil, err
	}
	if header == nil {
		return nil, nil
	}
	h := (*header).Hash()
	return &h, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) BlockHashFromID(id generic.BlockID) (*H, error) {
	switch id := id.(type) {
	case generic.BlockIDHash[H]:
		return &id.Inner, nil
	case generic.BlockIDNumber[N]:
		return bdb.Hash(id.Inner)
	default:
		panic("wtf?")
	}
}

func (bdb *BlockchainDB[H, N, E, Header]) BlockNumberFromID(id generic.BlockID) (*N, error) {
	switch id := id.(type) {
	case generic.BlockIDHash[H]:
		return bdb.Number(id.Inner)
	case generic.BlockIDNumber[N]:
		return &id.Inner, nil
	default:
		panic("wtf?")
	}
}

func (bdb *BlockchainDB[H, N, E, Header]) ExpectHeader(hash H) (runtime.Header[N, H], error) {
	header, err := bdb.Header(hash)
	if err != nil {
		return nil, err
	}
	if header == nil {
		return nil, fmt.Errorf("expect header: %v", hash)
	}
	return *header, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) ExpectBlockNumberFromID(id generic.BlockID) (N, error) {
	number, err := bdb.BlockNumberFromID(id)
	if err != nil {
		return 0, err
	}
	if number == nil {
		return 0, fmt.Errorf("expect block number from id: %v", id)
	}
	return *number, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) ExpectBlockHashFromID(id generic.BlockID) (H, error) {
	hash, err := bdb.BlockHashFromID(id)
	if err != nil {
		return *new(H), err
	}
	if hash == nil {
		return *new(H), fmt.Errorf("expect block number from id: %v", id)
	}
	return *hash, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) Body(hash H) (*[]runtime.Extrinsic, error) {
	bdb.pinnedBlocksCacheMtx.RLock()
	defer bdb.pinnedBlocksCacheMtx.RUnlock()
	body := bdb.pinnedBlocksCache.Body(hash)
	if body != nil {
		return body, nil
	}

	return bdb.bodyUncached(hash)
}

func (bdb *BlockchainDB[H, N, E, Header]) Justifications(hash H) (*runtime.Justifications, error) {
	bdb.pinnedBlocksCacheMtx.RLock()
	defer bdb.pinnedBlocksCacheMtx.RUnlock()
	justifications := bdb.pinnedBlocksCache.Justifications(hash)
	if justifications != nil {
		return justifications, nil
	}

	return bdb.justificationsUncached(hash)
}

func (bdb *BlockchainDB[H, N, E, Header]) LastFinalized() (H, error) {
	bdb.metaMtx.RLock()
	defer bdb.metaMtx.RUnlock()
	return bdb.meta.FinalizedHash, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) Leaves() ([]H, error) {
	bdb.leavesMtx.RLock()
	defer bdb.leavesMtx.RUnlock()
	return bdb.leaves.Hashes(), nil
}

func (bdb *BlockchainDB[H, N, E, Header]) DisplacedLeavesAfterFinalizing(blockNumber N) ([]H, error) {
	bdb.leavesMtx.RLock()
	defer bdb.leavesMtx.RUnlock()
	return bdb.leaves.DisplacedByFinalHeight(blockNumber).Leaves(), nil
}

func (bdb *BlockchainDB[H, N, E, Header]) Children(parentHash H) ([]H, error) {
	return readChildren[H, H](bdb.db, uint32(columns.Meta), metakeys.ChildrenPrefix, parentHash)
}

func (bdb *BlockchainDB[H, N, E, Header]) LongestContaining(baseHash H, importLock *sync.RWMutex) (*H, error) {
	baseHeader, err := bdb.Header(baseHash)
	if err != nil {
		return nil, err
	}
	if baseHeader == nil {
		return nil, nil
	}

	var getLeaves = func() ([]H, error) {
		// ensure no blocks are imported during this code block.
		// an import could trigger a reorg which could change the canonical chain.
		// we depend on the canonical chain staying the same during this code block.
		importLock.RLock()
		defer importLock.RUnlock()
		info := bdb.Info()
		if info.FinalizedNumber > (*baseHeader).Number() {
			// `baseHeader` is on a dead fork.
			return nil, nil
		}
		return bdb.Leaves()
	}

	leaves, err := getLeaves()
	if err != nil {
		return nil, err
	}

	// for each chain. longest chain first. shortest last
	// for leaf_hash in leaves {
	for _, leafHash := range leaves {
		currentHash := leafHash
		// go backwards through the chain (via parent links)
		for {
			if currentHash == baseHash {
				return &leafHash, nil
			}

			currentHeader, err := bdb.Header(currentHash)
			if err != nil {
				return nil, err
			}
			if currentHeader == nil {
				return nil, fmt.Errorf("Failed to get header for hash %v", currentHash)
			}

			if (*currentHeader).Number() < (*baseHeader).Number() {
				break
			}

			currentHash = (*currentHeader).ParentHash()
		}
	}

	// header may be on a dead fork -- the only leaves that are considered are
	// those which can still be finalized.
	//
	// FIXME substrate issue #1558 only issue this warning when not on a dead fork
	log.Printf("WARN: Block %v exists in chain but not found when following all leaves backwards\n", baseHash)
	return nil, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) IndexedTransaction(hash H) (*[]byte, error) {
	return bdb.db.Get(database.ColumnID(columns.Transaction), hash.Bytes()), nil
}

func (bdb *BlockchainDB[H, N, E, Header]) HasIndexedTransaction(hash H) (bool, error) {
	return bdb.db.Contains(database.ColumnID(columns.Transaction), hash.Bytes()), nil
}

func (bdb *BlockchainDB[H, N, E, Header]) BlockIndexedBody(hash H) (*[][]byte, error) {
	bodyBytes, err := readDB[H, N](bdb.db, uint32(columns.KeyLookup), uint32(columns.BodyIndex), generic.BlockIDHash[H]{Inner: hash})
	if err != nil {
		return nil, err
	}
	if bodyBytes == nil {
		return nil, err
	}
	index := make([]dbExtrinsic[E], 0)
	err = scale.Unmarshal(*bodyBytes, &index)
	if err != nil {
		return nil, fmt.Errorf("Error decoding body list %w", err)
	}
	var transactions [][]byte
	for _, ex := range index {
		hash, err := ex.Value()
		if err != nil {
			return nil, fmt.Errorf("Error decoding body list %w", err)
		}
		indexed, ok := hash.(dbExtrinsicIndexed)
		if !ok {
			continue
		}
		t := bdb.db.Get(database.ColumnID(columns.Transaction), indexed.Hash.Bytes())
		if t == nil {
			return nil, fmt.Errorf("Missing indexed transaction %v", hash)
		}
		transactions = append(transactions, *t)
	}
	return &transactions, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) HeaderMetadata(hash H) (blockchain.CachedHeaderMetadata[H, N], error) {
	meta := bdb.headerMetadataCache.HeaderMetadata(hash)
	if meta != nil {
		return *meta, nil
	}
	header, err := bdb.Header(hash)
	if err != nil {
		return blockchain.CachedHeaderMetadata[H, N]{}, err
	}
	if header == nil {
		return blockchain.CachedHeaderMetadata[H, N]{}, fmt.Errorf("Header was not found in the database: %v\n", hash)
	}
	headerMetadata := blockchain.NewCachedHeaderMetadata(*header)
	bdb.headerMetadataCache.InsertHeaderMetadata(headerMetadata.Hash, headerMetadata)
	return headerMetadata, nil
}

func (bdb *BlockchainDB[H, N, E, Header]) InsertHeaderMetadata(hash H, metadata blockchain.CachedHeaderMetadata[H, N]) {
	bdb.headerMetadataCache.InsertHeaderMetadata(hash, metadata)
}

func (bdb *BlockchainDB[H, N, E, Header]) RemoveHeaderMetadata(hash H) {
	bdb.headerCacheMtx.Lock()
	defer bdb.headerCacheMtx.Unlock()
	bdb.headerCache.Remove(hash)
	bdb.headerMetadataCache.RemoveHeaderMetadata(hash)
}

// / Database transaction
// pub struct BlockImportOperation<Block: BlockT> {
type BlockImportOperation struct {
	// old_state: RecordStatsState<RefTrackingState<Block>, Block>,
	oldState RefTrackingState // wip
	// db_updates: PrefixedMemoryDB<HashingFor<Block>>,
	// storage_updates: StorageCollection,
	// child_storage_updates: ChildStorageCollection,
	// offchain_storage_updates: OffchainChangesCollection,
	// pending_block: Option<PendingBlock<Block>>,
	// aux_ops: Vec<(Vec<u8>, Option<Vec<u8>>)>,
	// finalized_blocks: Vec<(Block::Hash, Option<Justification>)>,
	// set_head: Option<Block::Hash>,
	// commit_state: bool,
	// index_ops: Vec<IndexOperation>,
}

type storageDB[H runtime.Hash] struct {
	DB database.Database[hash.H256]
	// TODO: use generic param for db key?
	StateDB    statedb.StateDB[H, hash.H256]
	prefixKeys bool
}

// / Disk backend.
// /
// / Disk backend keeps data in a key-value store. In archive mode, trie nodes are kept from all
// / blocks. Otherwise, trie nodes are kept only from some recent blocks.
type Backend[H runtime.Hash, N runtime.Number, E runtime.Extrinsic, Header runtime.Header[N, H]] struct {
	// storage: Arc<StorageDb<Block>>,
	storage storageDB[H]
	// offchain_storage: offchain::LocalStorage,
	// blockchain: BlockchainDb<Block>,
	blockchain BlockchainDB[H, N, E, Header]
	// canonicalization_delay: u64,
	canonicalizationDelay uint64
	// import_lock: Arc<RwLock<()>>,
	importLock sync.RWMutex
	// is_archive: bool,
	isArchive bool
	// blocks_pruning: BlocksPruning,
	blocksPruning BlocksPruning
	// io_stats: FrozenForDuration<(kvdb::IoStats, StateUsageInfo)>,
	// state_usage: Arc<StateUsageStats>,
	// genesis_state: RwLock<Option<Arc<DbGenesisStorage<Block>>>>,
	// shared_trie_cache: Option<sp_trie::cache::SharedTrieCache<HashFor<Block>>>,
}

// / Create a new instance of database backend.
// /
// / The pruning window is how old a block must be before the state is pruned.
func NewBackend[H runtime.Hash, N runtime.Number, E runtime.Extrinsic, Hasher runtime.Hasher[H], Header runtime.Header[N, H]](
	dbConfig DatabaseSettings,
	canonicalizationDelay uint64,
) (Backend[H, N, E, Header], error) {
	dbSource := dbConfig.Source

	var (
		needsInit bool
		db        database.Database[hash.H256]
	)
	db, err := openDatabase(dbSource, false)
	if err != nil {
		if errors.Is(err, errDoesNotExist) {
			db, err = openDatabase(dbSource, true)
			if err != nil {
				return Backend[H, N, E, Header]{}, err
			}
			needsInit = true
		} else {
			return Backend[H, N, E, Header]{}, err
		}
	} else {
		needsInit = true
	}

	return newBackendFromDatabase[H, N, E, Hasher, Header](db, canonicalizationDelay, dbConfig, needsInit)
}

func newBackendFromDatabase[H runtime.Hash, N runtime.Number, E runtime.Extrinsic, Hasher runtime.Hasher[H], Header runtime.Header[N, H]](
	db database.Database[hash.H256],
	canonicalizationDelay uint64,
	config DatabaseSettings,
	shouldInit bool,
) (Backend[H, N, E, Header], error) {
	var dbInitTransaction database.Transaction[hash.H256]

	requestedStatePruning := config.StatePruning
	stateMetaDB := stateMetaDB{db}
	// var mapE error

	stateDBInitCommitSet, stateDB, err := statedb.NewStateDB[H, hash.H256](stateMetaDB, requestedStatePruning, shouldInit)
	if err != nil {
		return Backend[H, N, E, Header]{}, err
	}

	applyStateCommit(&dbInitTransaction, stateDBInitCommitSet)

	statePruningUsed := stateDB.PruningMode()
	isArchivePruning := statePruningUsed.IsArchive()
	blockchain, err := NewBlockchainDB[H, N, Hasher, E, Header](db)
	if err != nil {
		return Backend[H, N, E, Header]{}, err
	}

	storageDB := storageDB[H]{
		DB:         db,
		StateDB:    stateDB,
		prefixKeys: true,
	}
	// let offchain_storage = offchain::LocalStorage::new(db.clone());

	backend := Backend[H, N, E, Header]{
		storage:               storageDB,
		blockchain:            *blockchain,
		canonicalizationDelay: canonicalizationDelay,
		isArchive:             isArchivePruning,
		blocksPruning:         config.BlocksPruning,
	}

	// Older DB versions have no last state key. Check if the state is available and set it.
	info := backend.blockchain.Info()
	if info.FinalizedState == nil && info.FinalizedHash != *new(H) &&
		backend.HaveStateAt(info.FinalizedHash, info.FinalizedNumber) {
		backend.blockchain.updateMeta(metaUpdate[H, N]{
			Hash:        info.FinalizedHash,
			Number:      info.FinalizedNumber,
			IsBest:      info.FinalizedHash == info.BestHash,
			IsFinalized: true,
			WithState:   true,
		})
	}

	err = db.Commit(dbInitTransaction)
	if err != nil {
		return Backend[H, N, E, Header]{}, err
	}

	return backend, nil
}

func newTestBackend[H runtime.Hash, N runtime.Number, E runtime.Extrinsic, Header runtime.Header[N, H]](
	blocksPruning uint32, canonicalizationDelay uint64) Backend[H, N, E, Header] {
	panic("unimplemented")
}

func newTestWithTxStorage[H runtime.Hash, N runtime.Number, E runtime.Extrinsic, Header runtime.Header[N, H]](
	blocksPruning uint32, canonicalizationDelay uint64) Backend[H, N, E, Header] {
	panic("unimplemented")
}

// / Reset the shared trie cache.
func (b *Backend[H, N, E, Header]) ResetTrieCache() {
	panic("unimplemented")
}

type NumberHash[H, N any] struct {
	Number N
	Hash   H
}

// / Handle setting head within a transaction. `route_to` should be the last
// / block that existed in the database. `best_to` should be the best block
// / to be set.
// /
// / In the case where the new best block is a block to be imported, `route_to`
// / should be the parent of `best_to`. In the case where we set an existing block
// / to be best, `route_to` should equal to `best_to`.
func (b *Backend[H, N, E, Header]) SetHeadWithTransaction(transaction *database.Transaction[H], routeTo H, bestTo NumberHash[H, N]) ([2]H, error) {
	panic("unimplemented")
}

func (b *Backend[H, N, E, Header]) ensureSequentialFinalization(header Header, lastFinalized *H) error {
	panic("unimplemented")
}

func (b *Backend[H, N, E, Header]) finalizeBlockWithTransaction(
	transaction *database.Transaction[H],
	hash H,
	header Header,
	lastFinalized *H,
	justification *runtime.Justification,
	currentTransactionJustifications map[H]runtime.Justification,
) (metaUpdate[H, N], error) {
	panic("unimplemented")
}

// performs forced canonicalization with a delay after importing a non-finalized block.
func (b *Backend[H, N, E, Header]) forceDelayedCanonicalize(transaction *database.Transaction[H]) error {
	panic("unimplemented")
}

func (b *Backend[H, N, E, Header]) tryCommitOperation(operation *BlockImportOperation) error {
	// NOTE: u are working on this, this will get called by CommitOperation when impl backend.Backend
	panic("unimplemented")
}

// write stuff to a transaction after a new block is finalized.
// this canonicalizes finalized blocks. Fails if called with a block which
// was not a child of the last finalized block.
func (b *Backend[H, N, E, Header]) noteFinalized(
	transaction *database.Transaction[H],
	fHeader Header,
	fHash H,
	withState bool,
	currentTransactionJustifications map[H]runtime.Justification,
) error {
	panic("unimplemented")
}

func (b *Backend[H, N, E, Header]) pruneDisplacedBranches(
	transaction *database.Transaction[H],
	finalized H,
	displaced api.FinalizationOutcome[H, N],
) error {
	panic("unimplemented")
}

func (b *Backend[H, N, E, Header]) pruneBlock(
	transaction *database.Transaction[H],
	finalized H,
	id generic.BlockID,
) error {
	panic("unimplemented")
}

func (b *Backend[H, N, E, Header]) emptyState() error {
	panic("unimplemented")
}

func (b *Backend[H, N, E, Header]) HaveStateAt(hash H, number N) bool {
	panic("unimplemented")
}

func applyStateCommit[H runtime.Hash](transaction *database.Transaction[hash.H256], commit statedb.CommitSet[H]) {
	for _, hdbv := range commit.Data.Inserted {
		transaction.SetFromVec(database.ColumnID(columns.State), hdbv.Hash.Bytes(), hdbv.DBValue)
	}
	for _, key := range commit.Data.Deleted {
		transaction.Remove(database.ColumnID(columns.State), key.Bytes())
	}
	for _, hdbv := range commit.Meta.Inserted {
		transaction.SetFromVec(database.ColumnID(columns.StateMeta), hdbv.Hash, hdbv.DBValue)
	}
	for _, key := range commit.Meta.Deleted {
		transaction.Remove(database.ColumnID(columns.StateMeta), key)
	}
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
