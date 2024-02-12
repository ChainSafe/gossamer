package statedb

//! State database maintenance. Handles canonicalization and pruning in the database.
//!
//! # Canonicalization.
//! Canonicalization window tracks a tree of blocks identified by header hash. The in-memory
//! overlay allows to get any trie node that was inserted in any of the blocks within the window.
//! The overlay is journaled to the backing database and rebuilt on startup.
//! There's a limit of 32 blocks that may have the same block number in the canonicalization window.
//!
//! Canonicalization function selects one root from the top of the tree and discards all other roots
//! and their subtrees. Upon canonicalization all trie nodes that were inserted in the block are
//! added to the backing DB and block tracking is moved to the pruning window, where no forks are
//! allowed.
//!
//! # Canonicalization vs Finality
//! Database engine uses a notion of canonicality, rather then finality. A canonical block may not
//! be yet finalized from the perspective of the consensus engine, but it still can't be reverted in
//! the database. Most of the time during normal operation last canonical block is the same as last
//! finalized. However if finality stall for a long duration for some reason, there's only a certain
//! number of blocks that can fit in the non-canonical overlay, so canonicalization of an
//! unfinalized block may be forced.
//!
//! # Pruning.
//! See `RefWindow` for pruning algorithm details. `StateDb` prunes on each canonicalization until
//! pruning constraints are satisfied.

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

const defaultMaxBlockConstraint uint32 = 256

var (
	pruningMode             = []byte("mode")
	pruningModeArchive      = []byte("archive")
	pruningModeArchiveCanon = []byte("archive_canonical")
	pruningModeConstrained  = []byte("constrained")
)

// / Database value type.
type DBValue []byte

// / Basic set of requirements for the Block hash and node key types.
type Hash interface {
	comparable
}

type HashDBValue[H any] struct {
	Hash H
	DBValue
}

// / Backend database trait. Read-only.
type MetaDB interface {
	/// Get meta value, such as the journal.
	GetMeta(key []byte) (*DBValue, error)
}

// / Backend database trait. Read-only.
type NodeDB[Key comparable] interface {
	/// Get state trie node.
	Get(key Key) (*DBValue, error)
}

var (
	/// Trying to canonicalize invalid block.
	ErrInvalidBlock = errors.New("trying to canonicalize invalid block")
	/// Trying to insert block with invalid number.
	ErrInvalidBlockNumber = errors.New("trying to insert block with invalid number")
	/// Trying to insert block with unknown parent.
	ErrInvalidParent = errors.New("trying to insert block with unknown parent")
	/// Invalid pruning mode specified. Contains expected mode.
	ErrIncompatiblePruningModes = errors.New("incompatible pruning modes")
	/// Trying to insert existing block.
	ErrBlockAlreadyExists = errors.New("block already exists")
	/// Trying to get a block record from db while it is not commit to db yet
	ErrBlockUnavailable = errors.New("trying to get a block record from db while it is not commit to db yet")
	/// Invalid metadata
	ErrMetadata = errors.New("Invalid metadata:")
)

// / A set of state node changes.
type ChangeSet[H any] struct {
	/// Inserted nodes.
	Inserted []HashDBValue[H]
	/// Deleted nodes.
	Deleted []H
}

// / A set of changes to the backing database.
type CommitSet[H Hash] struct {
	/// State node changes.
	Data ChangeSet[H]
	/// Metadata changes.
	Meta ChangeSet[[]byte]
}

// / Pruning constraints. If none are specified pruning is
type Constraints struct {
	/// Maximum blocks. Defaults to 0 when unspecified, effectively keeping only non-canonical
	/// states.
	MaxBlocks *uint32
}

// / Pruning mode.
type PruningMode interface {
	IsArchive() bool
	ID() []byte
}
type PruningModes interface {
	PruningModeConstrained | PruningModeArchiveAll | PruningModeArchiveCanonical
}

func NewPruningModeFromID(id []byte) PruningMode {
	switch string(id) {
	case string(pruningModeArchive):
		return PruningModeArchiveAll{}
	case string(pruningModeArchiveCanon):
		return PruningModeArchiveCanonical{}
	case string(pruningModeConstrained):
		defaultBlocks := defaultMaxBlockConstraint
		return PruningModeConstrained{MaxBlocks: &defaultBlocks}
	default:
		return nil
	}
}

// / Maintain a pruning window.
type PruningModeConstrained Constraints

func (pmc PruningModeConstrained) IsArchive() bool {
	return false
}
func (pmc PruningModeConstrained) ID() []byte {
	return []byte("constrained")
}

// / No pruning. Canonicalization is a no-op.
type PruningModeArchiveAll struct{}

func (pmaa PruningModeArchiveAll) IsArchive() bool {
	return true
}
func (pmaa PruningModeArchiveAll) ID() []byte {
	return []byte("archive")
}

// / Canonicalization discards non-canonical nodes. All the canonical nodes are kept in the DB.
type PruningModeArchiveCanonical struct{}

func (pmac PruningModeArchiveCanonical) IsArchive() bool {
	return true
}
func (pmac PruningModeArchiveCanonical) ID() []byte {
	return []byte("archive_canonical")
}

func toMetaKey(suffix []byte, data any) []byte {
	key := scale.MustMarshal(data)
	key = append(key, suffix...)
	return key
}

// / Status information about the last canonicalized block.
type LastCanonicalized any
type LastCanonicalizedValues interface {
	LastCanonicalizedNone | LastCanonicalizedBlock | LastCanonicalizedNotCanonicalizing
}

// / Not yet have canonicalized any block.
type LastCanonicalizedNone struct{}

// / The given block number is the last canonicalized block.
type LastCanonicalizedBlock uint64

// / No canonicalization is happening (pruning mode is archive all).
type LastCanonicalizedNotCanonicalizing struct{}

type StateDBSync[BlockHash Hash, Key Hash] struct {
	mode         PruningMode
	nonCanonical NonCanonicalOverlay[BlockHash, Key]
	pruning      *refWindow[BlockHash, Key]
	pinned       map[BlockHash]uint32
	// refCounting  bool
}

func NewStateDBSync[BlockHash Hash, Key Hash](
	mode PruningMode,
	// refCounting bool,
	db MetaDB,
) (StateDBSync[BlockHash, Key], error) {
	nonCanonical, err := NewNonCanonicalOverlay[BlockHash, Key](db)
	if err != nil {
		return StateDBSync[BlockHash, Key]{}, err
	}
	var pruning *refWindow[BlockHash, Key]
	switch mode := mode.(type) {
	case PruningModeConstrained:
		rw, err := newRefWindow[BlockHash, Key](db, *mode.MaxBlocks)
		if err != nil {
			return StateDBSync[BlockHash, Key]{}, err
		}
		pruning = &rw
	}
	return StateDBSync[BlockHash, Key]{
		mode:         mode,
		nonCanonical: nonCanonical,
		pruning:      pruning,
		pinned:       make(map[BlockHash]uint32),
	}, nil
}

func (sdbs *StateDBSync[BlockHash, Key]) insertBlock(
	hash BlockHash,
	number uint64,
	parentHash BlockHash,
	changeset ChangeSet[Key],
) (CommitSet[Key], error) {
	switch sdbs.mode.(type) {
	case PruningModeArchiveAll:
		changeset.Deleted = nil
		return CommitSet[Key]{
			Data: changeset,
		}, nil
	default:
		return sdbs.nonCanonical.Insert(hash, number, parentHash, changeset)
	}
}

func (sdbs *StateDBSync[BlockHash, Key]) canonicalizeBlock(hash BlockHash) (CommitSet[Key], error) {
	// NOTE: it is important that the change to `LAST_CANONICAL` (emit from
	// `non_canonical.canonicalize`) and the insert of the new pruning journal (emit from
	// `pruning.note_canonical`) are collected into the same `CommitSet` and are committed to
	// the database atomically to keep their consistency when restarting the node
	commit := CommitSet[Key]{}
	if _, ok := sdbs.mode.(PruningModeArchiveAll); ok {
		return commit, nil
	}
	number, err := sdbs.nonCanonical.Canonicalize(hash, &commit)
	if err != nil {
		return CommitSet[Key]{}, err
	}
	if _, ok := sdbs.mode.(PruningModeArchiveCanonical); ok {
		commit.Data.Deleted = nil
	}
	if sdbs.pruning != nil {
		err := sdbs.pruning.NoteCanonical(hash, number, &commit)
		if err != nil {
			return CommitSet[Key]{}, err
		}
	}
	err = sdbs.prune(&commit)
	if err != nil {
		return CommitSet[Key]{}, err
	}
	return commit, nil
}

// / Returns the block number of the last canonicalized block.
func (sdbs *StateDBSync[BlockHash, Key]) lastCanonicalized() LastCanonicalized {
	switch sdbs.mode.(type) {
	case PruningModeArchiveAll:
		return LastCanonicalizedNotCanonicalizing{}
	default:
		num := sdbs.nonCanonical.LastCanonicalizedBlockNumber()
		if num == nil {
			return LastCanonicalizedNone{}
		}
		return LastCanonicalizedBlock(*num)
	}
}

func (sdbs *StateDBSync[BlockHash, Key]) isPruned(hash BlockHash, number uint64) IsPruned {
	switch sdbs.mode.(type) {
	case PruningModeArchiveAll:
		return IsPrunedNotPruned
	case PruningModeArchiveCanonical, PruningModeConstrained:
		var cond bool
		num := sdbs.nonCanonical.LastCanonicalizedBlockNumber()
		if num != nil {
			cond = number > *num
		} else {
			cond = true
		}
		if cond {
			if sdbs.nonCanonical.HaveBlock(hash) {
				return IsPrunedNotPruned
			}
			return IsPrunedPruned
		}

		if sdbs.pruning != nil {
			switch sdbs.pruning.HaveBlock(hash, number) {
			case haveBlockNo:
				return IsPrunedPruned
			case haveBlockYes:
				return IsPrunedNotPruned
			case haveBlockMaybe:
				return IsPrunedMaybePruned
			}
		}
		// We don't know for sure.
		return IsPrunedMaybePruned
	default:
		panic("wtf?")
	}
}

func (sdbs *StateDBSync[BlockHash, Key]) prune(commit *CommitSet[Key]) error {
	if constraints, ok := sdbs.mode.(PruningModeConstrained); ok {
		for {
			var maxBlocks uint64
			if constraints.MaxBlocks != nil {
				maxBlocks = uint64(*constraints.MaxBlocks)
			}
			if sdbs.pruning.WindowSize() <= maxBlocks {
				break
			}

			hash, err := sdbs.pruning.NextHash()
			if err != nil {
				if errors.Is(err, ErrBlockUnavailable) {
					// the block record is temporary unavailable, break and try next time
					break
				}
				return err
			}
			if hash != nil {
				_, ok := sdbs.pinned[*hash]
				if ok {
					break
				}
			}
			err = sdbs.pruning.PruneOne(commit)
			// this branch should not reach as previous `next_hash` don't return error
			// keeping it for robustness
			if err != nil {
				if errors.Is(err, ErrBlockUnavailable) {
					break
				}
				return err
			}
		}
	}
	return nil
}

// / Revert all non-canonical blocks with the best block number.
// / Returns a database commit or `None` if not possible.
// / For archive an empty commit set is returned.
func (sdbs *StateDBSync[BlockHash, Key]) revertOne() *CommitSet[Key] {
	switch sdbs.mode.(type) {
	case PruningModeArchiveAll:
		return &CommitSet[Key]{}
	case PruningModeArchiveCanonical, PruningModeConstrained:
		return sdbs.nonCanonical.RevertOne()
	default:
		panic("wtf?")
	}
}

func (sdbs *StateDBSync[BlockHash, Key]) remove(hash BlockHash) *CommitSet[Key] {
	switch sdbs.mode.(type) {
	case PruningModeArchiveAll:
		return &CommitSet[Key]{}
	case PruningModeArchiveCanonical, PruningModeConstrained:
		return sdbs.nonCanonical.Remove(hash)
	default:
		panic("wtf?")
	}
}

func (sdbs *StateDBSync[BlockHash, Key]) pin(hash BlockHash, number uint64, hint func() bool) error {
	switch sdbs.mode.(type) {
	case PruningModeArchiveAll:
		return nil
	case PruningModeArchiveCanonical, PruningModeConstrained:
		var haveBlock bool
		left := sdbs.nonCanonical.HaveBlock(hash)
		var right bool
		if sdbs.pruning != nil {
			hb := sdbs.pruning.HaveBlock(hash, number)
			switch hb {
			case haveBlockNo:
				right = false
			case haveBlockYes:
				right = true
			case haveBlockMaybe:
				right = hint()
			}
		} else {
			right = hint()
		}
		haveBlock = left || right
		if haveBlock {
			refs := sdbs.pinned[hash]
			if refs == 0 {
				log.Println("TRACE: Pinned block:", hash)
				sdbs.nonCanonical.Pin(hash)
			}
			sdbs.pinned[hash] += 1
			return nil
		}
		return ErrInvalidBlock
	default:
		panic("wtf?")
	}
}

func (sdbs *StateDBSync[BlockHash, Key]) unpin(hash BlockHash) {
	entry, ok := sdbs.pinned[hash]
	if ok {
		sdbs.pinned[hash] -= 1
		if entry == 0 {
			log.Println("TRACE: Unpinned block:", hash)
			delete(sdbs.pinned, hash)
			sdbs.nonCanonical.Unpin(hash)
		} else {
			log.Println("TRACE: Releasing reference for ", hash)
		}
	}
}

func (sdbs *StateDBSync[BlockHash, Key]) sync() {
	sdbs.nonCanonical.Sync()
}

func (sdbs *StateDBSync[BlockHash, Key]) get(key Key, db NodeDB[Key]) (*DBValue, error) {
	val := sdbs.nonCanonical.Get(key)
	if val != nil {
		return val, nil
	}
	return db.Get(key)
}

// / State DB maintenance. See module description.
// / Can be shared across threads.
type StateDB[BlockHash Hash, Key Hash] struct {
	db StateDBSync[BlockHash, Key]
	sync.RWMutex
}

func NewStateDB[BlockHash Hash, Key Hash](
	db MetaDB,
	requestedMode PruningMode,
	// refCounting bool,
	shouldInit bool,
) (CommitSet[Key], StateDB[BlockHash, Key], error) {
	storedMode, err := fetchStoredPruningMode(db)
	if err != nil {
		return CommitSet[Key]{}, StateDB[BlockHash, Key]{}, err
	}

	var selectedMode PruningMode
	switch {
	case shouldInit:
		if storedMode != nil {
			panic("The storage has just been initialized. No meta-data is expected to be found in it.")
		}
		if requestedMode != nil {
			selectedMode = requestedMode
		} else {
			maxBlocks := defaultMaxBlockConstraint
			selectedMode = PruningModeConstrained{
				MaxBlocks: &maxBlocks,
			}
		}
	case !shouldInit && storedMode == nil:
		return CommitSet[Key]{}, StateDB[BlockHash, Key]{}, fmt.Errorf("%w An existing StateDb does not have PRUNING_MODE stored in its meta-data", ErrMetadata)
	case !shouldInit && storedMode != nil && requestedMode == nil:
		selectedMode = storedMode
	case !shouldInit && storedMode != nil && requestedMode != nil:
		mode, err := choosePruningMode(storedMode, requestedMode)
		if err != nil {
			return CommitSet[Key]{}, StateDB[BlockHash, Key]{}, err
		}
		selectedMode = mode
	default:
		panic("wtf?")
	}

	var dbInitCommitSet CommitSet[Key]
	if shouldInit {
		var cs CommitSet[Key]

		key := toMetaKey(pruningMode, struct{}{})
		value := selectedMode.ID()

		cs.Meta.Inserted = append(cs.Meta.Inserted, HashDBValue[[]byte]{
			Hash:    key,
			DBValue: value,
		})
		dbInitCommitSet = cs
	}

	stateDBSync, err := NewStateDBSync[BlockHash, Key](selectedMode, db)
	if err != nil {
		return CommitSet[Key]{}, StateDB[BlockHash, Key]{}, err
	}
	stateDB := StateDB[BlockHash, Key]{
		db: stateDBSync,
	}
	return dbInitCommitSet, stateDB, nil
}

func (sdb *StateDB[BlockHash, Key]) PruningMode() PruningMode {
	sdb.RLock()
	defer sdb.RUnlock()
	return sdb.db.mode
}

// / Add a new non-canonical block.
func (sdb *StateDB[BlockHash, Key]) InsertBlock(
	hash BlockHash,
	number uint64,
	parentHash BlockHash,
	changeset ChangeSet[Key],
) (CommitSet[Key], error) {
	sdb.Lock()
	defer sdb.Unlock()
	return sdb.db.insertBlock(hash, number, parentHash, changeset)
}

// / Finalize a previously inserted block.
func (sdb *StateDB[BlockHash, Key]) CanonicalizeBlock(hash BlockHash) (CommitSet[Key], error) {
	sdb.Lock()
	defer sdb.Unlock()
	return sdb.db.canonicalizeBlock(hash)
}

// / Prevents pruning of specified block and its descendants.
// / `hint` used for further checking if the given block exists
func (sdb *StateDB[BlockHash, Key]) Pin(hash BlockHash, number uint64, hint func() bool) error {
	sdb.Lock()
	defer sdb.Unlock()
	return sdb.db.pin(hash, number, hint)
}

// / Allows pruning of specified block.
func (sdb *StateDB[BlockHash, Key]) Unpin(hash BlockHash) {
	sdb.Lock()
	defer sdb.Unlock()
	sdb.db.unpin(hash)
}

// / Confirm that all changes made to commit sets are on disk. Allows for temporarily pinned
// / blocks to be released.
func (sdb *StateDB[BlockHash, Key]) Sync() {
	sdb.Lock()
	defer sdb.Unlock()
	sdb.db.sync()
}

// / Get a value from non-canonical/pruning overlay or the backing DB.
func (sdb *StateDB[BlockHash, Key]) Get(key Key, db NodeDB[Key]) (*DBValue, error) {
	sdb.RLock()
	defer sdb.RUnlock()
	return sdb.db.get(key, db)
}

// / Revert all non-canonical blocks with the best block number.
// / Returns a database commit or `None` if not possible.
// / For archive an empty commit set is returned.
func (sdb *StateDB[BlockHash, Key]) RevertOne() *CommitSet[Key] {
	sdb.Lock()
	defer sdb.Unlock()
	return sdb.db.revertOne()
}

// / Remove specified non-canonical block.
// / Returns a database commit or `None` if not possible.
func (sdb *StateDB[BlockHash, Key]) Remove(hash BlockHash) *CommitSet[Key] {
	sdb.Lock()
	defer sdb.Unlock()
	return sdb.db.remove(hash)
}

// / Returns last canonicalized block.
func (sdb *StateDB[BlockHash, Key]) LastCanonicalized() LastCanonicalized {
	sdb.RLock()
	defer sdb.RUnlock()
	return sdb.db.lastCanonicalized()
}

// / Check if block is pruned away.
func (sdb *StateDB[BlockHash, Key]) IsPruned(hash BlockHash, number uint64) IsPruned {
	sdb.RLock()
	defer sdb.RUnlock()
	return sdb.db.isPruned(hash, number)
}

// / Reset in-memory changes to the last disk-backed state.
func (sdb *StateDB[BlockHash, Key]) Reset(db MetaDB) error {
	sdb.Lock()
	defer sdb.Unlock()
	new, err := NewStateDBSync[BlockHash, Key](sdb.db.mode, db)
	if err != nil {
		return err
	}
	sdb.db = new
	return nil
}

// / The result returned by `StateDB.IsPruned()`
type IsPruned uint

const (
	/// Definitely pruned
	IsPrunedPruned IsPruned = iota
	/// Definitely not pruned
	IsPrunedNotPruned
	/// May or may not pruned, need further checking
	IsPrunedMaybePruned
)

func choosePruningMode(stored, requested PruningMode) (PruningMode, error) {
	switch stored.(type) {
	case PruningModeArchiveAll:
		switch requested.(type) {
		case PruningModeArchiveAll:
			return PruningModeArchiveAll{}, nil
		default:
			return nil, fmt.Errorf("%w [stored: %T; requested: %T]",
				ErrIncompatiblePruningModes, stored, requested)
		}
	case PruningModeArchiveCanonical:
		switch requested.(type) {
		case PruningModeArchiveCanonical:
			return PruningModeArchiveCanonical{}, nil
		default:
			return nil, fmt.Errorf("%w [stored: %T; requested: %T]",
				ErrIncompatiblePruningModes, stored, requested)
		}
	case PruningModeConstrained:
		switch req := requested.(type) {
		case PruningModeConstrained:
			return req, nil
		default:
			return nil, fmt.Errorf("%w [stored: %T; requested: %T]",
				ErrIncompatiblePruningModes, stored, requested)
		}
	default:
		return nil, fmt.Errorf("%w [stored: %T; requested: %T]",
			ErrIncompatiblePruningModes, stored, requested)
	}
}

func fetchStoredPruningMode(db MetaDB) (PruningMode, error) {
	metaKeyNode := toMetaKey(pruningMode, struct{}{})
	val, err := db.GetMeta(metaKeyNode)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	mode := NewPruningModeFromID(*val)
	if mode != nil {
		return mode, nil
	}
	return nil, fmt.Errorf("invalid value stored for PRUNING_MODE: %v", *val)
}
