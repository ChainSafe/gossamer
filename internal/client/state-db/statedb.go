package statedb

import (
	"errors"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

const defaultMaxBlockConstraint uint32 = 256

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

var (
	/// Trying to canonicalize invalid block.
	ErrInvalidBlock = errors.New("trying to canonicalize invalid block")
	/// Trying to insert block with invalid number.
	ErrInvalidBlockNumber = errors.New("trying to insert block with invalid number")
	/// Trying to insert block with unknown parent.
	ErrInvalidParent = errors.New("trying to insert block with unknown parent")
	/// Trying to insert existing block.
	ErrBlockAlreadyExists = errors.New("block already exists")
	/// Trying to get a block record from db while it is not commit to db yet
	ErrBlockUnavailable = errors.New("trying to get a block record from db while it is not commit to db yet")
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
}
type PruningModes interface {
	PruningModeConstrained | PruningModeArchiveAll | PruningModeArchiveCanonical
}

// / Maintain a pruning window.
type PruningModeConstrained Constraints

func (pmc PruningModeConstrained) IsArchive() bool {
	return false
}

// / No pruning. Canonicalization is a no-op.
type PruningModeArchiveAll struct{}

func (pmaa PruningModeArchiveAll) IsArchive() bool {
	return true
}

// / Canonicalization discards non-canonical nodes. All the canonical nodes are kept in the DB.
type PruningModeArchiveCanonical struct{}

func (pmac PruningModeArchiveCanonical) IsArchive() bool {
	return true
}

func toMetaKey(suffix []byte, data any) []byte {
	key := scale.MustMarshal(data)
	key = append(key, suffix...)
	return key
}

type StateDBSync[BlockHash Hash, Key Hash] struct {
	mode         PruningMode
	nonCanonical NonCanonicalOverlay[BlockHash, Key]
	pruning      *refWindow[BlockHash, Key]
	pinned       map[BlockHash]uint32
	refCounting  bool
}

type StateDB struct {
}
