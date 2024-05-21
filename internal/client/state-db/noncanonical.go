// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statedb

import (
	"fmt"
	"log"
	"math/bits"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/gammazero/deque"
)

var lastCanonical = []byte("last_canonical")

const maxBlocksPerLevel uint64 = 32

// nonCanonicalOverlay maintains trees of block overlays and allows discarding trees/roots.
// The overlays are added in `Insert` and removed in `Canonicalize`.
type nonCanonicalOverlay[BlockHash Hash, Key Hash] struct {
	lastCanonicalized *hashBlock[BlockHash]
	levels            deque.Deque[overlayLevel[BlockHash, Key]]
	parents           map[BlockHash]BlockHash
	values            map[Key]struct {
		count uint32
		value DBValue
	}
	// would be deleted but kept around because block is pinned, ref counted.
	pinned           map[BlockHash]uint32
	pinnedInsertions map[BlockHash]struct {
		keys  []Key
		count uint32
	}
	pinnedCanonicalized []BlockHash
}

type hashBlock[BlockHash Hash] struct {
	Hash  BlockHash
	Block uint64
}

// NewnonCanonicalOverlay is constructor for nonCanonicalOverlay
func newNonCanonicalOverlay[BlockHash Hash, Key Hash](db MetaDB) (nonCanonicalOverlay[BlockHash, Key], error) {
	lastCanonicalizedMeta, err := db.GetMeta(toMetaKey(lastCanonical, struct{}{}))
	if err != nil {
		return nonCanonicalOverlay[BlockHash, Key]{}, err
	}
	var lastCanonicalized *hashBlock[BlockHash]
	if lastCanonicalizedMeta != nil {
		bhk := hashBlock[BlockHash]{}
		err := scale.Unmarshal(*lastCanonicalizedMeta, &bhk)
		if err != nil {
			return nonCanonicalOverlay[BlockHash, Key]{}, err
		}
		lastCanonicalized = &bhk
	}
	var levels deque.Deque[overlayLevel[BlockHash, Key]]
	var parents = make(map[BlockHash]BlockHash)
	var values = make(map[Key]struct {
		count uint32
		value DBValue
	})
	if lastCanonicalized != nil {
		block := lastCanonicalized.Block
		hash := lastCanonicalized.Hash
		log.Printf("TRACE: Reading uncanonicalized journal. Last canonicalized %v (%v)", block, hash)
		var total uint64
		block += 1
		for {
			level := newOverlayLevel[BlockHash, Key]()
			for index := uint64(0); index <= maxBlocksPerLevel; index++ {
				journalKey := toJournalKey(block, index)
				record, err := db.GetMeta(journalKey)
				if err != nil {
					return nonCanonicalOverlay[BlockHash, Key]{}, err
				}
				if record != nil {
					recordBytes := *record
					var record journalRecord[BlockHash, Key]
					err := scale.Unmarshal(recordBytes, &record)
					if err != nil {
						return nonCanonicalOverlay[BlockHash, Key]{}, err
					}
					var inserted []Key
					for _, kv := range record.Inserted {
						inserted = append(inserted, kv.Hash)
					}
					overlay := blockOverlay[BlockHash, Key]{
						hash:         record.Hash,
						journalIndex: index,
						journalKey:   journalKey,
						inserted:     inserted,
						deleted:      record.Deleted,
					}
					insertValues(values, record.Inserted)
					log.Printf("TRACE: Uncanonicalized journal entry %v.%v (%v) (%v inserted, %v deleted)\n",
						block,
						index,
						record.Hash,
						len(overlay.inserted),
						len(overlay.deleted),
					)
					level.push(overlay)
					parents[record.Hash] = record.ParentHash
					total += 1
				}
			}
			if len(level.blocks) == 0 {
				break
			}
			levels.PushBack(level)
			block += 1
		}
		log.Printf("TRACE: Finished reading uncanonicalized journal, %v entries\n", total)
	}
	return nonCanonicalOverlay[BlockHash, Key]{
		lastCanonicalized: lastCanonicalized,
		levels:            levels,
		parents:           parents,
		values:            values,
		pinned:            make(map[BlockHash]uint32),
		pinnedInsertions: make(map[BlockHash]struct {
			keys  []Key
			count uint32
		}),
	}, nil
}

// Insert a new block into the overlay. If inserted on the second level or lover expects parent
// to be present in the window.
func (nco *nonCanonicalOverlay[BlockHash, Key]) Insert(
	hash BlockHash,
	number uint64,
	parentHash BlockHash,
	changeset ChangeSet[Key],
) (CommitSet[Key], error) {
	var commit CommitSet[Key]
	frontBlockNumber := nco.frontBlockNumber()
	if nco.levels.Len() == 0 && nco.lastCanonicalized == nil && number > 0 {
		// assume that parent was canonicalized
		lastCanonicalized := hashBlock[BlockHash]{parentHash, number - 1}
		commit.Meta.Inserted = append(commit.Meta.Inserted, HashDBValue[[]byte]{
			Hash:    toMetaKey(lastCanonical, struct{}{}),
			DBValue: scale.MustMarshal(lastCanonicalized),
		})
		nco.lastCanonicalized = &lastCanonicalized
	} else if nco.lastCanonicalized != nil {
		if number < frontBlockNumber || number > frontBlockNumber+uint64(nco.levels.Len()) {
			log.Printf(
				"TRACE: Failed to insert block %v, current is %v .. %v)\n",
				number, frontBlockNumber, frontBlockNumber+uint64(nco.levels.Len()))
			return CommitSet[Key]{}, ErrInvalidBlockNumber
		}
		// check for valid parent if inserting on second level or higher
		if number == frontBlockNumber {
			if !(nco.lastCanonicalized.Hash == parentHash && nco.lastCanonicalized.Block == number-1) {
				return CommitSet[Key]{}, ErrInvalidParent
			}
		} else if _, ok := nco.parents[parentHash]; !ok {
			return CommitSet[Key]{}, ErrInvalidParent
		}
	}
	var level overlayLevel[BlockHash, Key] = newOverlayLevel[BlockHash, Key]()
	var levelIndex int
	if nco.levels.Len() == 0 || number == frontBlockNumber+uint64(nco.levels.Len()) {
		nco.levels.PushBack(newOverlayLevel[BlockHash, Key]())
		level = nco.levels.Back()
		levelIndex = nco.levels.Len() - 1
	} else {
		level = nco.levels.At(int(number - frontBlockNumber))
		levelIndex = int(number - frontBlockNumber)
	}

	if len(level.blocks) >= int(maxBlocksPerLevel) {
		hashes := make([]BlockHash, 0)
		for _, block := range level.blocks {
			hashes = append(hashes, block.hash)
		}
		log.Printf(
			"TRACE: Too many sibling blocks at %v: %v\n",
			number, hashes)
		return CommitSet[Key]{}, fmt.Errorf("too many sibling blocks at %d inserted", number)
	}
	for _, block := range level.blocks {
		if block.hash == hash {
			return CommitSet[Key]{}, ErrBlockAlreadyExists
		}
	}

	index := level.availableIndex()
	journalKey := toJournalKey(number, index)

	var inserted []Key
	for _, kv := range changeset.Inserted {
		inserted = append(inserted, kv.Hash)
	}
	overlay := blockOverlay[BlockHash, Key]{
		hash:         hash,
		journalIndex: index,
		journalKey:   journalKey,
		inserted:     inserted,
		deleted:      changeset.Deleted,
	}
	level.push(overlay)
	// update level after modifying
	nco.levels.Set(levelIndex, level)
	nco.parents[hash] = parentHash
	journalRecord := journalRecord[BlockHash, Key]{
		Hash:       hash,
		ParentHash: parentHash,
		Inserted:   changeset.Inserted,
		Deleted:    changeset.Deleted,
	}
	commit.Meta.Inserted = append(commit.Meta.Inserted, HashDBValue[[]byte]{journalKey, scale.MustMarshal(journalRecord)})
	log.Printf("TRACE: Inserted uncanonicalized changeset %v.%v %v (%v inserted, %v deleted)\n",
		number, index, hash, len(journalRecord.Inserted), len(journalRecord.Deleted))
	insertValues(nco.values, journalRecord.Inserted)
	return commit, nil
}

func (nco *nonCanonicalOverlay[BlockHash, Key]) discardJournals(
	levelIndex uint, discardedJournals *[][]byte, hash BlockHash) {
	if levelIndex >= uint(nco.levels.Len()) {
		return
	}
	level := nco.levels.At(int(levelIndex))
	for _, overlay := range level.blocks {
		parent, ok := nco.parents[overlay.hash]
		if !ok {
			panic("there is a parent entry for each entry in levels; qed")
		}
		if parent == hash {
			*discardedJournals = append(*discardedJournals, overlay.journalKey)
			nco.discardJournals(levelIndex+1, discardedJournals, overlay.hash)
		}
	}
}

func (nco *nonCanonicalOverlay[BlockHash, Key]) frontBlockNumber() uint64 {
	if nco.lastCanonicalized != nil {
		return nco.lastCanonicalized.Block + 1
	} else {
		return 0
	}
}

func (nco *nonCanonicalOverlay[BlockHash, Key]) LastCanonicalizedBlockNumber() *uint64 {
	if nco.lastCanonicalized != nil {
		return &nco.lastCanonicalized.Block
	}
	return nil
}

// Sync will confirm that all changes made to commit sets are on disk. Allows for temporarily pinned
// blocks to be released.
func (nco *nonCanonicalOverlay[BlockHash, Key]) Sync() {
	pinned := nco.pinnedCanonicalized
	nco.pinnedCanonicalized = nil
	for _, hash := range pinned {
		nco.Unpin(hash)
	}
	pinned = nil
	// Reuse the same memory buffer
	nco.pinnedCanonicalized = pinned
}

// Canonicalize will select a top-level root and canonicalized it. Discards all sibling subtrees and the root.
// Add a set of changes of the canonicalized block to a provided `CommitSet`
// Return the block number of the canonicalized block
func (nco *nonCanonicalOverlay[BlockHash, Key]) Canonicalize(
	hash BlockHash,
	commit *CommitSet[Key],
) (uint64, error) {
	log.Printf("TRACE: Canonicalizing %v\n", hash)
	if nco.levels.Len() == 0 {
		return 0, ErrInvalidBlock
	}
	level := nco.levels.PopFront()
	var index = -1
	for i, overlay := range level.blocks {
		if overlay.hash == hash {
			index = i
		}
	}
	if index == -1 {
		return 0, ErrInvalidBlock
	}

	// No failures are possible beyond this point.

	// Force pin canonicalized block so that it is not discarded immediately
	nco.Pin(hash)
	nco.pinnedCanonicalized = append(nco.pinnedCanonicalized, hash)

	var discardedJournals [][]byte
	levelBlocks := level.blocks
	level.blocks = nil
	for i, overlay := range levelBlocks {
		var pinnedChildren uint32
		// That's the one we need to canonicalize
		if i == index {
			for _, k := range overlay.inserted {
				cv, ok := nco.values[k]
				if !ok {
					panic("For each key in overlays there's a value in values")
				}
				commit.Data.Inserted = append(commit.Data.Inserted, HashDBValue[Key]{k, cv.value})
			}
			commit.Data.Deleted = append(commit.Data.Deleted, overlay.deleted...)
		} else {
			// Discard this overlay
			nco.discardJournals(0, &discardedJournals, overlay.hash)
			var levels splitLevels[BlockHash, Key] = newSplitLevels[BlockHash, Key]()
			for i := 0; i < nco.levels.Len(); i++ {
				level := nco.levels.At(i)
				if i < nco.levels.Len()/2 {
					*levels[0] = append(*levels[0], level)
				} else {
					*levels[1] = append(*levels[1], level)
				}
			}
			pinnedChildren = discardDescendants(levels, nco.values, nco.parents, nco.pinned, nco.pinnedInsertions, overlay.hash)
			for i := 0; i < (len(*levels[0]) + len(*levels[1])); i++ {
				if i < len(*levels[0]) {
					nco.levels.Set(i, (*levels[0])[i])
				} else {
					nco.levels.Set(i, (*levels[1])[i-len(*levels[0])])
				}
			}
		}
		if _, ok := nco.pinned[overlay.hash]; ok {
			pinnedChildren += 1
		}
		if pinnedChildren != 0 {
			keys := make([]Key, len(overlay.inserted))
			_ = copy(keys, overlay.inserted)
			nco.pinnedInsertions[overlay.hash] = struct {
				keys  []Key
				count uint32
			}{
				keys, pinnedChildren,
			}
		} else {
			delete(nco.parents, overlay.hash)
			discardValues(nco.values, overlay.inserted)
		}
		discardedJournals = append(discardedJournals, overlay.journalKey)
	}
	commit.Meta.Deleted = append(commit.Meta.Deleted, discardedJournals...)

	canonicalized := hashBlock[BlockHash]{hash, nco.frontBlockNumber()}
	commit.Meta.Inserted = append(commit.Meta.Inserted, HashDBValue[[]byte]{
		Hash:    toMetaKey(lastCanonical, struct{}{}),
		DBValue: scale.MustMarshal(canonicalized),
	})
	log.Printf("TRACE: Discarding %v records\n", len(commit.Meta.Deleted))

	num := canonicalized.Block
	nco.lastCanonicalized = &canonicalized
	return num, nil
}

// Get a value from the node overlay. This searches in every existing changeset.
func (nco *nonCanonicalOverlay[BlockHash, Key]) Get(key Key) *DBValue {
	cv, ok := nco.values[key]
	if !ok {
		return nil
	}
	return &cv.value
}

// HaveBlock checks if the block is in the canonicalization queue.
func (nco *nonCanonicalOverlay[BlockHash, Key]) HaveBlock(hash BlockHash) bool {
	_, ok := nco.parents[hash]
	return ok
}

// RevertOne will revert a single level. Returns commit set that deletes the journal or `nil` if not
// possible.
func (nco *nonCanonicalOverlay[BlockHash, Key]) RevertOne() *CommitSet[Key] {
	if nco.levels.Len() == 0 {
		return nil
	}
	level := nco.levels.PopBack()
	commit := CommitSet[Key]{}
	for _, overlay := range level.blocks {
		commit.Meta.Deleted = append(commit.Meta.Deleted, overlay.journalKey)
		delete(nco.parents, overlay.hash)
		discardValues(nco.values, overlay.inserted)
	}
	return &commit
}

// Remove will revert a single block. Returns commit set that deletes the journal or `nil` if not
// possible.
func (nco *nonCanonicalOverlay[BlockHash, Key]) Remove(hash BlockHash) *CommitSet[Key] {
	commit := CommitSet[Key]{}
	levelCount := nco.levels.Len()
	for levelIndex := nco.levels.Len() - 1; levelIndex >= 0; levelIndex-- {
		level := nco.levels.At(levelIndex)
		var index int = -1
		for i, overlay := range level.blocks {
			if overlay.hash == hash {
				index = i
				break
			}
		}
		if index == -1 {
			continue
		}
		// Check that it does not have any children
		if levelIndex != levelCount-1 {
			for _, h := range nco.parents {
				if h == hash {
					log.Printf("DEBUG: Trying to remove block %v with children\n", hash)
					return nil
				}
			}
		}
		overlay := level.remove(uint(index))
		nco.levels.Set(levelIndex, level)
		commit.Meta.Deleted = append(commit.Meta.Deleted, overlay.journalKey)
		delete(nco.parents, overlay.hash)
		discardValues(nco.values, overlay.inserted)
		break
	}
	if nco.levels.Len() > 0 && len(nco.levels.Back().blocks) == 0 {
		nco.levels.PopBack()
	}
	if len(commit.Meta.Deleted) > 0 {
		return &commit
	}
	return nil
}

// Pin state values in memory
func (nco *nonCanonicalOverlay[BlockHash, Key]) Pin(hash BlockHash) {
	refs := nco.pinned[hash]
	if refs == 0 {
		log.Println("TRACE: Pinned non-canon block:", hash)
	}
	refs += 1
	nco.pinned[hash] = refs
}

// Unpin will discard pinned state
func (nco *nonCanonicalOverlay[BlockHash, Key]) Unpin(hash BlockHash) {
	var removed bool
	entry, ok := nco.pinned[hash]
	if ok {
		entry -= 1
		if entry == 0 {
			delete(nco.pinned, hash)
			removed = true
		} else {
			removed = false
		}
	}

	if removed {
		var parent *BlockHash = &hash
		for parent != nil {
			hash := *parent
			parentHash, ok := nco.parents[hash]
			if !ok {
				parent = nil
			} else {
				parent = &parentHash
			}

			entry, ok := nco.pinnedInsertions[hash]
			if ok {
				entry.count -= 1
				if entry.count == 0 {
					delete(nco.pinnedInsertions, hash)
					log.Println("TRACE: Discarding unpinned non-canon block:", hash)
					discardValues(nco.values, entry.keys)
					delete(nco.parents, hash)
				}
			} else {
				break
			}
		}
	}
}

type overlayLevel[BlockHash Hash, Key Hash] struct {
	blocks      []blockOverlay[BlockHash, Key]
	usedIndices uint64
}

func (ol *overlayLevel[BlockHash, Key]) push(overlay blockOverlay[BlockHash, Key]) {
	ol.usedIndices = ol.usedIndices | 1<<overlay.journalIndex
	ol.blocks = append(ol.blocks, overlay)
}

func (ol *overlayLevel[BlockHash, Key]) availableIndex() uint64 {
	return uint64(bits.TrailingZeros64(^ol.usedIndices))
}

func (ol *overlayLevel[BlockHash, Key]) remove(index uint) blockOverlay[BlockHash, Key] {
	ol.usedIndices = ol.usedIndices & ^(1 << ol.blocks[index].journalIndex)
	b := ol.blocks[index]
	ol.blocks = append(ol.blocks[:index], ol.blocks[index+1:]...)
	return b
}

func newOverlayLevel[BlockHash Hash, Key Hash]() overlayLevel[BlockHash, Key] {
	return overlayLevel[BlockHash, Key]{}
}

type journalRecord[BlockHash Hash, Key Hash] struct {
	Hash       BlockHash
	ParentHash BlockHash
	Inserted   []HashDBValue[Key]
	Deleted    []Key
}

func toJournalKey(block uint64, index uint64) []byte {
	type blockIndex struct {
		Block uint64
		Index uint64
	}
	return toMetaKey([]byte("noncanonical_journal"), blockIndex{block, index})
}

type blockOverlay[BlockHash Hash, Key Hash] struct {
	hash         BlockHash
	journalIndex uint64
	journalKey   []byte
	inserted     []Key
	deleted      []Key
}

func insertValues[Key Hash](values map[Key]struct {
	count uint32
	value DBValue
}, inserted []HashDBValue[Key]) {
	for _, kv := range inserted {
		cv, ok := values[kv.Hash]
		if !ok {
			values[kv.Hash] = struct {
				count uint32
				value DBValue
			}{0, kv.DBValue}
			cv = values[kv.Hash]
		}
		cv.count += 1
		values[kv.Hash] = cv
	}
}

func discardValues[Key Hash](values map[Key]struct {
	count uint32
	value DBValue
}, inserted []Key) {
	for _, k := range inserted {
		cv, ok := values[k]
		if !ok {
			panic("trying to discard missing value")
		}
		cv.count--
		if cv.count == 0 {
			delete(values, k)
		} else {
			values[k] = cv
		}
	}
}

func splitFirst[T any](levels *[]T) (
	first *T,
	remainder *[]T,
	err error,
) {
	if len(*levels) == 0 {
		return nil, nil, fmt.Errorf("zero length level")
	}
	first = &(*levels)[0]
	if len(*levels) >= 1 {
		remainderSlice := (*levels)[1:]
		remainder = &remainderSlice
	}
	return
}

type splitLevels[BlockHash Hash, Key Hash] [2]*[]overlayLevel[BlockHash, Key]

func newSplitLevels[BlockHash Hash, Key Hash]() splitLevels[BlockHash, Key] {
	first := make([]overlayLevel[BlockHash, Key], 0)
	last := make([]overlayLevel[BlockHash, Key], 0)
	return splitLevels[BlockHash, Key]{
		&first, &last,
	}
}

func discardDescendants[BlockHash Hash, Key Hash](
	levels splitLevels[BlockHash, Key],
	values map[Key]struct {
		count uint32
		value DBValue
	},
	parents map[BlockHash]BlockHash,
	pinned map[BlockHash]uint32,
	pinnedInsertions map[BlockHash]struct {
		keys  []Key
		count uint32
	},
	hash BlockHash,
) uint32 {
	var firstLevel *overlayLevel[BlockHash, Key]
	var remainder splitLevels[BlockHash, Key]

	first, rest, err := splitFirst(levels[0])
	if err == nil {
		firstLevel = first
		remainder = splitLevels[BlockHash, Key]{rest, levels[1]}
	} else {
		first, rest, err := splitFirst(levels[1])
		if err == nil {
			firstLevel = first
			remainder = splitLevels[BlockHash, Key]{levels[0], rest}
		} else {
			firstLevel = nil
			remainder = splitLevels[BlockHash, Key]{levels[0], levels[1]}
		}
	}

	var pinnedChildren uint32
	if firstLevel != nil {
		level := firstLevel
		var index uint
	main:
		for {
			for i, overlay := range level.blocks {
				h, ok := parents[overlay.hash]
				if !ok {
					panic("there is a parent entry for each entry in levels; qed")
				}
				if h == hash {
					index = uint(i)
					overlay := level.remove(index)
					numPinned := discardDescendants(remainder, values, parents, pinned, pinnedInsertions, overlay.hash)
					if _, ok := pinned[overlay.hash]; ok {
						numPinned += 1
					}
					if numPinned != 0 {
						// save to be discarded later.
						pinnedInsertions[overlay.hash] = struct {
							keys  []Key
							count uint32
						}{overlay.inserted, numPinned}
						pinnedChildren += numPinned
					} else {
						// discard immediately.
						delete(parents, overlay.hash)
						discardValues(values, overlay.inserted)
					}
					continue main
				}
			}
			break main
		}

	}
	return pinnedChildren
}
