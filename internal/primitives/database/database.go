package database

import "github.com/ChainSafe/gossamer/internal/primitives/runtime"

// / An identifier for a column.
// pub type ColumnId = u32;
type ColumnID uint32

// / An alteration to the database.
type Change[H any] interface{}
type Changes[H any] interface {
	ChangeSet | ChangeRemove | ChangeStore[H] | ChangeReference[H] | ChangeRelease[H]
}
type ChangeSet struct {
	ColumnID
	Key   []byte
	Value []byte
}
type ChangeRemove struct {
	ColumnID
	Key []byte
}
type ChangeStore[H any] struct {
	ColumnID
	Key   H
	Value []byte
}
type ChangeReference[H any] struct {
	ColumnID
	Key H
}
type ChangeRelease[H any] struct {
	ColumnID
	Key H
}

// / A series of changes to the database that can be committed atomically. They do not take effect
// /// until passed into `Database::commit`.
// #[derive(Default, Clone)]
// pub struct Transaction<H>(pub Vec<Change<H>>);
type Transaction[H any] []Change[H]

// / Set the value of `key` in `col` to `value`, replacing anything that is there currently.
func (t *Transaction[H]) SetFromVec(col ColumnID, key []byte, value []byte) {
	*t = append(*t, ChangeSet{col, key, value})
}

// / Remove the value of `key` in `col`.
func (t *Transaction[H]) Remove(col ColumnID, key []byte) {
	*t = append(*t, ChangeRemove{col, key})
}

type Database[H runtime.Hash] interface {
	/// Commit the `transaction` to the database atomically. Any further calls to `get` or `lookup`
	/// will reflect the new state.
	// fn commit(&self, transaction: Transaction<H>) -> error::Result<()>;
	Commit(transaction Transaction[H]) error

	/// Retrieve the value previously stored against `key` or `None` if
	/// `key` is not currently in the database.
	// fn get(&self, col: ColumnId, key: &[u8]) -> Option<Vec<u8>>;
	Get(col ColumnID, key []byte) *[]byte

	/// Check if the value exists in the database without retrieving it.
	// fn contains(&self, col: ColumnId, key: &[u8]) -> bool {
	// 	self.get(col, key).is_some()
	// }
	Contains(col ColumnID, key []byte) bool

	/// Check value size in the database possibly without retrieving it.
	// fn value_size(&self, col: ColumnId, key: &[u8]) -> Option<usize> {
	// 	self.get(col, key).map(|v| v.len())
	// }

	/// Call `f` with the value previously stored against `key`.
	///
	/// This may be faster than `get` since it doesn't allocate.
	/// Use `with_get` helper function if you need `f` to return a value from `f`
	// fn with_get(&self, col: ColumnId, key: &[u8], f: &mut dyn FnMut(&[u8])) {
	// 	self.get(col, key).map(|v| f(&v));
	// }

	/// Check if database supports internal ref counting for state data.
	///
	/// For backwards compatibility returns `false` by default.
	// fn supports_ref_counting(&self) -> bool {
	// 	false
	// }

	/// Remove a possible path-prefix from the key.
	///
	/// Not all database implementations use a prefix for keys, so this function may be a noop.
	// fn sanitize_key(&self, _key: &mut Vec<u8>) {}
}
