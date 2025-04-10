// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/api/utils"
	"github.com/ChainSafe/gossamer/internal/client/db/columns"
	"github.com/ChainSafe/gossamer/internal/client/db/metakeys"
	"github.com/ChainSafe/gossamer/internal/client/db/offchain"
	statedb "github.com/ChainSafe/gossamer/internal/client/state-db"
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/log"
	memorydb "github.com/ChainSafe/gossamer/internal/memory-db"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	p_offchain "github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/internal/primitives/trie/cache"
	"github.com/ChainSafe/gossamer/internal/saturating"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"golang.org/x/exp/slices"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "client/db"))

// BlocksPruning represent block pruning settings.
type BlocksPruning interface {
	isBlocksPruning()
}
type BlocksPruningValues interface {
	BlocksPruningKeepAll | BlocksPruningKeepFinalized | BlocksPruningSome
}

// BlocksPruningKeepAll keeps full block history, of every block that was ever imported.
type BlocksPruningKeepAll struct{}

// BlocksPruningKeepFinalized keeps full finalized block history.
type BlocksPruningKeepFinalized struct{}

// BlocksPruningSome keep a defined number of recent finalized blocks.
type BlocksPruningSome uint32

func (BlocksPruningKeepAll) isBlocksPruning()       {}
func (BlocksPruningKeepFinalized) isBlocksPruning() {}
func (BlocksPruningSome) isBlocksPruning()          {}

// DatabaseSource is the source of the database.
// NOTE: only uses a custom already-open database.
type DatabaseSource struct {
	// the handle to the custom storage
	DB database.Database[hash.H256]
	// if set, the create flag will be required to open such datasource
	RequireCreateFlag bool
}

// DBState is a db backed patricia trie state, transaction type is an overlay of changes to commit.
type DBState[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	*statemachine.TrieBackend[H, Hasher]
}

// refTrackingState is a reference tracking state.
//
// It makes sure that the hash we are using stays pinned in storage
// until this structure is dropped.
type refTrackingState[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	state      DBState[H, Hasher]
	storage    storageDB[H]
	parentHash *H // can be nil
}

func (rts *refTrackingState[H, Hasher]) Drop() {
	if rts.parentHash != nil {
		rts.storage.StateDB.Unpin(*rts.parentHash)
	}
}

// DatabaseConfig is the database configuration.
type DatabaseConfig struct {
	// TrieCacheMaximumSize is the maximum trie cache size in bytes.
	// If nil is given, the cache is disabled.
	TrieCacheMaximumSize *uint
	// StatePruning is the requested state pruning mode.
	StatePruning statedb.PruningMode
	// Source is the source of the database
	Source DatabaseSource
	// BlocksPruning is the block pruning mode.
	// NOTE: only finalized blocks are subject for removal!
	BlocksPruning BlocksPruning
}

// wrapper around [database.Database] that implements [statedb.MetaDB]
type stateMetaDB struct {
	db database.Database[hash.H256]
}

func (smdb stateMetaDB) GetMeta(key []byte) (statedb.DBValue, error) {
	val := smdb.db.Get(columns.StateMeta, key)
	if val == nil {
		return nil, nil
	}
	dbVal := statedb.DBValue(val)
	return dbVal, nil
}

type finalizedBlock[H runtime.Hash] struct {
	Hash H
	*runtime.Justification
}

// BlockImportOperation is [Backend] block import operation which represents a transaction.
type BlockImportOperation[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	oldState               refTrackingState[H, Hasher]
	dbUpdates              trie.PrefixedMemoryDB[H, Hasher]
	storageUpdates         overlayedchanges.StorageCollection
	childStorageUpdates    overlayedchanges.ChildStorageCollection
	offchainStorageUpdates overlayedchanges.OffchainChangesCollection
	pendingBlock           *pendingBlock[H, N, Header, E] // can be nil to represent no pending block
	auxOps                 api.AuxDataOperations
	finalizedBlocks        []finalizedBlock[H]
	setHead                *H // can be nil to represent no head
	commitState            bool
	createGap              bool
	indexOps               []overlayedchanges.IndexOperation
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) applyOffchain(transaction *database.Transaction[hash.H256]) {
	var count uint32
	offchainStorageUpdates := bio.offchainStorageUpdates
	bio.offchainStorageUpdates = nil
	for _, update := range offchainStorageUpdates {
		prefix := update.PrefixKey.Prefix
		key := update.PrefixKey.Key
		valueOperation := update.ValueOperation
		count += 1
		key = offchain.ConcatenatePrefixAndKey(prefix, key)
		switch valueOperation := valueOperation.(type) {
		case p_offchain.OffchainOverlayedChangeSetValue:
			transaction.Set(columns.Offchain, key, valueOperation)
		case p_offchain.OffchainOverlayedChangeRemove:
			transaction.Remove(columns.Offchain, key)
		default:
			panic("unreachable")
		}
	}

	if count > 0 {
		logger.Debugf("Applied %d offchain indexing changes.", count)
	}
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) applyAux(transaction *database.Transaction[hash.H256]) {
	auxOps := bio.auxOps
	bio.auxOps = nil
	for _, op := range auxOps {
		switch op.Data {
		case nil:
			transaction.Remove(columns.Aux, op.Key)
		default:
			transaction.Set(columns.Aux, op.Key, op.Data)
		}
	}
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) applyNewState(
	storage storage.Storage, stateVersion storage.StateVersion,
) (root H, err error) {
	contains := slices.ContainsFunc(storage.Top.Keys(), func(key string) bool {
		return keys.IsChildStorageKey([]byte(key))
	})
	if contains {
		return root, blockchain.ErrInvalidState
	}

	var childDeltas []statemachine.ChildDelta
	for childContent := range maps.Values(storage.ChildrenDefault) {
		var deltas []statemachine.Delta
		childContent.Data.Scan(func(key string, value []byte) bool {
			deltas = append(deltas, statemachine.Delta{
				Key:   []byte(key),
				Value: value,
			})
			return true
		})
		childDeltas = append(childDeltas, statemachine.ChildDelta{
			ChildInfo: childContent.ChildInfo,
			Deltas:    deltas,
		})
	}

	var deltas []statemachine.Delta
	storage.Top.Scan(func(key string, value []byte) bool {
		deltas = append(deltas, statemachine.Delta{
			Key:   []byte(key),
			Value: value,
		})
		return true
	})

	root, transaction := bio.oldState.state.FullStorageRoot(deltas, childDeltas, stateVersion)
	bio.dbUpdates = *transaction.PrefixedMemoryDB
	return root, nil
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) State() (statemachine.Backend[H, Hasher], error) {
	return &bio.oldState.state, nil
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) SetBlockData(
	header Header,
	body []E,
	indexedBody [][]byte,
	justifications runtime.Justifications,
	leafState api.NewBlockState,
) error {
	if bio.pendingBlock != nil {
		return fmt.Errorf("only one block per operation is allowed")
	}
	bio.pendingBlock = &pendingBlock[H, N, Header, E]{
		header:         header,
		body:           body,
		indexedBody:    indexedBody,
		justifications: justifications,
		leafState:      leafState,
	}
	return nil
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) UpdateDBStorage(
	update statemachine.BackendTransaction[H, Hasher],
) error {
	bio.dbUpdates = *update.PrefixedMemoryDB
	return nil
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) SetGenesisState(
	storage storage.Storage, commit bool, stateVersion storage.StateVersion,
) (H, error) {
	root, err := bio.applyNewState(storage, stateVersion)
	if err != nil {
		return root, err
	}
	bio.commitState = commit
	return root, err
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) ResetStorage(
	storage storage.Storage, stateVersion storage.StateVersion,
) (H, error) {
	root, err := bio.applyNewState(storage, stateVersion)
	if err != nil {
		return root, err
	}
	bio.commitState = true
	return root, err
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) UpdateStorage(
	update overlayedchanges.StorageCollection, childUpdate overlayedchanges.ChildStorageCollection,
) error {
	bio.storageUpdates = update
	bio.childStorageUpdates = childUpdate
	return nil
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) UpdateOffchainStorage(
	offchainUpdate overlayedchanges.OffchainChangesCollection,
) error {
	bio.offchainStorageUpdates = offchainUpdate
	return nil
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) InsertAux(ops api.AuxDataOperations) error {
	bio.auxOps = append(bio.auxOps, ops...)
	return nil
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) MarkFinalized(
	hash H, justification *runtime.Justification,
) error {
	bio.finalizedBlocks = append(bio.finalizedBlocks, finalizedBlock[H]{hash, justification})
	return nil
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) MarkHead(hash H) error {
	if bio.setHead != nil {
		panic("only one set head per operation is allowed")
	}
	bio.setHead = &hash
	return nil
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) UpdateTransactionIndex(
	indexOps []overlayedchanges.IndexOperation,
) error {
	bio.indexOps = indexOps
	return nil
}

func (bio *BlockImportOperation[H, Hasher, N, Header, E]) SetCreateGap(createGap bool) {
	bio.createGap = createGap
}

type pendingBlock[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H], E runtime.Extrinsic] struct {
	header         Header
	justifications runtime.Justifications // can be nil
	body           []E                    // can be nil to represent no body
	indexedBody    [][]byte               // can be nil to represent no indexed body
	leafState      api.NewBlockState
}

type nodeDBStorageDB[H runtime.Hash] struct {
	*storageDB[H]
}

func (ndbsdb nodeDBStorageDB[H]) Get(key string) (statedb.DBValue, error) {
	return ndbsdb.storageDB.db.Get(columns.State, []byte(key)), nil
}

type storageDB[H runtime.Hash] struct {
	db         database.Database[hash.H256]
	StateDB    *statedb.StateDB[H, string]
	prefixKeys bool
}

func (sdb *storageDB[H]) Get(key H, prefix hashdb.Prefix) ([]byte, error) {
	if sdb.prefixKeys {
		key := memorydb.NewPrefixedKey[H](key, prefix)
		return sdb.StateDB.Get(string(key), nodeDBStorageDB[H]{sdb})
	} else {
		return sdb.StateDB.Get(string(key.Bytes()), nodeDBStorageDB[H]{sdb})
	}
}

type dbGenesisStorage[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	root    H
	storage trie.PrefixedMemoryDB[H, Hasher]
}

type emptyStorage[H runtime.Hash] struct {
	root H
}

func newEmptyStorage[H runtime.Hash, Hasher runtime.Hasher[H]]() emptyStorage[H] {
	var root H
	mdb := trie.NewMemoryDB[H, Hasher]()
	trie := triedb.NewEmptyTrieDB[H, Hasher](mdb)
	trie.SetVersion(triedb.V1)
	root = trie.MustHash()
	return emptyStorage[H]{root}
}

// Backend keeps data in a key-value store. In archive mode, trie nodes are kept from all
// blocks. Otherwise, trie nodes are kept only from some recent blocks.
type Backend[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
] struct {
	storage               storageDB[H]
	offchainStorage       *offchain.LocalStorage
	blockchain            *blockchainDB[H, N, E, Header]
	canonicalizationDelay uint64
	importLock            sync.RWMutex
	isArchive             bool
	blocksPruning         BlocksPruning
	genesisState          *dbGenesisStorage[H, Hasher] // can be nil to represent no genesisState
	genesisStateMtx       sync.RWMutex
	sharedTrieCache       *cache.SharedTrieCache[H] // can be nil to respresent no shared trie cache
}

// NewBackend creates a new instance of database backend.
//
// dbConfig is of type [DatabaseConfig] and contains both state and block history pruning settings.
// canonicalizationDelay represents the number of blocks it waits to canonicalize the block and initiate
// pruning based on canonicalization.
func NewBackend[
	H runtime.Hash,
	N runtime.Number,
	E runtime.Extrinsic,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
](
	dbConfig DatabaseConfig,
	canonicalizationDelay uint64,
) (*Backend[H, Hasher, N, E, Header], error) {
	var (
		needsInit bool
		db        database.Database[hash.H256]
	)
	dbSource := dbConfig.Source
	db, err := openDatabase(dbSource, false)
	if err != nil {
		if errors.Is(err, errDoesNotExist) {
			db, err = openDatabase(dbSource, true)
			if err != nil {
				return nil, err
			}
			needsInit = true
		} else {
			return nil, err
		}
	} else {
		needsInit = false
	}

	return newBackendFromDatabase[H, N, E, Hasher, Header](db, canonicalizationDelay, dbConfig, needsInit)
}

func newBackendFromDatabase[
	H runtime.Hash,
	N runtime.Number,
	E runtime.Extrinsic,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
](
	db database.Database[hash.H256],
	canonicalizationDelay uint64,
	config DatabaseConfig,
	shouldInit bool,
) (*Backend[H, Hasher, N, E, Header], error) {
	var dbInitTransaction database.Transaction[hash.H256]

	requestedStatePruning := config.StatePruning
	stateMetaDB := stateMetaDB{db}

	stateDBInitCommitSet, stateDB, err := statedb.NewStateDB[H, string](stateMetaDB, requestedStatePruning, shouldInit)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", blockchain.ErrStateDatabase, err)
	}

	applyStateCommit(&dbInitTransaction, stateDBInitCommitSet)

	statePruningUsed := stateDB.PruningMode()
	isArchivePruning := statePruningUsed.IsArchive()
	blockchain, err := newBlockchainDB[H, N, Hasher, E, Header](db)
	if err != nil {
		return nil, err
	}

	storageDB := storageDB[H]{
		db:         db,
		StateDB:    stateDB,
		prefixKeys: true,
	}

	backend := Backend[H, Hasher, N, E, Header]{
		storage:               storageDB,
		offchainStorage:       offchain.NewLocalStorage(db),
		blockchain:            blockchain,
		canonicalizationDelay: canonicalizationDelay,
		isArchive:             isArchivePruning,
		blocksPruning:         config.BlocksPruning,
	}
	if config.TrieCacheMaximumSize != nil {
		backend.sharedTrieCache = cache.NewSharedTrieCache[H](*config.TrieCacheMaximumSize)
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
		return nil, err
	}

	return &backend, nil
}

// ResetTrieCache resets the shared trie cache.
func (b *Backend[H, Hasher, N, E, Header]) ResetTrieCache() {
	if b.sharedTrieCache != nil {
		b.sharedTrieCache.Reset()
	}
}

type numberHash[H, N any] struct {
	Number N
	Hash   H
}

// Handles setting head within a transaction. routeTo should be the last
// block that existed in the database. bestTo should be the best block
// to be set.
//
// In the case where the new best block is a block to be imported, routeTo
// should be the parent of bestTO. In the case where we set an existing block
// to be best, routeTo should equal to bestTo.
func (b *Backend[H, Hasher, N, E, Header]) setHeadWithTransaction(
	transaction *database.Transaction[hash.H256], routeTo H, bestTo numberHash[H, N],
) ([2][]H, error) {
	var (
		enacted   []H
		retracted []H
	)
	bestNumber := bestTo.Number
	bestHash := bestTo.Hash

	b.blockchain.metaMtx.RLock()
	defer b.blockchain.metaMtx.RUnlock()
	meta := b.blockchain.meta
	if saturating.Into[N, uint64](saturating.Sub(meta.BestNumber, bestNumber)) > b.canonicalizationDelay {
		return [2][]H{}, blockchain.ErrSetHeadTooOld
	}

	var parentExists bool
	status, err := b.blockchain.Status(routeTo)
	if err != nil {
		return [2][]H{}, err
	}
	parentExists = status == blockchain.BlockStatusInChain

	// Cannot find tree route with empty DB or when imported a detached block.
	if meta.BestHash != (*new(H)) && parentExists {
		treeRoute, err := blockchain.NewTreeRoute[H, N](b.blockchain, meta.BestHash, routeTo)
		if err != nil {
			return [2][]H{}, err
		}

		// uncanonicalize: check safety violations and ensure the numbers no longer
		// point to these block hashes in the key mapping.
		for _, r := range treeRoute.Retracted() {
			if r.Hash == meta.FinalizedHash {
				logger.Warnf("Potential safety failure: reverting finalized block %+v", r)

				return [2][]H{}, blockchain.ErrNotInFinalizedChain
			}

			retracted = append(retracted, r.Hash)
			err := removeNumberToKeyMapping(transaction, uint32(columns.KeyLookup), r.Number)
			if err != nil {
				return [2][]H{}, err
			}
		}

		// canonicalize: set the number lookup to map to this block's hash.
		for _, e := range treeRoute.Enacted() {
			enacted = append(enacted, e.Hash)
			err := insertNumberToKeyMapping(transaction, uint32(columns.KeyLookup), e.Number, e.Hash)
			if err != nil {
				return [2][]H{}, err
			}
		}
	}

	lookupKey, err := newLookupKey(bestNumber, bestHash)
	if err != nil {
		return [2][]H{}, err
	}
	transaction.Set(columns.Meta, metakeys.BestBlock, lookupKey)
	err = insertNumberToKeyMapping(transaction, uint32(columns.KeyLookup), bestNumber, bestHash)
	if err != nil {
		return [2][]H{}, err
	}

	return [2][]H{enacted, retracted}, nil
}

func (b *Backend[H, Hasher, N, E, Header]) ensureSequentialFinalization(header Header, lastFinalized *H) error {
	if lastFinalized == nil {
		b.blockchain.metaMtx.RLock()
		lf := b.blockchain.meta.FinalizedHash
		lastFinalized = &lf
		b.blockchain.metaMtx.RUnlock()
	}
	b.blockchain.metaMtx.RLock()
	defer b.blockchain.metaMtx.RUnlock()
	if *lastFinalized != b.blockchain.meta.FinalizedHash && header.ParentHash() != *lastFinalized {
		return fmt.Errorf("%w: Last finalized %s not parent of %s",
			blockchain.ErrNonSequentialFinalization, *lastFinalized, header.Hash())
	}
	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) finalizeBlockWithTransaction(
	transaction *database.Transaction[hash.H256],
	hash H,
	header *Header,
	lastFinalized *H,
	justification *runtime.Justification,
	currentTransactionJustifications map[H]runtime.Justification,
) (metaUpdate[H, N], error) {
	// TODO: ensure best chain contains this block. (from substrate as well)
	number := (*header).Number()
	err := b.ensureSequentialFinalization(*header, lastFinalized)
	if err != nil {
		return metaUpdate[H, N]{}, err
	}
	withState := b.HaveStateAt(hash, number)

	err = b.noteFinalized(transaction, *header, hash, withState, currentTransactionJustifications)
	if err != nil {
		return metaUpdate[H, N]{}, err
	}

	if justification != nil {
		lookupKey, err := newLookupKey(number, hash)
		if err != nil {
			return metaUpdate[H, N]{}, err
		}
		justifications := runtime.Justifications{*justification}
		transaction.Set(columns.Justifications, lookupKey, scale.MustMarshal(justifications))
		currentTransactionJustifications[hash] = *justification
	}
	return metaUpdate[H, N]{
		Hash:        hash,
		Number:      number,
		IsBest:      false,
		IsFinalized: true,
		WithState:   withState,
	}, nil
}

// performs forced canonicalization with a delay after importing a non-finalized block.
func (b *Backend[H, Hasher, N, E, Header]) forceDelayedCanonicalize(
	transaction *database.Transaction[hash.H256],
) error {
	var bestCanonical uint64
	switch lc := b.storage.StateDB.LastCanonicalized().(type) {
	case statedb.LastCanonicalizedNone:
		bestCanonical = 0
	case statedb.LastCanonicalizedBlock:
		bestCanonical = uint64(lc)
	case statedb.LastCanonicalizedNotCanonicalizing:
		// Nothing needs to be done when canonicalization is not happening.
		return nil
	}

	info := b.blockchain.Info()
	bestNumber := saturating.Into[N, uint64](info.BestNumber)

	end := saturating.Sub(bestNumber, b.canonicalizationDelay)
	for toCanonicalize := bestCanonical + 1; toCanonicalize <= end; toCanonicalize++ {
		hashToCanonicalize, err := b.blockchain.Hash(saturating.Into[uint64, N](toCanonicalize))
		if err != nil {
			return err
		}
		if hashToCanonicalize == nil {
			bestHash := info.BestHash
			return fmt.Errorf("%w: Can't canonicalize missing block number %d when for best block %s (%d})",
				blockchain.ErrBackend, toCanonicalize, bestHash, bestNumber)
		}

		if !b.HaveStateAt(*hashToCanonicalize, saturating.Into[uint64, N](toCanonicalize)) {
			return nil
		}

		logger.Tracef("Canonicalize block #%d (%s)", toCanonicalize, *hashToCanonicalize)
		commit, err := b.storage.StateDB.CanonicalizeBlock(*hashToCanonicalize)
		if err != nil {
			return fmt.Errorf("%w: %v", blockchain.ErrStateDatabase, err)
		}
		applyStateCommit(transaction, commit)
	}

	return nil
}

// TODO: add create_gap logic to this function
func (b *Backend[H, Hasher, N, E, Header]) tryCommitOperation( //nolint:gocyclo
	operation *BlockImportOperation[H, Hasher, N, Header, E],
) error {
	var transaction database.Transaction[hash.H256]

	operation.applyAux(&transaction)
	operation.applyOffchain(&transaction)

	var metaUpdates []metaUpdate[H, N]

	b.blockchain.metaMtx.RLock()

	meta := b.blockchain.meta
	bestNum := meta.BestNumber
	lastFinalizedHash := meta.FinalizedHash
	lastFinalizedNumber := meta.FinalizedNumber
	blockGap := meta.BlockGap
	blockGapUpdated := false

	b.blockchain.metaMtx.RUnlock()

	currentTransactionJustifications := make(map[H]runtime.Justification)
	for _, finalizedBlock := range operation.finalizedBlocks {
		blockHeader, err := b.blockchain.header(finalizedBlock.Hash)
		if err != nil {
			return err
		}
		meta, err := b.finalizeBlockWithTransaction(
			&transaction,
			finalizedBlock.Hash,
			blockHeader,
			&lastFinalizedHash,
			finalizedBlock.Justification,
			currentTransactionJustifications,
		)
		if err != nil {
			return err
		}
		metaUpdates = append(metaUpdates, meta)
		lastFinalizedHash = finalizedBlock.Hash
		lastFinalizedNumber = (*blockHeader).Number()
	}

	type headerHash struct {
		Header Header
		Hash   H
	}
	var imported *headerHash
	if pendingBlock := operation.pendingBlock; pendingBlock != nil {
		hash := pendingBlock.header.Hash()

		parentHash := pendingBlock.header.ParentHash()
		number := pendingBlock.header.Number()
		b.blockchain.leavesMtx.RLock()
		highestLeaf := b.blockchain.leaves.HighestLeaf()
		b.blockchain.leavesMtx.RUnlock()
		var highestLeafNumber N
		if highestLeaf != nil {
			highestLeafNumber = highestLeaf.Number
		}
		var existingHeader bool
		if number <= highestLeafNumber {
			header, err := b.blockchain.Header(hash)
			if err != nil {
				return err
			}
			existingHeader = header != nil
		}

		existingBody := pendingBlock.body != nil

		// blocks are keyed by number + hash.
		lookupKey, err := newLookupKey(number, hash)
		if err != nil {
			return err
		}

		if pendingBlock.leafState.IsBest() {
			_, err := b.setHeadWithTransaction(&transaction, parentHash, numberHash[H, N]{
				Number: number,
				Hash:   hash,
			})
			if err != nil {
				return err
			}
		}

		err = insertHashToKeyMapping(&transaction, uint32(columns.KeyLookup), number, hash)
		if err != nil {
			return err
		}

		transaction.Set(columns.Header, lookupKey, scale.MustMarshal(pendingBlock.header))
		if body := pendingBlock.body; body != nil {
			// If we have any index operations we save block in the new format with indexed
			// extrinsic headers Otherwise we save the body as a single blob.
			if len(operation.indexOps) == 0 {
				transaction.Set(columns.Body, lookupKey, scale.MustMarshal(body))
			} else {
				body := applyIndexOps(&transaction, body, operation.indexOps)
				transaction.Set(columns.BodyIndex, lookupKey, body)
			}
		}
		if body := pendingBlock.indexedBody; body != nil {
			applyIndexedBody(&transaction, body)
		}
		if justifications := pendingBlock.justifications; justifications != nil {
			transaction.Set(columns.Justifications, lookupKey, scale.MustMarshal(justifications))
		}

		if number == 0 {
			transaction.Set(columns.Meta, metakeys.GenesisHash, hash.Bytes())

			if operation.commitState {
				transaction.Set(columns.Meta, metakeys.FinalizedState, lookupKey)
			} else {
				// When we don't want to commit the genesis state, we still preserve it in
				// memory to bootstrap consensus. It is queried for an initial list of
				// authorities, etc.
				b.genesisStateMtx.Lock()
				b.genesisState = &dbGenesisStorage[H, Hasher]{
					root:    pendingBlock.header.StateRoot(),
					storage: operation.dbUpdates,
				}
				b.genesisStateMtx.Unlock()
			}
		}

		var finalized bool
		if operation.commitState {
			var (
				changeset    statedb.ChangeSet[string]
				ops          uint64
				bytes        uint64
				removal      uint64
				bytesRemoval uint64
			)
			for key, update := range operation.dbUpdates.Drain() {
				if rc := update.RC; rc > 0 {
					ops += 1
					bytes += uint64(len(key) + len(update.Data))
					if rc == 1 {
						changeset.Inserted = append(changeset.Inserted, statedb.HashDBValue[string]{
							Hash:    key,
							DBValue: update.Data,
						})
					} else {
						changeset.Inserted = append(changeset.Inserted, statedb.HashDBValue[string]{
							Hash:    key,
							DBValue: update.Data,
						})
						for i := int32(0); i < rc-1; i++ {
							changeset.Inserted = append(changeset.Inserted, statedb.HashDBValue[string]{
								Hash:    key,
								DBValue: make([]byte, 0),
							})
						}
					}
				} else if rc < 0 {
					removal += 1
					bytesRemoval += uint64(len(key))
					if rc == -1 {
						changeset.Deleted = append(changeset.Deleted, key)
					} else {
						for i := int32(0); i < (rc * -1); i++ {
							changeset.Deleted = append(changeset.Deleted, key)
						}
					}
				}
			}

			numberU64 := saturating.Into[N, uint64](number)
			commit, err := b.storage.StateDB.InsertBlock(hash, numberU64, pendingBlock.header.ParentHash(), changeset)
			if err != nil {
				return fmt.Errorf("%w: %v", blockchain.ErrStateDatabase, err)
			}
			applyStateCommit(&transaction, commit)
			if number <= lastFinalizedNumber {
				// Canonicalize in the db when re-importing existing blocks with state.
				commit, err := b.storage.StateDB.CanonicalizeBlock(hash)
				if err != nil {
					return fmt.Errorf("%w: %v", blockchain.ErrStateDatabase, err)
				}
				applyStateCommit(&transaction, commit)
				metaUpdates = append(metaUpdates, metaUpdate[H, N]{
					Hash:        hash,
					Number:      number,
					IsBest:      false,
					IsFinalized: true,
					WithState:   true,
				})
			}

			// Check if need to finalize. Genesis is always finalized instantly.
			finalized = numberU64 == 0 || pendingBlock.leafState.IsFinal()
		} else {
			finalized = number == 0 && lastFinalizedNumber == 0 || pendingBlock.leafState.IsFinal()
		}

		header := pendingBlock.header
		isBest := pendingBlock.leafState.IsBest()
		logger.Debugf("DB commit %s (%d), best=%v, state=%+v, existing=%+v, finalized=%v",
			hash, number, isBest, operation.commitState, existingHeader, finalized,
		)

		// VERY IMPORTANT: drop state reference so that it can be finalized
		// NOTE: this is supposed to merge the state usage stats as well if we decide to implement that
		operation.oldState.Drop()

		if finalized {
			// TODO: ensure best chain contains this block. (from substrate as well)
			err := b.ensureSequentialFinalization(header, &lastFinalizedHash)
			if err != nil {
				return err
			}
			currentTransactionJustifications := make(map[H]runtime.Justification)
			err = b.noteFinalized(&transaction, header, hash, operation.commitState, currentTransactionJustifications)
			if err != nil {
				return err
			}
		} else {
			err := b.forceDelayedCanonicalize(&transaction)
			if err != nil {
				return err
			}
		}

		if !existingHeader {
			// Add a new leaf if the block has the potential to be finalized.
			if number > lastFinalizedNumber || lastFinalizedNumber == 0 {
				b.blockchain.leavesMtx.Lock()
				b.blockchain.leaves.Import(hash, number, parentHash)
				b.blockchain.leaves.PrepareTransaction(&transaction, uint32(columns.Meta), metakeys.LeafPrefix)
				b.blockchain.leavesMtx.Unlock()
			}

			children, err := readChildren(b.storage.db, columns.Meta, metakeys.ChildrenPrefix, parentHash)
			if err != nil {
				return err
			}
			if !slices.Contains(children, hash) {
				children = append(children, hash)
				writeChildren(&transaction, columns.Meta, metakeys.ChildrenPrefix, parentHash, children)
			}
		}

		shouldCheckBlockGap := !existingHeader && !existingBody
		if shouldCheckBlockGap {
			insertNewGap := func(
				transaction *database.Transaction[dbHash],
				newGap blockchain.BlockGap[N],
				gap *blockchain.BlockGap[N],
			) {
				transaction.Set(columns.Meta, metakeys.BlockGap, scale.MustMarshal(newGap))
				transaction.Set(columns.Meta, metakeys.BlockGapVersion, scale.MustMarshal(blockGapCurrentVersion))
				*gap = newGap
			}

			if blockGap != nil {
				switch blockGap.Type {
				case blockchain.BlockGapMissingHeaderAndBody:
					if number == blockGap.Start {
						blockGap.Start += 1
						err := insertNumberToKeyMapping(&transaction, uint32(columns.KeyLookup), number, hash)
						if err != nil {
							return err
						}

						if blockGap.Start > blockGap.End {
							transaction.Remove(columns.Meta, metakeys.BlockGap)
							transaction.Remove(columns.Meta, metakeys.BlockGapVersion)
							blockGap = nil
							logger.Debugf("Removed block gap")
						} else {
							insertNewGap(&transaction, *blockGap, blockGap)
							logger.Debugf("Updated block gap %v", *blockGap)
						}
						blockGapUpdated = true
					}
				case blockchain.BlockGapMissingBody:
					// Gap increased when syncing the header chain during fast sync.
					if number == blockGap.End+1 && !existingBody {
						blockGap.End += 1
						err := insertNumberToKeyMapping(&transaction, uint32(columns.KeyLookup), number, hash)
						if err != nil {
							return err
						}
						insertNewGap(&transaction, *blockGap, blockGap)
						logger.Debugf("Updated block gap %v", *blockGap)
						blockGapUpdated = true

					} else if number == blockGap.Start && existingBody { // Gap decreased when downloading the full blocks.
						blockGap.Start += 1
						if blockGap.Start > blockGap.End {
							transaction.Remove(columns.Meta, metakeys.BlockGap)
							transaction.Remove(columns.Meta, metakeys.BlockGapVersion)
							blockGap = nil
							logger.Debugf("Removed block gap")
						} else {
							insertNewGap(&transaction, *blockGap, blockGap)
							logger.Debugf("Updated block gap %v", *blockGap)
						}
						blockGapUpdated = true
					}
				}
			} else if operation.createGap {
				parentHeader, err := b.blockchain.Header(parentHash)
				if err != nil {
					return err
				}
				if number > bestNum+1 && parentHeader == nil {
					newGap := blockchain.BlockGap[N]{
						Start: bestNum + 1,
						End:   number - 1,
						Type:  blockchain.BlockGapMissingHeaderAndBody,
					}
					insertNewGap(&transaction, newGap, blockGap)
					blockGapUpdated = true
					logger.Debugf("Detected block gap (warp sync) %v", blockGap)
				} else if number == bestNum+1 && parentHeader != nil && !existingBody {
					newGap := blockchain.BlockGap[N]{
						Start: number,
						End:   number,
						Type:  blockchain.BlockGapMissingBody,
					}
					insertNewGap(&transaction, newGap, blockGap)
					blockGapUpdated = true
					logger.Debugf("Detected block gap (fast sync) %v", blockGap)
				}
			}
		}

		metaUpdates = append(metaUpdates, metaUpdate[H, N]{
			Hash:        hash,
			Number:      number,
			IsBest:      pendingBlock.leafState.IsBest(),
			IsFinalized: finalized,
			WithState:   operation.commitState,
		})
		imported = &headerHash{
			Header: pendingBlock.header,
			Hash:   hash,
		}
	}

	if setHead := operation.setHead; setHead != nil {
		header, err := b.blockchain.header(*setHead)
		if err != nil {
			return err
		}
		if header != nil {
			number := (*header).Number()
			hash := (*header).Hash()

			_, err := b.setHeadWithTransaction(&transaction, hash, numberHash[H, N]{Number: number, Hash: hash})
			if err != nil {
				return err
			}

			metaUpdates = append(metaUpdates, metaUpdate[H, N]{
				Hash:        hash,
				Number:      number,
				IsBest:      true,
				IsFinalized: false,
				WithState:   false,
			})
		} else {
			return fmt.Errorf("%w Cannot set head %s", blockchain.ErrUnknownBlock, setHead)
		}
	}

	err := b.storage.db.Commit(transaction)
	if err != nil {
		return err
	}

	// Apply all in-memory state changes.
	// Code beyond this point can't fail.
	if imported != nil {
		header := &(imported.Header)
		hash := imported.Hash
		logger.Tracef("DB commit done %s", hash)
		headerMetadata := blockchain.NewCachedHeaderMetadata(*header)
		b.blockchain.InsertHeaderMetadata(headerMetadata.Hash, headerMetadata)
		b.blockchain.headerCacheMtx.Lock()
		b.blockchain.cacheHeader(hash, header)
		b.blockchain.headerCacheMtx.Unlock()
	}

	for _, m := range metaUpdates {
		b.blockchain.updateMeta(m)
	}

	if blockGapUpdated {
		b.blockchain.updateBlockGap(blockGap)
	}

	return nil
}

// Write to a transaction after a new block is finalized.
// This canonicalizes finalized blocks. Fails if called with a block which
// is not a child of the last finalized block.
func (b *Backend[H, Hasher, N, E, Header]) noteFinalized(
	transaction *database.Transaction[hash.H256],
	fHeader Header,
	fHash H,
	withState bool,
	currentTransactionJustifications map[H]runtime.Justification,
) error {
	fNum := fHeader.Number()

	lookupKey, err := newLookupKey(fNum, fHash)
	if err != nil {
		return err
	}
	if withState {
		transaction.Set(columns.Meta, metakeys.FinalizedState, lookupKey)
	}
	transaction.Set(columns.Meta, metakeys.FinalizedBlock, lookupKey)

	var requiresCanonicalization bool
	switch lc := b.storage.StateDB.LastCanonicalized().(type) {
	case statedb.LastCanonicalizedNone:
		requiresCanonicalization = true
	case statedb.LastCanonicalizedBlock:
		requiresCanonicalization = saturating.Into[N, uint64](fNum) > uint64(lc)
	case statedb.LastCanonicalizedNotCanonicalizing:
		requiresCanonicalization = false
	default:
		panic("unreachable")
	}

	if requiresCanonicalization && b.HaveStateAt(fHash, fNum) {
		commit, err := b.storage.StateDB.CanonicalizeBlock(fHash)
		if err != nil {
			return fmt.Errorf("%w: %v", blockchain.ErrStateDatabase, err)
		}
		applyStateCommit(transaction, commit)
	}

	b.blockchain.leavesMtx.Lock()
	defer b.blockchain.leavesMtx.Unlock()
	newDisplaced := b.blockchain.leaves.FinalizeHeight(fNum)
	err = b.pruneBlocks(transaction, fNum, fHash, newDisplaced, currentTransactionJustifications)
	if err != nil {
		return err
	}

	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) pruneBlocks(
	transaction *database.Transaction[hash.H256],
	finalizedNumber N,
	finalizedHash H,
	displaced api.FinalizationOutcome[H, N],
	currentTransactionJustifications map[H]runtime.Justification,
) error {
	switch blocksPruning := b.blocksPruning.(type) {
	case BlocksPruningKeepAll:
	case BlocksPruningSome:
		// Always keep the last finalized block
		keep := uint32(1)
		if uint32(blocksPruning) > keep {
			keep = uint32(blocksPruning)
		}
		if finalizedNumber >= N(keep) {
			number := saturating.Sub(finalizedNumber, N(keep))

			// Before we prune a block, check if it is pinned
			hash, err := b.blockchain.Hash(number)
			if err != nil {
				return err
			}
			if hash != nil {
				err := b.blockchain.insertPersistedBodyIfPinned(*hash)
				if err != nil {
					return err
				}

				// If the block was finalized in this transaction, it will not be in the db
				// yet.
				justification, ok := currentTransactionJustifications[*hash]
				if ok {
					delete(currentTransactionJustifications, *hash)
					b.blockchain.insertJustifcationsIfPinned(*hash, justification)
				} else {
					err := b.blockchain.insertPersistedJustificationsIfPinned(*hash)
					if err != nil {
						return err
					}
				}
			}
			err = b.pruneBlock(transaction, generic.BlockIDNumber[N]{Number: number})
			if err != nil {
				return err
			}
		}
		err := b.pruneDisplacedBranches(transaction, finalizedHash, displaced)
		if err != nil {
			return err
		}
	case BlocksPruningKeepFinalized:
		err := b.pruneDisplacedBranches(transaction, finalizedHash, displaced)
		if err != nil {
			return err
		}
	default:
		panic("unreachable")
	}
	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) pruneDisplacedBranches(
	transaction *database.Transaction[hash.H256],
	finalized H,
	displaced api.FinalizationOutcome[H, N],
) error {
	// Discard all blocks from displaced branches
	for _, h := range displaced.Leaves() {
		treeRoute, err := blockchain.NewTreeRoute(b.blockchain, h, finalized)
		if err != nil {
			if errors.Is(err, blockchain.ErrUnknownBlock) {
				// Sometimes routes can't be calculated. eg. after warp sync.
				return nil
			}
			return err
		}
		for _, r := range treeRoute.Retracted() {
			err := b.blockchain.insertPersistedBodyIfPinned(r.Hash)
			if err != nil {
				return err
			}
			err = b.pruneBlock(transaction, generic.BlockIDHash[H]{Hash: r.Hash})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) pruneBlock(
	transaction *database.Transaction[hash.H256],
	id generic.BlockID,
) error {
	logger.Debugf("Removing block %s", id)
	err := removeFromDB[H, N](transaction, b.storage.db, uint32(columns.KeyLookup), uint32(columns.Body), id)
	if err != nil {
		return err
	}
	err = removeFromDB[H, N](transaction, b.storage.db, uint32(columns.KeyLookup), uint32(columns.Justifications), id)
	if err != nil {
		return err
	}
	index, err := readDB[H, N](b.storage.db, columns.KeyLookup, columns.BodyIndex, id)
	if err != nil {
		return err
	}
	if index != nil {
		err := removeFromDB[H, N](transaction, b.storage.db, uint32(columns.KeyLookup), uint32(columns.BodyIndex), id)
		if err != nil {
			return err
		}
		var dbExtrinsics []dbExtrinsic[E]
		err = scale.Unmarshal(index, &dbExtrinsics)
		if err != nil {
			return fmt.Errorf("%w: Error decoding body list: %v", blockchain.ErrBackend, err)
		}
		for _, ex := range dbExtrinsics {
			val, err := ex.Value()
			if err != nil {
				return err
			}
			indexed, ok := val.(dbExtrinsicIndexed)
			if ok {
				transaction.Release(columns.Transaction, indexed.Hash)
			}
		}
	}
	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) emptyState() refTrackingState[H, Hasher] {
	root := newEmptyStorage[H, Hasher]().root
	var localCache *cache.LocalTrieCache[H]
	if b.sharedTrieCache != nil {
		lcc := b.sharedTrieCache.LocalTrieCache()
		localCache = &lcc
	}
	dbState := statemachine.NewTrieBackend[H, Hasher](&b.storage, root, localCache, nil)
	state := refTrackingState[H, Hasher]{
		state:      DBState[H, Hasher]{dbState},
		storage:    b.storage,
		parentHash: nil,
	}
	return state
}

func applyStateCommit(transaction *database.Transaction[hash.H256], commit statedb.CommitSet[string]) {
	for _, hdbv := range commit.Data.Inserted {
		transaction.Set(columns.State, []byte(hdbv.Hash), hdbv.DBValue)
	}
	for _, key := range commit.Data.Deleted {
		transaction.Remove(columns.State, []byte(key))
	}
	for _, hdbv := range commit.Meta.Inserted {
		transaction.Set(columns.StateMeta, hdbv.Hash, hdbv.DBValue)
	}
	for _, key := range commit.Meta.Deleted {
		transaction.Remove(columns.StateMeta, key)
	}
}

func applyIndexOps[E runtime.Extrinsic](
	transaction *database.Transaction[hash.H256], body []E, ops []overlayedchanges.IndexOperation,
) []byte {
	var extrinsicIndex []dbExtrinsic[E]
	indexMap := make(map[uint32]struct {
		Hash []byte
		Size uint32
	})
	renewedMap := make(map[uint32]hash.H256)
	for _, op := range ops {
		switch op := op.(type) {
		case overlayedchanges.IndexOperationInsert:
			indexMap[op.Extrinsic] = struct {
				Hash []byte
				Size uint32
			}{Hash: op.Hash, Size: op.Size}
		case overlayedchanges.IndexOperationRenew:
			renewedMap[op.Extrinsic] = hash.H256(op.Hash)
		default:
			panic("unreachable")
		}
	}
	for index, extrinsic := range body {
		var dbExtrinsic dbExtrinsic[E]
		hash, ok := renewedMap[uint32(index)]
		if ok {
			// Bump ref counter
			encoded := scale.MustMarshal(extrinsic)
			transaction.Reference(columns.Transaction, hash)
			dbExtrinsic = newDbExtrinsic[E](dbExtrinsicIndexed{Hash: hash, Header: encoded})
		} else {
			i, ok := indexMap[uint32(index)]
			if ok {
				encoded := scale.MustMarshal(extrinsic)
				if int(i.Size) <= len(encoded) {
					offset := len(encoded) - int(i.Size)
					transaction.Store(columns.Transaction, dbHash(i.Hash), encoded[offset:])
					dbExtrinsic = newDbExtrinsic[E](dbExtrinsicIndexed{Hash: dbHash(i.Hash), Header: encoded[:offset]})
				} else {
					// Invalid indexed slice. Just store full data and don't index anything.
					dbExtrinsic = newDbExtrinsic[E](dbExtrinsicFull[E]{Extrinsic: extrinsic})
				}
			} else {
				dbExtrinsic = newDbExtrinsic[E](dbExtrinsicFull[E]{Extrinsic: extrinsic})
			}
		}
		extrinsicIndex = append(extrinsicIndex, dbExtrinsic)
	}
	logger.Debugf("DB transaction index: %d inserted, %d renewed, %d full",
		len(indexMap), len(renewedMap), len(extrinsicIndex)-len(indexMap)-len(renewedMap))

	return scale.MustMarshal(extrinsicIndex)
}

func applyIndexedBody(transaction *database.Transaction[hash.H256], body [][]byte) {
	for _, extrinsic := range body {
		hash := runtime.BlakeTwo256{}.Hash(extrinsic)
		transaction.Store(columns.Transaction, hash, extrinsic)
	}
}

func (b *Backend[H, Hasher, N, E, Header]) InsertAux(insert []api.KeyValue, delete [][]byte) error {
	var transaction database.Transaction[dbHash]
	for _, kv := range insert {
		transaction.Set(columns.Aux, kv.Key, kv.Value)
	}
	for _, h := range delete {
		transaction.Remove(columns.Aux, h)
	}
	err := b.storage.db.Commit(transaction)
	if err != nil {
		return err
	}
	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) GetAux(key []byte) ([]byte, error) {
	return b.storage.db.Get(columns.Aux, key), nil
}

func (b *Backend[H, Hasher, N, E, Header]) beginOperation() *BlockImportOperation[H, Hasher, N, Header, E] {
	return &BlockImportOperation[H, Hasher, N, Header, E]{
		pendingBlock:           nil,
		oldState:               b.emptyState(),
		dbUpdates:              *trie.NewPrefixedMemoryDB[H, Hasher](),
		storageUpdates:         make(overlayedchanges.StorageCollection, 0),
		childStorageUpdates:    make(overlayedchanges.ChildStorageCollection, 0),
		offchainStorageUpdates: make(overlayedchanges.OffchainChangesCollection, 0),
		auxOps:                 make(api.AuxDataOperations, 0),
		finalizedBlocks:        make([]finalizedBlock[H], 0),
		setHead:                nil,
		commitState:            false,
		indexOps:               make([]overlayedchanges.IndexOperation, 0),
	}
}

func (b *Backend[H, Hasher, N, E, Header]) BeginOperation() (
	api.BlockImportOperation[N, H, Hasher, Header, E], error,
) {
	return b.beginOperation(), nil
}

func (b *Backend[H, Hasher, N, E, Header]) BeginStateOperation(
	operation api.BlockImportOperation[N, H, Hasher, Header, E], block H,
) error {
	op := operation.(*BlockImportOperation[H, Hasher, N, Header, E])
	if block == *(new(H)) {
		op.oldState = b.emptyState()
	} else {
		state, err := b.stateAt(block)
		if err != nil {
			return err
		}
		op.oldState = state
	}

	op.commitState = true
	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) CommitOperation(
	operation api.BlockImportOperation[N, H, Hasher, Header, E],
) error {
	op := operation.(*BlockImportOperation[H, Hasher, N, Header, E])

	err := b.tryCommitOperation(op)
	if err != nil {
		stateMetaDB := stateMetaDB{b.storage.db}
		resetErr := b.storage.StateDB.Reset(stateMetaDB)
		if resetErr != nil {
			return fmt.Errorf("%w: %v", blockchain.ErrStateDatabase, resetErr)
		}
		b.blockchain.clearPinningCache()
		return err
	}
	b.storage.StateDB.Sync()
	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) FinalizeBlock(hash H, justification *runtime.Justification) error {
	var transaction database.Transaction[dbHash]
	header, err := b.blockchain.header(hash)
	if err != nil {
		return err
	}
	currentTransactionJustifications := make(map[H]runtime.Justification)
	m, err := b.finalizeBlockWithTransaction(
		&transaction, hash, header, nil, justification, currentTransactionJustifications,
	)
	if err != nil {
		return err
	}

	err = b.storage.db.Commit(transaction)
	if err != nil {
		return err
	}
	b.blockchain.updateMeta(m)
	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) AppendJustification(hash H, justification runtime.Justification) error {
	var transaction database.Transaction[dbHash]
	header, err := b.blockchain.header(hash)
	if err != nil {
		return err
	}
	number := (*header).Number()

	// Check if the block is finalized first.
	isDescendantOf := utils.IsDescendantOf(b.blockchain, nil)
	lastFinalized, err := b.blockchain.LastFinalized()
	if err != nil {
		return err
	}

	// We can do a quick check first, before doing a proper but more expensive check.
	if number > b.blockchain.Info().FinalizedNumber {
		return blockchain.ErrNotInFinalizedChain
	}
	if hash != lastFinalized {
		ido, err := isDescendantOf(hash, lastFinalized)
		if err != nil {
			return err
		}
		if !ido {
			return blockchain.ErrNotInFinalizedChain
		}
	}

	var justifications runtime.Justifications
	storedJustifications, err := b.blockchain.Justifications(hash)
	if err != nil {
		return err
	}
	if storedJustifications != nil {
		if !storedJustifications.Append(justification) {
			return fmt.Errorf("%w: Duplicate consensus engine ID", blockchain.ErrBadJustification)
		}
		justifications = storedJustifications
	} else {
		justifications = runtime.Justifications{justification}
	}

	lookupKey, err := newLookupKey(number, hash)
	if err != nil {
		return err
	}
	transaction.Set(columns.Justifications, lookupKey, scale.MustMarshal(justifications))

	err = b.storage.db.Commit(transaction)
	if err != nil {
		return err
	}

	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) OffchainStorage() p_offchain.OffchainStorage {
	return b.offchainStorage
}

func (b *Backend[H, Hasher, N, E, Header]) Revert(n N, revertFinalized bool) (N, map[H]any, error) {
	revertedFinalized := make(map[H]any)

	info := b.blockchain.Info()

	var highestLeaf *numberHash[H, N]
	b.blockchain.leavesMtx.RLock()
	numberHashes := b.blockchain.leaves.HighestLeaf()
	if numberHashes != nil {
		highestLeaf = &numberHash[H, N]{
			Number: numberHashes.Number,
			Hash:   numberHashes.Hashes[len(numberHashes.Hashes)-1],
		}
	}
	b.blockchain.leavesMtx.RUnlock()
	bestNumber := info.BestNumber
	bestHash := info.BestHash

	finalized := info.FinalizedNumber

	revertible := bestNumber - finalized

	if !revertFinalized && revertible < n {
		n = revertible
	}

	var (
		numberToRevert N
		hashToRevert   H
	)
	if highestLeaf != nil {
		n = n + highestLeaf.Number - bestNumber
		numberToRevert = highestLeaf.Number
		hashToRevert = highestLeaf.Hash
	} else {
		numberToRevert = bestNumber
		hashToRevert = bestHash
	}

	var revertBlocks = func() (N, error) {
		for c := uint64(0); c < saturating.Into[N, uint64](n); c++ {
			if numberToRevert == 0 {
				return saturating.Into[uint64, N](c), nil
			}
			var transaction database.Transaction[hash.H256]
			removed, err := b.blockchain.Header(hashToRevert)
			if err != nil {
				return 0, err
			}
			if removed == nil {
				return 0, fmt.Errorf("%w: Error reverting to %s, Block header not found",
					blockchain.ErrUnknownBlock, hashToRevert)
			}
			removedHash := (*removed).Hash()

			prevNumber := saturating.Sub(numberToRevert, 1)
			var prevHash H
			if prevNumber == bestNumber {
				prevHash = bestHash
			} else {
				prevHash = (*removed).ParentHash()
			}

			if !b.HaveStateAt(prevHash, prevNumber) {
				return saturating.Into[uint64, N](c), nil
			}

			commit := b.storage.StateDB.RevertOne()
			if commit != nil {
				applyStateCommit(&transaction, *commit)

				numberToRevert = prevNumber
				hashToRevert = prevHash

				updateFinalized := numberToRevert < finalized

				key, err := newLookupKey(numberToRevert, hashToRevert)
				if err != nil {
					return 0, err
				}

				if updateFinalized {
					transaction.Set(columns.Meta, metakeys.FinalizedBlock, key)

					revertedFinalized[removedHash] = nil
					if finalizedState := b.blockchain.Info().FinalizedState; finalizedState != nil {
						if finalizedState.Hash == hashToRevert {
							if !(numberToRevert == 0) && b.HaveStateAt(prevHash, numberToRevert-1) {
								lookupKey, err := newLookupKey(numberToRevert-1, prevHash)
								if err != nil {
									return 0, err
								}
								transaction.Set(columns.Meta, metakeys.FinalizedState, lookupKey)
							} else {
								transaction.Remove(columns.Meta, metakeys.FinalizedState)
							}
						}
					}
				}
				transaction.Set(columns.Meta, metakeys.BestBlock, key)
				transaction.Remove(columns.KeyLookup, (*removed).Hash().Bytes())
				removeChildren(&transaction, columns.Meta, metakeys.ChildrenPrefix, hashToRevert)
				err = b.storage.db.Commit(transaction)
				if err != nil {
					return 0, err
				}

				isBest := numberToRevert < bestNumber

				b.blockchain.updateMeta(metaUpdate[H, N]{
					Hash:        hashToRevert,
					Number:      numberToRevert,
					IsBest:      isBest,
					IsFinalized: updateFinalized,
					WithState:   false,
				})
			} else {
				return saturating.Into[uint64, N](c), nil
			}

		}
		return n, nil
	}

	reverted, err := revertBlocks()
	if err != nil {
		return 0, nil, err
	}

	var revertLeaves = func() error {
		var transaction database.Transaction[hash.H256]
		b.blockchain.leavesMtx.Lock()
		defer b.blockchain.leavesMtx.Unlock()
		leaves := &b.blockchain.leaves

		leaves.Revert(hashToRevert, numberToRevert)
		leaves.PrepareTransaction(&transaction, uint32(columns.Meta), metakeys.LeafPrefix)
		err := b.storage.db.Commit(transaction)
		if err != nil {
			return err
		}
		return nil
	}

	err = revertLeaves()
	if err != nil {
		return 0, nil, err
	}

	return reverted, revertedFinalized, nil
}

func (b *Backend[H, Hasher, N, E, Header]) RemoveLeafBlock(hash H) error {
	bestHash := b.blockchain.Info().BestHash

	if bestHash == hash {
		return fmt.Errorf("%w: Can't remove best block %s", blockchain.ErrBackend, hash)
	}

	hdr, err := b.blockchain.HeaderMetadata(hash)
	if err != nil {
		return err
	}
	if !b.HaveStateAt(hash, hdr.Number) {
		return fmt.Errorf("%w: State already discarded for %s", blockchain.ErrUnknownBlock, hash)
	}

	b.blockchain.leavesMtx.Lock()
	defer b.blockchain.leavesMtx.Unlock()
	if !b.blockchain.leaves.Contains(hdr.Number, hash) {
		return fmt.Errorf("%w: Can't remove non-leaf block %s", blockchain.ErrBackend, hash)
	}

	var transaction database.Transaction[dbHash]
	commit := b.storage.StateDB.Remove(hash)
	if commit != nil {
		applyStateCommit(&transaction, *commit)
	}
	transaction.Remove(columns.KeyLookup, hash.Bytes())

	unfiltered, err := b.blockchain.Children(hdr.Parent)
	if err != nil {
		return err
	}
	var children []H
	for _, child := range unfiltered {
		if child != hash {
			children = append(children, child)
		}
	}

	var parentLeaf *H
	if len(children) == 0 {
		removeChildren(&transaction, columns.Meta, metakeys.ChildrenPrefix, hdr.Parent)
		parentLeaf = &hdr.Parent
	} else {
		writeChildren(&transaction, columns.Meta, metakeys.ChildrenPrefix, hdr.Parent, children)
	}

	removeOutcome := b.blockchain.leaves.Remove(hash, hdr.Number, parentLeaf)
	b.blockchain.leaves.PrepareTransaction(&transaction, uint32(columns.Meta), metakeys.LeafPrefix)
	err = b.storage.db.Commit(transaction)
	if err != nil {
		if removeOutcome != nil {
			b.blockchain.leaves.Undo().UndoRemove(*removeOutcome)
		}
		return err
	}
	b.blockchain.RemoveHeaderMetadata(hash)

	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) HaveStateAt(hash H, number N) bool {
	if b.isArchive {
		header, err := b.blockchain.HeaderMetadata(hash)
		if err != nil {
			return false
		}
		val, err := b.storage.Get(header.StateRoot, hashdb.EmptyPrefix)
		if err != nil {
			return false
		}
		return val != nil
	} else {
		isPruned := b.storage.StateDB.IsPruned(hash, saturating.Into[N, uint64](number))
		switch isPruned {
		case statedb.IsPrunedPruned:
			return false
		case statedb.IsPrunedNotPruned:
			return true
		case statedb.IsPrunedMaybePruned:
			header, err := b.blockchain.HeaderMetadata(hash)
			if err != nil {
				return false
			}
			val, err := b.storage.Get(header.StateRoot, hashdb.EmptyPrefix)
			if err != nil {
				return false
			}
			return val != nil
		default:
			panic("unreachable")
		}
	}
}

func (b *Backend[H, Hasher, N, E, Header]) stateAt(hash H) (refTrackingState[H, Hasher], error) {
	b.blockchain.metaMtx.RLock()
	if hash == b.blockchain.meta.GenesisHash {
		b.genesisStateMtx.RLock()
		if genesisState := b.genesisState; genesisState != nil {
			root := genesisState.root
			var localCache *cache.LocalTrieCache[H]
			if b.sharedTrieCache != nil {
				lcc := b.sharedTrieCache.LocalTrieCache()
				localCache = &lcc
			}
			dbState := statemachine.NewTrieBackend[H, Hasher](&b.storage, root, localCache, nil)
			state := refTrackingState[H, Hasher]{
				state:      DBState[H, Hasher]{dbState},
				storage:    b.storage,
				parentHash: nil,
			}
			b.genesisStateMtx.RUnlock()
			b.blockchain.metaMtx.RUnlock()
			return state, nil
		}
		b.genesisStateMtx.RUnlock()
	}
	b.blockchain.metaMtx.RUnlock()

	hdr, err := b.blockchain.HeaderMetadata(hash)
	if err != nil {
		return refTrackingState[H, Hasher]{}, err
	}

	var hint = func() bool {
		val := b.storage.db.Get(columns.State, hdr.StateRoot.Bytes())
		return val != nil
	}

	err = b.storage.StateDB.Pin(hash, saturating.Into[N, uint64](hdr.Number), hint)
	if err != nil {
		return refTrackingState[H, Hasher]{},
			fmt.Errorf("%w: State already discarded for %s", blockchain.ErrUnknownBlock, hash)
	}
	root := hdr.StateRoot
	var localCache *cache.LocalTrieCache[H]
	if b.sharedTrieCache != nil {
		lcc := b.sharedTrieCache.LocalTrieCache()
		localCache = &lcc
	}
	dbState := statemachine.NewTrieBackend[H, Hasher](&b.storage, root, localCache, nil)
	state := refTrackingState[H, Hasher]{
		state:      DBState[H, Hasher]{dbState},
		storage:    b.storage,
		parentHash: &hash,
	}
	return state, nil
}

func (b *Backend[H, Hasher, N, E, Header]) StateAt(hash H) (statemachine.Backend[H, Hasher], error) {
	state, err := b.stateAt(hash)
	if err != nil {
		return nil, err
	}
	backend := state.state
	return &backend, nil
}

func (b *Backend[H, Hasher, N, E, Header]) Blockchain() blockchain.Backend[H, N, Header, E] {
	return b.blockchain
}

func (b *Backend[H, Hasher, N, E, Header]) GetImportLock() *sync.RWMutex {
	return &b.importLock
}

func (b *Backend[H, Hasher, N, E, Header]) RequiresFullSync() bool {
	pruningMode := b.storage.StateDB.PruningMode()
	switch pruningMode.(type) {
	case statedb.PruningModeArchiveAll:
		return true
	case statedb.PruningModeArchiveCanonical:
		return true
	case statedb.PruningModeConstrained:
		return false
	default:
		panic("unreachable")
	}
}

func (b *Backend[H, Hasher, N, E, Header]) PinBlock(hash H) error {
	var hint = func() bool {
		hdr, err := b.blockchain.HeaderMetadata(hash)
		if err != nil {
			return false
		}
		val := b.storage.db.Get(columns.State, hdr.StateRoot.Bytes())
		return val != nil
	}

	number, err := b.blockchain.Number(hash)
	if err != nil {
		return err
	}
	if number != nil {
		err := b.storage.StateDB.Pin(hash, saturating.Into[N, uint64](*number), hint)
		if err != nil {
			return fmt.Errorf("%w: State already discarded for %s", blockchain.ErrUnknownBlock, hash)
		}
	} else {
		return fmt.Errorf("%w: Can not pin block with hash %s. Block not found", blockchain.ErrUnknownBlock, hash)
	}

	if _, ok := b.blocksPruning.(BlocksPruningKeepAll); !ok {
		// Only increase reference count for this hash. Value is loaded once we prune.
		b.blockchain.bumpRef(hash)
	}
	return nil
}

func (b *Backend[H, Hasher, N, E, Header]) UnpinBlock(hash H) {
	b.storage.StateDB.Unpin(hash)

	if _, ok := b.blocksPruning.(BlocksPruningKeepAll); !ok {
		// Only increase reference count for this hash. Value is loaded once we prune.
		b.blockchain.unpin(hash)
	}
}
