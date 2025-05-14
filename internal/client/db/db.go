// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/db/columns"
	"github.com/ChainSafe/gossamer/internal/client/db/metakeys"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ugurcsen/gods-generic/maps/linkedhashmap"
)

// Hash type that this backend uses for the database.
type dbHash = hash.H256

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

func newDbExtrinsic[E runtime.Extrinsic, Value dbExtrinsicValues[E]](val Value) dbExtrinsic[E] {
	dbe := dbExtrinsic[E]{}
	setDbExtrinsic(&dbe, val)
	return dbe
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
	headerCache          linkedhashmap.Map[H, *Header]
	headerCacheMtx       sync.Mutex
	pinnedBlocksCache    pinnedBlocksCache[H, E]
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
		headerCache:         *linkedhashmap.New[H, *Header](),
		pinnedBlocksCache:   newPinnedBlocksCache[H, E](),
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

func (bdb *blockchainDB[H, N, E, Header]) updateBlockGap(gap *blockchain.BlockGap[N]) {
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
	blockID := generic.NewBlockID[H, N](generic.BlockIDHash[H]{Hash: hash})
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

func (bdb *blockchainDB[H, N, E, Header]) bodyUncached(hash H) ([]E, error) {
	blockID := generic.NewBlockID[H, N](generic.BlockIDHash[H]{Hash: hash})
	bodyBytes, err := readDB[H, N](bdb.db, columns.KeyLookup, columns.Body, blockID)
	if err != nil {
		return nil, err
	}
	if bodyBytes != nil {
		var body []E
		err := scale.Unmarshal(bodyBytes, &body)
		if err != nil {
			return nil, err
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
	err = scale.Unmarshal(indexBytes, &index)
	if err != nil {
		return nil, err
	}
	var body []E
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
					return nil, fmt.Errorf("error decoding indexed extrinsic: %w", err)
				}
				body = append(body, ex)
			} else {
				return nil, fmt.Errorf("missing indexed transaction %v", hash)
			}
		case dbExtrinsicFull[E]:
			body = append(body, dbex.Extrinsic)
		}
	}
	return body, nil
}

func (bdb *blockchainDB[H, N, E, Header]) cacheHeader(hash H, header *Header) {
	bdb.headerCache.Put(hash, header)
	for bdb.headerCache.Size() > numCachedHeaders {
		iterator := bdb.headerCache.Iterator()
		if !iterator.First() {
			panic("headerCache is empty")
		}
		bdb.headerCache.Remove(iterator.Key())
	}
}

func (bdb *blockchainDB[H, N, E, Header]) header(hash H) (*Header, error) {
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
		columns.KeyLookup,
		columns.Header,
		generic.BlockIDHash[H]{Hash: hash},
	)
	if err != nil {
		return header, err
	}
	bdb.cacheHeader(hash, header)
	return header, nil
}

func (bdb *blockchainDB[H, N, E, Header]) Header(hash H) (*Header, error) {
	header, err := bdb.header(hash)
	if err != nil {
		return nil, err
	}
	return header, nil
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
		NumberLeaves:    bdb.leaves.Count(),
		BlockGap:        bdb.meta.BlockGap,
	}
	if bdb.meta.FinalizedState != nil {
		info.FinalizedState = &struct {
			Hash   H
			Number N
		}{bdb.meta.FinalizedState.Hash, bdb.meta.FinalizedState.Number}
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
		generic.BlockIDNumber[N]{Number: number},
	)
	if err != nil {
		return nil, err
	}
	if header != nil {
		h := (*header).Hash()
		return &h, nil
	}
	return nil, nil
}

func (bdb *blockchainDB[H, N, E, Header]) BlockHashFromID(id generic.BlockID) (*H, error) {
	switch id := id.(type) {
	case generic.BlockIDHash[H]:
		return &id.Hash, nil
	case generic.BlockIDNumber[N]:
		return bdb.Hash(id.Number)
	default:
		panic("unsupported block id type")
	}
}

func (bdb *blockchainDB[H, N, E, Header]) BlockNumberFromID(id generic.BlockID) (*N, error) {
	switch id := id.(type) {
	case generic.BlockIDHash[H]:
		return bdb.Number(id.Hash)
	case generic.BlockIDNumber[N]:
		return &id.Number, nil
	default:
		panic("unsupported block id type")
	}
}

func (bdb *blockchainDB[H, N, E, Header]) Body(hash H) ([]E, error) {
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
		return *justifications, nil
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

func (bdb *blockchainDB[H, N, E, Header]) DisplacedLeavesAfterFinalizing(
	finalizedBlockHash H, finalizedBlockNumber N,
) (blockchain.DisplacedLeavesAfterFinalization[H, N], error) {
	bdb.leavesMtx.RLock()
	defer bdb.leavesMtx.RUnlock()
	leaves, err := bdb.Leaves()
	if err != nil {
		return blockchain.DisplacedLeavesAfterFinalization[H, N]{}, err
	}

	now := time.Now()
	logger.Debugf(`
		target=db::blockchain,
		leaves=%v,
		finalized_block_hash=%s,
		finalized_block_number=%d,
		Checking for displaced leaves after finalization.
	`,
		leaves,
		finalizedBlockHash.String(),
		finalizedBlockNumber,
	)

	// If we have only one leaf there are no forks, and we can return early.
	if finalizedBlockNumber == 0 || len(leaves) == 1 {
		return blockchain.DisplacedLeavesAfterFinalization[H, N]{}, nil
	}

	// Store hashes of finalized blocks for quick checking later, the last block is the
	// finalized one
	finalizedChain := list.New()
	currentFinalized, err := bdb.HeaderMetadata(finalizedBlockHash)

	if err != nil {
		if errors.Is(err, blockchain.ErrUnknownBlock) {
			logger.Debugf(`
				target=db::blockchain,
				hash=%s,
				elapsed=%f,
				Tried to fetch unknown block, block ancestry has gaps.
			`,
				finalizedBlockHash.String(),
				time.Since(now),
			)
			return blockchain.DisplacedLeavesAfterFinalization[H, N]{}, nil
		}
		logger.Debugf(`
				target=db::blockchain,
				hash=%s,
				err=%w,
				elapsed=%f,
				Failed to fetch block.
			`,
			finalizedBlockHash.String(),
			err,
			time.Since(now),
		)
		return blockchain.DisplacedLeavesAfterFinalization[H, N]{}, err
	}

	finalizedChain.PushFront(minimalBlockMetadata[H, N]{
		number: currentFinalized.Number,
		hash:   currentFinalized.Hash,
		parent: currentFinalized.Parent,
	})

	// Local cache is a performance optimization in case of finalized block deep below the
	// tip of the chain with a lot of leaves above finalized block
	localCache := make(map[H]minimalBlockMetadata[H, N])

	result := blockchain.DisplacedLeavesAfterFinalization[H, N]{
		DisplacedLeaves: make([]blockchain.HashNumber[H, N], 0),
		DisplacedBlocks: make([]H, 0),
	}

	genesisHash := bdb.Info().GenesisHash

	for _, leafHash := range leaves {
		headerMetadata, err := bdb.HeaderMetadata(leafHash)
		if err != nil {
			logger.Debugf(`
				target=db::blockchain,
				leaf_hash=%s,
				err=%w,
				elapsed=%f,
				Failed to fetch leaf header.
			`,
				leafHash.String(),
				err,
				time.Since(now),
			)
			return blockchain.DisplacedLeavesAfterFinalization[H, N]{}, err
		}

		currentHeaderMetadata := minimalBlockMetadata[H, N]{
			number: headerMetadata.Number,
			hash:   headerMetadata.Hash,
			parent: headerMetadata.Parent,
		}

		leafNumber := headerMetadata.Number

		// The genesis block is part of the canonical chain
		if leafHash == genesisHash {
			result.DisplacedLeaves = append(result.DisplacedLeaves, blockchain.HashNumber[H, N]{
				Hash:   leafHash,
				Number: leafNumber,
			})
			logger.Debugf(`
				target=db::blockchain,
				leaf_hash=%s,
				elapsed=%f,
				Added genesis leaf to displaced leaves.
			`,
				leafHash.String(),
				time.Since(now),
			)
			continue
		}

		logger.Debugf(`
				target=db::blockchain,
				leaf_number=%d,
				leaf_hash=%s,
				elapsed=%f,
				Handle displaced leaf.
			`,
			leafNumber,
			leafHash.String(),
			time.Since(now),
		)

		// Collect all block hashes until the height of the finalized block
		displacedBlocksCandidates := make([]H, 0)
		for currentHeaderMetadata.number > finalizedBlockNumber {
			displacedBlocksCandidates = append(displacedBlocksCandidates, currentHeaderMetadata.hash)

			parentHash := currentHeaderMetadata.parent
			val, has := localCache[parentHash]

			if has {
				currentHeaderMetadata = val
			} else {
				headerMetadata, err := bdb.HeaderMetadata(leafHash)
				if err != nil {
					logger.Debugf(`
						target=db::blockchain,
						err=%w,
						parent_hash=%s,
						leaf_hash=%s,
						elapsed=%f,
						Failed to fetch parent header during leaf tracking.
					`,
						err,
						parentHash.String(),
						leafHash.String(),
						time.Since(now),
					)
					return blockchain.DisplacedLeavesAfterFinalization[H, N]{}, err
				}

				currentHeaderMetadata = minimalBlockMetadata[H, N]{
					number: headerMetadata.Number,
					hash:   headerMetadata.Hash,
					parent: headerMetadata.Parent,
				}
				// Cache locally in case more branches above finalized block reference
				// the same block hash
				localCache[parentHash] = currentHeaderMetadata
			}
		}

		// If points back to the finalized header then nothing left to do, this leaf will be
		// checked again later
		if currentHeaderMetadata.hash == finalizedBlockHash {
			logger.Debugf(`
					target=db::blockchain,
					leaf_hash=%s,
					elapsed=%f,
					Leaf points to the finalized header, skipping for now.
				`,
				leafHash.String(),
				time.Since(now),
			)
			continue
		}

		// We reuse `displaced_blocks_candidates` to store the current metadata.
		// This block is not displaced if there is a gap in the ancestry. We
		// check for this gap later.
		displacedBlocksCandidates = append(displacedBlocksCandidates, currentHeaderMetadata.hash)

		logger.Debugf(`
				target=db::blockchain,
				current_hash=%s,
				current_num=%d,
				finalized_block_number=%d,
				elapsed=%f,
				Looking for path from finalized block number to current leaf number
			`,
			currentHeaderMetadata.hash.String(),
			currentHeaderMetadata.number,
			finalizedBlockNumber,
			time.Since(now),
		)

		// Collect the rest of the displaced blocks of leaf branch
		for distanceFromFinalized := 1; ; distanceFromFinalized++ {
			var finalizedChainBlockNumber N
			var finalizedChainBlockHash H

			header, ok := findNthFromEnd[minimalBlockMetadata[H, N]](finalizedChain, distanceFromFinalized)
			if ok {
				finalizedChainBlockNumber = header.number
				finalizedChainBlockHash = header.hash
			} else {
				toFetch := finalizedChain.Front()
				if toFetch == nil {
					panic("expect not empty")
				}

				headerMetadata, err := bdb.HeaderMetadata(toFetch.Value.(minimalBlockMetadata[H, N]).hash)
				if err != nil {
					if errors.Is(err, blockchain.ErrUnknownBlock) {
						logger.Debugf(`
								target=db::blockchain,
								distance_from_finalized=%d,
								hash=%s,
								number=%d,
								elapsed=%f,
								Tried to fetch unknown block, block ancestry has gaps.
							`,
							distanceFromFinalized,
							toFetch.Value.(minimalBlockMetadata[H, N]).hash.String(),
							toFetch.Value.(minimalBlockMetadata[H, N]).number,
							time.Since(now),
						)
						break
					}
					logger.Debugf(`
							target=db::blockchain,
							hash=%s,
							number=%d,
							err=%w,
							elapsed=%f,
							Failed to fetch header for parent hash.
						`,
						toFetch.Value.(minimalBlockMetadata[H, N]).hash.String(),
						toFetch.Value.(minimalBlockMetadata[H, N]).number,
						err,
						time.Since(now),
					)
					return blockchain.DisplacedLeavesAfterFinalization[H, N]{}, err
				}

				metadata := minimalBlockMetadata[H, N]{
					number: headerMetadata.Number,
					hash:   headerMetadata.Hash,
					parent: headerMetadata.Parent,
				}
				finalizedChain.PushFront(metadata)

				finalizedChainBlockNumber = metadata.number
				finalizedChainBlockHash = metadata.hash
			}

			if currentHeaderMetadata.hash == finalizedChainBlockHash {
				// Found the block on the finalized chain, nothing left to do
				result.DisplacedLeaves = append(result.DisplacedLeaves, blockchain.HashNumber[H, N]{
					Hash:   leafHash,
					Number: leafNumber,
				})

				logger.Debugf(`
						target=db::blockchain,
						leaf_hash=%s,
						elapsed=%f,
						Leaf is ancestor of finalized block.
					`,
					leafHash.String(),
					time.Since(now),
				)
				break
			}

			if currentHeaderMetadata.number <= finalizedChainBlockNumber {
				// Skip more blocks until we get all blocks on finalized chain until the height
				// of the parent block
				continue
			}

			parentHash := currentHeaderMetadata.parent

			if finalizedChainBlockHash == parentHash {
				// Reached finalized chain, nothing left to do
				result.DisplacedBlocks = append(result.DisplacedBlocks, displacedBlocksCandidates...)
				result.DisplacedLeaves = append(result.DisplacedLeaves, blockchain.HashNumber[H, N]{
					Hash:   leafHash,
					Number: leafNumber,
				})

				logger.Debugf(`
						target=db::blockchain,
						leaf_hash=%s,
						elapsed=%f,
						Found displaced leaf.
					`,
					leafHash.String(),
					time.Since(now),
				)
				break
			}

			// Store displaced block and look deeper for block on finalized chain
			logger.Debugf(`
					target=db::blockchain,
					parent_hash=%s,
					elapsed=%f,
					Found displaced block. Looking further.
				`,
				parentHash.String(),
				time.Since(now),
			)

			displacedBlocksCandidates = append(displacedBlocksCandidates, parentHash)

			headerMetadata, err := bdb.HeaderMetadata(parentHash)
			if err != nil {
				logger.Debugf(`
					target=db::blockchain,
					err=%w,
					parent_hash=%s,
					elapsed=%f,
					Failed to fetch header for parent during displaced block collection
				`,
					err,
					parentHash.String(),
					time.Since(now),
				)
				return blockchain.DisplacedLeavesAfterFinalization[H, N]{}, err
			}

			currentHeaderMetadata = minimalBlockMetadata[H, N]{
				number: headerMetadata.Number,
				hash:   headerMetadata.Hash,
				parent: headerMetadata.Parent,
			}
		}
	}

	// There could be duplicates shared by multiple branches, clean them up
	result.SortAndDedupDisplacedBlocks()

	logger.Debugf(`
		target=db::blockchain,
		finalized_block_hash=%s,
		finalized_block_number=%d,
		result=%v,
		elapsed=%f,
		Finished checking for displaced leaves after finalization.
	`,
		finalizedBlockHash.String(),
		finalizedBlockNumber,
		result,
		time.Since(now),
	)

	return result, nil
}

func (bdb *blockchainDB[H, N, E, Header]) Children(parentHash H) ([]H, error) {
	return readChildren(bdb.db, columns.Meta, metakeys.ChildrenPrefix, parentHash)
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
		if info.FinalizedNumber > (*baseHeader).Number() {
			// baseHeader is on a dead fork.
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
				return nil, fmt.Errorf("failed to get header for hash %v", currentHash)
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
	// FIXME: substrate issue #1558 only issue this warning when not on a dead fork
	log.Printf("WARN: Block %v exists in chain but not found when following all leaves backwards", baseHash)
	return nil, nil
}

func (bdb *blockchainDB[H, N, E, Header]) IndexedTransaction(hash H) ([]byte, error) {
	return bdb.db.Get(columns.Transaction, hash.Bytes()), nil
}

func (bdb *blockchainDB[H, N, E, Header]) HasIndexedTransaction(hash H) (bool, error) {
	return bdb.db.Contains(columns.Transaction, hash.Bytes()), nil
}

func (bdb *blockchainDB[H, N, E, Header]) BlockIndexedBody(hash H) ([][]byte, error) {
	bodyBytes, err := readDB[H, N](bdb.db, columns.KeyLookup, columns.BodyIndex, generic.BlockIDHash[H]{Hash: hash})
	if err != nil {
		return nil, err
	}
	if bodyBytes == nil {
		return nil, err
	}
	index := make([]dbExtrinsic[E], 0)
	err = scale.Unmarshal(bodyBytes, &index)
	if err != nil {
		return nil, fmt.Errorf("error decoding body list %w", err)
	}
	var transactions [][]byte
	for _, ex := range index {
		hash, err := ex.Value()
		if err != nil {
			return nil, fmt.Errorf("error decoding body list %w", err)
		}
		indexed, ok := hash.(dbExtrinsicIndexed)
		if !ok {
			continue
		}
		t := bdb.db.Get(columns.Transaction, indexed.Hash.Bytes())
		if t == nil {
			return nil, fmt.Errorf("missing indexed transaction %v", hash)
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
		return blockchain.CachedHeaderMetadata[H, N]{},
			fmt.Errorf("%w: header was not found in the database: %v", blockchain.ErrUnknownBlock, hash)
	}
	headerMetadata := blockchain.NewCachedHeaderMetadata(*header)
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

func findNthFromEnd[E any](l *list.List, n int) (E, bool) {
	if l.Len() == 0 || n >= l.Len() {
		return *new(E), false
	}

	currentElement := l.Back()
	count := 0

	for currentElement != nil && count < n {
		currentElement = currentElement.Prev()
		count++
	}

	if currentElement == nil {
		return *new(E), false
	}

	return currentElement.Value.(E), true
}
