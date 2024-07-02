// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"fmt"
	"log"
	"sync"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/db/columns"
	"github.com/ChainSafe/gossamer/internal/client/db/metakeys"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/li1234yun/gods-generic/maps/linkedhashmap"
)

const numCachedHeaders = 8

// An extrinsic entry in the database.
type dbExtrinsic[E runtime.Extrinsic] struct {
	inner any
}

type dbExtrinsicValues[E runtime.Extrinsic] interface {
	dbExtrinsicIndexed | dbExtrinsicFull[E]
}

func setDbExtrinsic[E runtime.Extrinsic, Value dbExtrinsicValues[E]](mvdt *dbExtrinsic[E], value Value) {
	mvdt.inner = value
}

func (mvdt *dbExtrinsic[E]) SetValue(value any) (err error) {
	switch value := value.(type) {
	case dbExtrinsicIndexed:
		setDbExtrinsic[E](mvdt, value)
		return
	case dbExtrinsicFull[E]:
		setDbExtrinsic[E](mvdt, value)
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
	return nil, scale.ErrUnsupportedVaryingDataTypeValue
}

// Extrinsic that contains indexed data.
type dbExtrinsicIndexed struct {
	// Hash of the indexed part.
	Hash hash.H256
	// Extrinsic header.
	Header []byte
}
type dbExtrinsicFull[E runtime.Extrinsic] struct {
	Extrinsic E
}

type metaUpdate[H, N any] struct {
	Hash        H
	Number      N
	IsBest      bool
	IsFinalized bool
	WithState   bool
}

// blockchainDB is the block database.
type blockchainDB[H runtime.Hash, N runtime.Number, E runtime.Extrinsic, Header runtime.Header[N, H]] struct {
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

func newBlockchainDB[
	H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H], E runtime.Extrinsic, Header runtime.Header[N, H],
](db database.Database[hash.H256]) (*blockchainDB[H, N, E, Header], error) {
	meta, err := readMeta[H, N, *generic.Header[N, H, Hasher]](db, columns.Header)
	if err != nil {
		return nil, err
	}
	leaves, err := api.NewLeafSetFromDB[H, N](db, uint32(columns.Meta), metakeys.LeafPrefix)
	if err != nil {
		return nil, err
	}
	return &blockchainDB[H, N, E, Header]{
		db:                  db,
		leaves:              leaves,
		meta:                meta,
		headerMetadataCache: blockchain.NewHeaderMetadataCache[H, N](),
		headerCache:         *linkedhashmap.New[H, *runtime.Header[N, H]](),
		pinnedBlocksCache:   newPinnedBlocksCache[H](),
	}, nil
}

func (bdb *blockchainDB[H, N, E, Header]) updateMeta(update metaUpdate[H, N]) {
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
			bdb.meta.FinalizedState = &finalizedState[H, N]{update.Hash, update.Number}
		}
		bdb.meta.FinalizedNumber = update.Number
		bdb.meta.FinalizedHash = update.Hash
	}
}

func (bdb *blockchainDB[H, N, E, Header]) updateBlockGap(gap *[2]N) {
	bdb.metaMtx.Lock()
	defer bdb.metaMtx.Unlock()
	bdb.meta.BlockGap = gap
}

// Empty the cache of pinned items.
func (bdb *blockchainDB[H, N, E, Header]) clearPinningCache() {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	bdb.pinnedBlocksCache.Clear()
}

// Load a justification into the cache of pinned items.
// Reference count of the item will not be increased. Use this
// to load values for items into the cache which have already been pinned.
func (bdb *blockchainDB[H, N, E, Header]) insertJustifcationsIfPinned(hash H, justification runtime.Justification) {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	if !bdb.pinnedBlocksCache.Contains(hash) {
		return
	}

	justifications := runtime.Justifications{justification}
	bdb.pinnedBlocksCache.InsertJustifications(hash, justifications)
}

// Load a justification from the db into the cache of pinned items.
// Reference count of the item will not be increased. Use this
// to load values for items into the cache which have already been pinned.
func (bdb *blockchainDB[H, N, E, Header]) insertPersistedJustificationsIfPinned(hash H) error {
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

// Load a block body from the db into the cache of pinned items.
// Reference count of the item will not be increased. Use this
// to load values for items items into the cache which have already been pinned.
func (bdb *blockchainDB[H, N, E, Header]) insertPersistedBodyIfPinned(hash H) error {
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

// Bump reference count for pinned item.
func (bdb *blockchainDB[H, N, E, Header]) bumpRef(hash H) {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	bdb.pinnedBlocksCache.Pin(hash)
}

// Decrease reference count for pinned item and remove if reference count is 0.
func (bdb *blockchainDB[H, N, E, Header]) unpin(hash H) {
	bdb.pinnedBlocksCacheMtx.Lock()
	defer bdb.pinnedBlocksCacheMtx.Unlock()
	bdb.pinnedBlocksCache.Unpin(hash)
}

func (bdb *blockchainDB[H, N, E, Header]) justificationsUncached(hash H) (runtime.Justifications, error) {
	blockID := generic.NewBlockID[H, N](generic.BlockIDHash[H]{Inner: hash})
	justificationsBytes, err := readDB[H, N](bdb.db, columns.KeyLookup, columns.Justifications, blockID)
	if err != nil {
		return nil, err
	}
	if justificationsBytes != nil {
		var justifications runtime.Justifications
		err := scale.Unmarshal(justificationsBytes, &justifications)
		if err != nil {
			return nil, err
		}
		return justifications, nil
	}
	return nil, nil
}

func (bdb *blockchainDB[H, N, E, Header]) bodyUncached(hash H) ([]runtime.Extrinsic, error) {
	blockID := generic.NewBlockID[H, N](generic.BlockIDHash[H]{Inner: hash})
	bodyBytes, err := readDB[H, N](bdb.db, columns.KeyLookup, columns.Body, blockID)
	if err != nil {
		return nil, err
	}
	if bodyBytes != nil {
		var extrinsics []E
		err := scale.Unmarshal(bodyBytes, &extrinsics)
		if err != nil {
			return nil, err
		}
		var body []runtime.Extrinsic
		for _, e := range extrinsics {
			body = append(body, e)
		}
		return body, nil
	}

	indexBytes, err := readDB[H, N](bdb.db, columns.KeyLookup, columns.BodyIndex, blockID)
	if err != nil {
		return nil, err
	}
	if indexBytes == nil {
		return nil, nil
	}
	var index []dbExtrinsic[E]
	err = scale.Unmarshal(indexBytes, index)
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
			t := bdb.db.Get(columns.Transaction, hash.Bytes())
			if t != nil {
				input := joinInput(dbex.Header, t)
				var ex E
				err := scale.Unmarshal(input, &ex)
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
	return body, nil
}

func (bdb *blockchainDB[H, N, E, Header]) cacheHeader(hash H, header *runtime.Header[N, H]) {
	bdb.headerCache.Put(hash, header)
	for bdb.headerCache.Size() > numCachedHeaders {
		iterator := bdb.headerCache.Iterator()
		if !iterator.First() {
			panic("headerCache is empty")
		}
		bdb.headerCache.Remove(iterator.Key())
	}
}

func (bdb *blockchainDB[H, N, E, Header]) Header(hash H) (runtime.Header[N, H], error) {
	bdb.headerCacheMtx.Lock()
	defer bdb.headerCacheMtx.Unlock()
	val, ok := bdb.headerCache.Get(hash)
	if ok {
		// TODO: create issue to fork linkedhashmap, and add cache.get_refresh(&hash)
		bdb.headerCache.Remove(hash)
		bdb.headerCache.Put(hash, val)
		return *val, nil
	}
	header, err := readHeader[H, N, Header](
		bdb.db,
		columns.KeyLookup,
		columns.Header,
		generic.BlockIDHash[H]{Inner: hash},
	)
	if err != nil {
		return nil, err
	}
	bdb.cacheHeader(hash, header)
	return *header, nil
}

func (bdb *blockchainDB[H, N, E, Header]) Info() blockchain.Info[H, N] {
	bdb.metaMtx.RLock()
	defer bdb.metaMtx.RUnlock()
	info := blockchain.Info[H, N]{
		BestHash:        bdb.meta.BestHash,
		BestNumber:      bdb.meta.BestNumber,
		GenesisHash:     bdb.meta.GenesisHash,
		FinalizedHash:   bdb.meta.FinalizedHash,
		FinalizedNumber: bdb.meta.FinalizedNumber,
		FinalizedState: &struct {
			Hash   H
			Number N
		}{bdb.meta.FinalizedState.Hash, bdb.meta.FinalizedState.Number},
		NumberLeaves: bdb.leaves.Count(),
		BlockGap:     bdb.meta.BlockGap,
	}
	return info
}

func (bdb *blockchainDB[H, N, E, Header]) Status(hash H) (blockchain.BlockStatus, error) {
	header, err := bdb.Header(hash)
	if err != nil {
		return 0, err
	}
	if header != nil {
		return blockchain.BlockStatusInChain, nil
	}
	return blockchain.BlockStatusUnknown, nil
}

func (bdb *blockchainDB[H, N, E, Header]) Number(hash H) (*N, error) {
	meta, err := bdb.HeaderMetadata(hash)
	if err != nil {
		return nil, err
	}
	return &meta.Number, nil
}

func (bdb *blockchainDB[H, N, E, Header]) Hash(number N) (*H, error) {
	header, err := readHeader[H, N, Header](
		bdb.db,
		columns.KeyLookup,
		columns.Header,
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

func (bdb *blockchainDB[H, N, E, Header]) BlockHashFromID(id generic.BlockID) (*H, error) {
	switch id := id.(type) {
	case generic.BlockIDHash[H]:
		return &id.Inner, nil
	case generic.BlockIDNumber[N]:
		return bdb.Hash(id.Inner)
	default:
		panic("unsupported block id type")
	}
}

func (bdb *blockchainDB[H, N, E, Header]) BlockNumberFromID(id generic.BlockID) (*N, error) {
	switch id := id.(type) {
	case generic.BlockIDHash[H]:
		return bdb.Number(id.Inner)
	case generic.BlockIDNumber[N]:
		return &id.Inner, nil
	default:
		panic("unsupported block id type")
	}
}

func (bdb *blockchainDB[H, N, E, Header]) Body(hash H) ([]runtime.Extrinsic, error) {
	bdb.pinnedBlocksCacheMtx.RLock()
	defer bdb.pinnedBlocksCacheMtx.RUnlock()
	body := bdb.pinnedBlocksCache.Body(hash)
	if body != nil {
		return *body, nil
	}

	return bdb.bodyUncached(hash)
}

func (bdb *blockchainDB[H, N, E, Header]) Justifications(hash H) (runtime.Justifications, error) {
	bdb.pinnedBlocksCacheMtx.RLock()
	defer bdb.pinnedBlocksCacheMtx.RUnlock()
	justifications := bdb.pinnedBlocksCache.Justifications(hash)
	if justifications != nil {
		return justifications, nil
	}

	return bdb.justificationsUncached(hash)
}

func (bdb *blockchainDB[H, N, E, Header]) LastFinalized() (H, error) {
	bdb.metaMtx.RLock()
	defer bdb.metaMtx.RUnlock()
	return bdb.meta.FinalizedHash, nil
}

func (bdb *blockchainDB[H, N, E, Header]) Leaves() ([]H, error) {
	bdb.leavesMtx.RLock()
	defer bdb.leavesMtx.RUnlock()
	return bdb.leaves.Hashes(), nil
}

func (bdb *blockchainDB[H, N, E, Header]) DisplacedLeavesAfterFinalizing(blockNumber N) ([]H, error) {
	bdb.leavesMtx.RLock()
	defer bdb.leavesMtx.RUnlock()
	return bdb.leaves.DisplacedByFinalHeight(blockNumber).Leaves(), nil
}

func (bdb *blockchainDB[H, N, E, Header]) Children(parentHash H) ([]H, error) {
	return readChildren[H](bdb.db, columns.Meta, metakeys.ChildrenPrefix, parentHash)
}

func (bdb *blockchainDB[H, N, E, Header]) LongestContaining(baseHash H, importLock *sync.RWMutex) (*H, error) {
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
		if info.FinalizedNumber > baseHeader.Number() {
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

			if currentHeader.Number() < baseHeader.Number() {
				break
			}

			currentHash = currentHeader.ParentHash()
		}
	}

	// header may be on a dead fork -- the only leaves that are considered are
	// those which can still be finalized.
	//
	// FIXME: substrate issue #1558 only issue this warning when not on a dead fork
	log.Printf("WARN: Block %v exists in chain but not found when following all leaves backwards\n", baseHash)
	return nil, nil
}

func (bdb *blockchainDB[H, N, E, Header]) IndexedTransaction(hash H) ([]byte, error) {
	return bdb.db.Get(columns.Transaction, hash.Bytes()), nil
}

func (bdb *blockchainDB[H, N, E, Header]) HasIndexedTransaction(hash H) (bool, error) {
	return bdb.db.Contains(columns.Transaction, hash.Bytes()), nil
}

func (bdb *blockchainDB[H, N, E, Header]) BlockIndexedBody(hash H) ([][]byte, error) {
	bodyBytes, err := readDB[H, N](bdb.db, columns.KeyLookup, columns.BodyIndex, generic.BlockIDHash[H]{Inner: hash})
	if err != nil {
		return nil, err
	}
	if bodyBytes == nil {
		return nil, err
	}
	index := make([]dbExtrinsic[E], 0)
	err = scale.Unmarshal(bodyBytes, &index)
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
		t := bdb.db.Get(columns.Transaction, indexed.Hash.Bytes())
		if t == nil {
			return nil, fmt.Errorf("Missing indexed transaction %v", hash)
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}

func (bdb *blockchainDB[H, N, E, Header]) HeaderMetadata(hash H) (blockchain.CachedHeaderMetadata[H, N], error) {
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
	headerMetadata := blockchain.NewCachedHeaderMetadata(header)
	bdb.headerMetadataCache.InsertHeaderMetadata(headerMetadata.Hash, headerMetadata)
	return headerMetadata, nil
}

func (bdb *blockchainDB[H, N, E, Header]) InsertHeaderMetadata(hash H, metadata blockchain.CachedHeaderMetadata[H, N]) {
	bdb.headerMetadataCache.InsertHeaderMetadata(hash, metadata)
}

func (bdb *blockchainDB[H, N, E, Header]) RemoveHeaderMetadata(hash H) {
	bdb.headerCacheMtx.Lock()
	defer bdb.headerCacheMtx.Unlock()
	bdb.headerCache.Remove(hash)
	bdb.headerMetadataCache.RemoveHeaderMetadata(hash)
}
