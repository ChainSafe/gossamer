// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package kvdb

import "iter"

// / Database value.
type DBValue []byte

// / Database keys.
type DBKey []byte

type DBKeyValue struct {
	Key   DBKey
	Value DBValue
}

// / Database operation.
type DBOp interface {
	Key() []byte
	Col() uint32
}

type InsertDBOp struct {
	col   uint32
	key   DBKey
	Value DBValue
}

func (idbo InsertDBOp) Key() []byte {
	return idbo.key
}
func (idbo InsertDBOp) Col() uint32 {
	return idbo.col
}

type DeleteDBOp struct {
	col uint32
	key DBKey
}

func (idbo DeleteDBOp) Key() []byte {
	return idbo.key
}
func (idbo DeleteDBOp) Col() uint32 {
	return idbo.col
}

type DeletePrefixDBOp struct {
	col    uint32
	Prefix DBKey
}

func (idbo DeletePrefixDBOp) Key() []byte {
	return idbo.Prefix
}
func (idbo DeletePrefixDBOp) Col() uint32 {
	return idbo.col
}

// / Write transaction. Batches a sequence of put/delete operations for efficiency.
type DBTransaction struct {
	/// Database operations.
	Ops []DBOp
}

// / Create new transaction.
func NewDBTransaction() DBTransaction {
	return DBTransaction{
		Ops: make([]DBOp, 0),
	}
}

// / Insert a key-value pair in the transaction. Any existing value will be overwritten upon write.
func (dbt *DBTransaction) Put(col uint32, key, value []byte) {
	dbt.Ops = append(dbt.Ops, InsertDBOp{
		col:   col,
		key:   key,
		Value: value,
	})
}

// / Insert a key-value pair in the transaction. Any existing value will be overwritten upon write.
func (dbt *DBTransaction) Delete(col uint32, key []byte) {
	dbt.Ops = append(dbt.Ops, DeleteDBOp{
		col: col,
		key: key,
	})
}

// / Delete all values with the given key prefix.
// / Using an empty prefix here will remove all keys
// / (all keys start with the empty prefix).
func (dbt *DBTransaction) DeletePrefix(col uint32, prefix []byte) {
	dbt.Ops = append(dbt.Ops, DeletePrefixDBOp{
		col:    col,
		Prefix: prefix,
	})
}

// / Generic key-value database.
// /
// / The `KeyValueDB` deals with "column families", which can be thought of as distinct
// / stores within a database. Keys written in one column family will not be accessible from
// / any other. The number of column families must be specified at initialization, with a
// / differing interface for each database.
// /
// / The API laid out here, along with the `Sync` bound implies interior synchronization for
// / implementation.
type KeyValueDB interface {
	/// Get a value by key.
	Get(col uint32, key []byte) (DBValue, error)

	/// Get the first value matching the given prefix.
	PrefixGet(col uint32, prefix []byte) (DBValue, error)

	/// Write a transaction of changes to the backing store.
	// fn write(&self, transaction: DBTransaction) -> io::Result<()>;
	Write(transaction DBTransaction) error

	/// Iterate over the data for a given column.
	// fn iter<'a>(&'a self, col: u32) -> Box<dyn Iterator<Item = io::Result<DBKeyValue>> + 'a>;
	Iter(col uint32) iter.Seq2[DBKeyValue, error]

	/// Iterate over the data for a given column, returning all key/value pairs
	/// where the key starts with the given prefix.
	// fn iter_with_prefix<'a>(
	// 	&'a self,
	// 	col: u32,
	// 	prefix: &'a [u8],
	// ) -> Box<dyn Iterator<Item = io::Result<DBKeyValue>> + 'a>;
	PrefixIter(col uint32, prefix []byte) iter.Seq2[DBKeyValue, error]

	/// Query statistics.
	///
	/// Not all kvdb implementations are able or expected to implement this, so by
	/// default, empty statistics is returned. Also, not all kvdb implementations
	/// can return every statistic or configured to do so (some statistics gathering
	/// may impede the performance and might be off by default).
	// fn io_stats(&self, _kind: IoStatsKind) -> IoStats {
	// 	IoStats::empty()
	// }

	/// Check for the existence of a value by key.
	// fn has_key(&self, col: u32, key: &[u8]) -> io::Result<bool> {
	// 	self.get(col, key).map(|opt| opt.is_some())
	// }
	HasKey(col uint32, key []byte) (bool, error)

	/// Check for the existence of a value by prefix.
	// fn has_prefix(&self, col: u32, prefix: &[u8]) -> io::Result<bool> {
	// 	self.get_by_prefix(col, prefix).map(|opt| opt.is_some())
	// }
	HasPrefix(col uint32, prefix []byte) (bool, error)
}

// / For a given start prefix (inclusive), returns the correct end prefix (non-inclusive).
// / This assumes the key bytes are ordered in lexicographical order.
// / Since key length is not limited, for some case we return `None` because there is
// / no bounded limit (every keys in the serie `[]`, `[255]`, `[255, 255]` ...).
func EndPrefix(prefix []byte) []byte {
	for len(prefix) > 0 && prefix[len(prefix)-1] == 0xff {
		prefix = prefix[:len(prefix)-1]
	}
	if len(prefix) > 0 {
		last := prefix[len(prefix)-1]
		last += 1
		prefix[len(prefix)-1] = last
		return prefix
	} else {
		return nil
	}
}
