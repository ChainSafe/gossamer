package statedb

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
)

type TestDB struct {
	Data map[hash.H256]DBValue
	Meta map[string]DBValue
}

func NewTestDB(inserted []uint64) TestDB {
	data := make(map[hash.H256]DBValue)
	for _, v := range inserted {
		data[hash.NewH256FromLowUint64BigEndian(v)] = DBValue(hash.NewH256FromLowUint64BigEndian(v))
	}
	return TestDB{
		Data: data,
		Meta: make(map[string]DBValue),
	}
}

func (tdb TestDB) GetMeta(key []byte) (*DBValue, error) {
	val, ok := tdb.Meta[string(key)]
	if !ok {
		return nil, nil
	}
	return &val, nil
}

func (tdb *TestDB) Commit(commitSet CommitSet[hash.H256]) {
	for _, insert := range commitSet.Data.Inserted {
		tdb.Data[insert.Hash] = insert.DBValue
	}
	for _, insert := range commitSet.Meta.Inserted {
		tdb.Meta[string(insert.Hash)] = insert.DBValue
	}
	for _, k := range commitSet.Data.Deleted {
		delete(tdb.Data, k)
	}
	for _, k := range commitSet.Meta.Deleted {
		delete(tdb.Meta, string(k))
	}
}

func NewCommit(inserted []uint64, deleted []uint64) CommitSet[hash.H256] {
	return CommitSet[hash.H256]{
		Data: NewChangeset(inserted, deleted),
	}
}

func NewChangeset(inserted []uint64, deleted []uint64) ChangeSet[hash.H256] {
	var insertedHDBVs []HashDBValue[hash.H256]
	for _, v := range inserted {
		insertedHDBVs = append(insertedHDBVs, HashDBValue[hash.H256]{
			Hash:    hash.NewH256FromLowUint64BigEndian(v),
			DBValue: DBValue(hash.NewH256FromLowUint64BigEndian(v)),
		})
	}
	var deletedHashes []hash.H256
	for _, v := range deleted {
		deletedHashes = append(deletedHashes, hash.NewH256FromLowUint64BigEndian(v))
	}
	return ChangeSet[hash.H256]{
		Inserted: insertedHDBVs,
		Deleted:  deletedHashes,
	}
}
