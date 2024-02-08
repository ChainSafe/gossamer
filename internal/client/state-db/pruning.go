package statedb

import (
	"log"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/gammazero/deque"
)

var lastPruned = []byte("last_pruned")

// ! Pruning window.
// !
// ! For each block we maintain a list of nodes pending deletion.
// ! There is also a global index of node key to block number.
// ! If a node is re-inserted into the window it gets removed from
// ! the death list.
// ! The changes are journaled in the DB.
type refWindow[BlockHash Hash, Key Hash] struct {
	/// A queue of blocks keep tracking keys that should be deleted for each block in the
	/// pruning window.
	queue deathRowQueue[BlockHash, Key]
	/// Block number that is next to be pruned.
	base uint64
}

func newRefWindow[BlockHash Hash, Key Hash](
	db MetaDB, windowSize uint32, countInsertions bool,
) (refWindow[BlockHash, Key], error) {
	// the block number of the first block in the queue or the next block number if the queue is
	// empty
	var base uint64
	val, err := db.GetMeta(toMetaKey(lastPruned, struct{}{}))
	if err != nil {
		return refWindow[BlockHash, Key]{}, err
	}
	if val != nil {
		err = scale.Unmarshal(*val, &base)
		if err != nil {
			return refWindow[BlockHash, Key]{}, err
		}
		base++
	}

	// NOTE: only needed if we have DB backed version of deathRowQueue
	// the block number of the last block in the queue
	// var lastCanonicalizedNumber *uint64
	// val, err = db.GetMeta(toMetaKey(lastCanonical, struct{}{}))
	// if err != nil {
	// 	return refWindow[BlockHash, Key]{}, err
	// }
	// if err != nil {
	// 	buffer := *val
	// 	var dst hashBlock[BlockHash]
	// 	err := scale.Unmarshal(buffer, &dst)
	// 	if err != nil {
	// 		return refWindow[BlockHash, Key]{}, err
	// 	}
	// 	lastCanonicalizedNumber = &dst.Block
	// }

	if !countInsertions {
		panic("currently only support counted insertions in memory")
	}

	if windowSize > 1000 {
		log.Printf(
			"TRACE: Large pruning window of %d detected! THIS CAN LEAD TO HIGH MEMORY USAGE AND CRASHES. Reduce the pruning window.",
			windowSize)
	}

	queue, err := newDeathRowQueueInMem[BlockHash, Key](db, base)
	if err != nil {
		return refWindow[BlockHash, Key]{}, err
	}

	return refWindow[BlockHash, Key]{
		queue: queue,
		base:  base,
	}, nil
}

func (rw *refWindow[BlockHash, Key]) WindowSize() uint64 {
	return rw.queue.Len(rw.base)
}

// / Get the hash of the next pruning block
func (rw *refWindow[BlockHash, Key]) NextHash() (*BlockHash, error) {
	return rw.queue.NextHash()
}

func (rw *refWindow[BlockHash, Key]) isEmpty() bool {
	return rw.WindowSize() == 0
}

// Check if a block is in the pruning window and not be pruned yet
func (rw *refWindow[BlockHash, Key]) HaveBlock(hash BlockHash, number uint64) haveBlock {
	// if the queue is empty or the block number exceed the pruning window, we definitely
	// do not have this block
	if rw.isEmpty() || number < rw.base || number >= rw.base+rw.WindowSize() {
		return haveBlockNo
	}
	return rw.queue.HaveBlock(hash, uint(number-rw.base))
}

// / Prune next block. Expects at least one block in the window. Adds changes to `commit`.
func (rw *refWindow[BlockHash, Key]) PruneOne(commit *CommitSet[Key]) error {
	pruned, err := rw.queue.PopFront(rw.base)
	if err != nil {
		return err
	}
	if pruned != nil {
		log.Printf("TRACE: Pruning %v (%v deleted)", pruned.hash, len(pruned.deleted))
		index := rw.base
		var deletedKeys []Key
		for deleted := range pruned.deleted {
			deletedKeys = append(deletedKeys, deleted)
		}
		commit.Data.Deleted = append(commit.Data.Deleted, deletedKeys...)
		commit.Meta.Inserted = append(commit.Meta.Inserted, HashDBValue[[]byte]{
			Hash:    toMetaKey(lastPruned, struct{}{}),
			DBValue: scale.MustMarshal(index),
		})
		commit.Meta.Deleted = append(commit.Meta.Deleted, toPruningJournalKey(rw.base))
		rw.base += 1
		return nil
	} else {
		log.Printf("TRACE: Trying to prune when there's nothing to prune")
		return ErrBlockUnavailable
	}
}

// / Add a change set to the window. Creates a journal record and pushes it to `commit`
func (rw *refWindow[BlockHash, Key]) NoteCanonical(hash BlockHash, number uint64, commit *CommitSet[Key]) error {
	if rw.base == 0 && rw.isEmpty() && number > 0 {
		// This branch is taken if the node imports the target block of a warp sync.
		// assume that the block was canonicalized
		rw.base = number
		// The parent of the block was the last block that got pruned.
		commit.Meta.Inserted = append(commit.Meta.Inserted, HashDBValue[[]byte]{
			Hash:    toMetaKey(lastPruned, struct{}{}),
			DBValue: scale.MustMarshal(number - 1),
		})
	} else if (rw.base + rw.WindowSize()) != number {
		return ErrInvalidBlockNumber
	}
	log.Printf("TRACE: Adding to pruning window: %v (%v inserted, %v deleted)", hash, len(commit.Data.Inserted), len(commit.Data.Deleted))
	var inserted []Key
	for _, kv := range commit.Data.Inserted {
		inserted = append(inserted, kv.Hash)
	}
	deleted := commit.Data.Deleted
	commit.Data.Deleted = nil
	journalRecord := pruningJournalRecord[BlockHash, Key]{
		hash, inserted, deleted,
	}
	commit.Meta.Inserted = append(commit.Meta.Inserted, HashDBValue[[]byte]{
		Hash:    toPruningJournalKey(number),
		DBValue: scale.MustMarshal(journalRecord),
	})
	rw.queue.Import(rw.base, number, journalRecord)
	return nil
}

type deathRowQueue[BlockHash Hash, Key Hash] interface {
	Import(base uint64, num uint64, journalRecord pruningJournalRecord[BlockHash, Key])
	PopFront(base uint64) (*deathRow[BlockHash, Key], error)
	HaveBlock(hash BlockHash, index uint) haveBlock
	Len(base uint64) uint64
	NextHash() (*BlockHash, error)
}

type deathRowQueueInMem[BlockHash Hash, Key Hash] struct {
	/// A queue of keys that should be deleted for each block in the pruning window.
	deathRows deque.Deque[deathRow[BlockHash, Key]]
	/// An index that maps each key from `death_rows` to block number.
	deathIndex map[Key]uint64
}

func newDeathRowQueueInMem[BlockHash Hash, Key Hash](db MetaDB, base uint64) (deathRowQueue[BlockHash, Key], error) {
	block := base
	queue := &deathRowQueueInMem[BlockHash, Key]{
		deathIndex: make(map[Key]uint64),
	}
	log.Printf("TRACE: Reading pruning journal for the memory queue. Pending #%v\n", base)
	for {
		journalKey := toPruningJournalKey(block)
		val, err := db.GetMeta(journalKey)
		if err != nil {
			return nil, err
		}
		if val != nil {
			var record pruningJournalRecord[BlockHash, Key]
			err := scale.Unmarshal(*val, &record)
			if err != nil {
				return nil, err
			}
			log.Printf(
				"TRACE: Pruning journal entry %v (%v inserted, %v deleted)",
				block, len(record.Inserted), len(record.Deleted))
			queue.Import(base, block, record)
		} else {
			break
		}
		block += 1
	}
	return queue, nil
}

// / import a new block to the back of the queue
func (drqim *deathRowQueueInMem[BlockHash, Key]) Import(
	base uint64, num uint64, journalRecord pruningJournalRecord[BlockHash, Key],
) {
	var (
		hash     = journalRecord.Hash
		inserted = journalRecord.Inserted
		deleted  = journalRecord.Deleted
	)
	log.Printf("TRACE: Importing %v, base=%v\n", num, base)
	// remove all re-inserted keys from death rows
	for _, k := range inserted {
		block, ok := drqim.deathIndex[k]
		if ok {
			delete(drqim.deathIndex, k)
			delete(drqim.deathRows.At(int(block-base)).deleted, k)
		}
	}
	// add new keys
	importedBlock := base + uint64(drqim.deathRows.Len())
	deletedMap := make(map[Key]any)
	for _, k := range deleted {
		drqim.deathIndex[k] = importedBlock
		deletedMap[k] = true
	}
	drqim.deathRows.PushBack(deathRow[BlockHash, Key]{hash, deletedMap})
}

// / Pop out one block from the front of the queue, `base` is the block number
// / of the first block of the queue
func (drqim *deathRowQueueInMem[BlockHash, Key]) PopFront(base uint64) (*deathRow[BlockHash, Key], error) {
	if drqim.deathRows.Len() == 0 {
		return nil, nil
	}
	row := drqim.deathRows.PopFront()
	for k := range row.deleted {
		delete(drqim.deathIndex, k)
	}
	return &row, nil
}

// / Check if the block at the given `index` of the queue exist
// / it is the caller's responsibility to ensure `index` won't be out of bounds
func (drqim *deathRowQueueInMem[BlockHash, Key]) HaveBlock(hash BlockHash, index uint) haveBlock {
	switch drqim.deathRows.At(int(index)).hash == hash {
	case true:
		return haveBlockYes
	default:
		return haveBlockNo
	}
}

// / Return the number of block in the pruning window
func (drqim *deathRowQueueInMem[BlockHash, Key]) Len(base uint64) uint64 {
	return uint64(drqim.deathRows.Len())
}

// / Get the hash of the next pruning block
func (drqim *deathRowQueueInMem[BlockHash, Key]) NextHash() (*BlockHash, error) {
	if drqim.deathRows.Len() == 0 {
		return nil, nil
	}
	row := drqim.deathRows.Front()
	return &row.hash, nil
}

var _ deathRowQueue[hash.H256, hash.H256] = &deathRowQueueInMem[hash.H256, hash.H256]{}

type deathRow[BlockHash Hash, Key Hash] struct {
	hash    BlockHash
	deleted map[Key]any
}

type pruningJournalRecord[BlockHash Hash, Key Hash] struct {
	Hash     BlockHash
	Inserted []Key
	Deleted  []Key
}

func toPruningJournalKey(block uint64) []byte {
	return toMetaKey([]byte("pruning_journal"), block)
}

type haveBlock uint

const (
	/// Definitely don't have this block.
	haveBlockNo haveBlock = iota
	/// May or may not have this block, need further checking
	haveBlockMaybe
	/// Definitely has this block
	haveBlockYes
)
