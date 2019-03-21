package polkadb

import (
	"log"

	"fmt"
	"github.com/dgraph-io/badger"
	"github.com/golang/snappy"
)

// BadgerDB struct contains directory path to data and db instance
type BadgerDB struct {
	path string
	db   *badger.DB
}

// Iterator struct contains a transaction, iterator and context fields released, initialized
type Iterator struct {
	txn      *badger.Txn
	iter     *badger.Iterator
	released bool
	init     bool
}

// Batch struct contains a database instance, key-value mapping for batch writes and length of item value for batch write
type batchWriter struct {
	db   *BadgerDB
	b    map[string][]byte
	size int
}

type table struct {
	db     Database
	prefix string
}

type tableBatch struct {
	batch  Batch
	prefix string
}

// NewBadgerDB opens and returns a new DB object
func NewBadgerDB(file string) (*BadgerDB, error) {
	opts := badger.DefaultOptions
	opts.Dir = file
	opts.ValueDir = file
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &BadgerDB{
		path: file,
		db:   db,
	}, nil
}

// Path returns the path to the database directory.
func (db *BadgerDB) Path() string {
	return db.path
}

func (db *BadgerDB) NewBatch() Batch {
	return &batchWriter{
		db: db,
		b: make(map[string][]byte),
	}
}

// Put puts the given key / value to the queue
func (db *BadgerDB) Put(key []byte, value []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(snappy.Encode(nil, key), snappy.Encode(nil, value))
		return err
	})
}

// Has checks the given key exists already; returning true or false
func (db *BadgerDB) Has(key []byte) (exists bool, err error) {
	err = db.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(snappy.Encode(nil, key))
		if item != nil {
			exists = true
		}
		if err == badger.ErrKeyNotFound {
			exists = false
			err = nil
		}
		return err
	})
	return exists, err
}

// Get returns the given key
func (db *BadgerDB) Get(key []byte) (data []byte, err error) {
	_ = db.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(snappy.Encode(nil, key))
		if err != nil {
			return err
		}
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		data, _ = snappy.Decode(nil, val)
		return nil
	})
	return data, nil
}

// Del removes the key from the queue and database
func (db *BadgerDB) Del(key []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(snappy.Encode(nil, key))
		if err == badger.ErrKeyNotFound {
			err = nil
		}
		return err
	})
}

// Close closes a DB
func (db *BadgerDB) Close() {
	err := db.db.Close()
	if err == nil {
		log.Println("Database closed")
	} else {
		log.Fatal("Failed to close database", "err", err)
	}
}

// NewIterator returns a new iterator within the Iterator struct along with a new transaction
func (db *BadgerDB) NewIterator() Iterator {
	txn := db.db.NewTransaction(false)
	opts := badger.DefaultIteratorOptions
	iter := txn.NewIterator(opts)
	return Iterator{
		txn:      txn,
		iter:     iter,
		released: false,
		init:     false,
	}
}

// Release closes the iterator, discards the created transaction and sets released value to true
func (i *Iterator) Release() {
	i.iter.Close()
	i.txn.Discard()
	i.released = true
}

// Released returns the boolean indicating whether the iterator and transaction was successfully released
func (i *Iterator) Released() bool {
	return i.released
}

// Next rewinds the iterator to the zero-th position if uninitialized, and then will advance the iterator by one
// returns bool to ensure access to the item
func (i *Iterator) Next() bool {
	if !i.init {
		i.iter.Rewind()
		i.init = true
	}
	i.iter.Next()
	return i.iter.Valid()
}

// Seek will look for the provided key if present
func (i *Iterator) Seek(key []byte) {
	i.iter.Seek(snappy.Encode(nil, key))
}

// Key returns an item key
func (i *Iterator) Key() []byte {
	fmt.Println("key")
	ret, err := snappy.Decode(nil, i.iter.Item().Key())
	if err != nil {
		fmt.Println("key retrieval error ", err.Error())
	}
	return ret
}

// Value returns a copy of the value of the item
func (i *Iterator) Value() []byte {
	val, err := i.iter.Item().ValueCopy(nil)
	if err != nil {
		fmt.Println("value retrieval error ", err.Error())
	}
	ret, err := snappy.Decode(nil, val)
	if err != nil {
		fmt.Println("value decoding error ", err.Error())
	}
	return ret
}

// Put encodes key-values and adds them to a mapping for batch writes, sets the size of item value
func (b *batchWriter) Put(key, value []byte) error {
	encodedKey := snappy.Encode(nil, key)
	encodedVal := snappy.Encode(nil, value)
	b.b[string(encodedKey)] = encodedVal
	b.size += len(value)
	return nil
}

// Write performs batched writes
func (b *batchWriter) Write() error {
	wb := b.db.db.NewWriteBatch()
	defer wb.Cancel()

	for k, v := range b.b {
		err := wb.Set([]byte(k), v, 0)
		if err != nil {
			fmt.Println("error writing batch txs", err.Error())
		}
	}
	if err := wb.Flush(); err != nil {
		fmt.Println("error stored by writeBatch ", err.Error())
	}
	return nil
}

// ValueSize returns the amount of data in the batch
func (b *batchWriter) ValueSize() int {
	return b.size
}

// Delete removes the key from the batch and database
func (b *batchWriter) Delete(key []byte) error {
	err := b.db.db.NewWriteBatch().Delete(key)
	if err != nil {
		fmt.Println("error batch deleting key ", err.Error())
	}
	b.size++
	return nil
}

// Reset clears batch key-values and resets the size to zero
func (b *batchWriter) Reset() {
	b.b = make(map[string][]byte)
	b.size = 0
}

// NewTable returns a Database object that prefixes all keys with a given
// string.
func NewTable(db Database, prefix string) Database {
	return &table{
		db:     db,
		prefix: prefix,
	}
}

func (dt *table) Put(key []byte, value []byte) error {
	return dt.db.Put(append([]byte(dt.prefix), key...), value)
}

func (dt *table) Has(key []byte) (bool, error) {
	return dt.db.Has(append([]byte(dt.prefix), key...))
}

func (dt *table) Get(key []byte) ([]byte, error) {
	return dt.db.Get(append([]byte(dt.prefix), key...))
}

func (dt *table) Del(key []byte) error {
	return dt.db.Del(append([]byte(dt.prefix), key...))
}

// NewTableBatch returns a Batch object which prefixes all keys with a given string.
func NewTableBatch(db Database, prefix string) Batch {
	return &tableBatch{db.NewBatch(), prefix}
}

func (dt *table) NewBatch() Batch {
	return &tableBatch{dt.db.NewBatch(), dt.prefix}
}

func (tb *tableBatch) Put(key, value []byte) error {
	return tb.batch.Put(append([]byte(tb.prefix), key...), value)
}

func (tb *tableBatch) Write() error {
	return tb.batch.Write()
}

func (tb *tableBatch) ValueSize() int {
	return tb.batch.ValueSize()
}

func (tb *tableBatch) Reset() {
	tb.batch.Reset()
}

func (tb *tableBatch) Delete(k []byte) {
	return tb.batch.Delete(k)
}