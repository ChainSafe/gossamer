package polkadb

import (
	"github.com/dgraph-io/badger"
	"github.com/golang/snappy"
	"log"
)

type BadgerDB struct {
	dir string
	db *badger.DB
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
	defer db.Close()

	return &BadgerDB{
		dir: file,
		db:  db,
	}, nil
}

// Path returns the path to the database directory.
func (db *BadgerDB) Path() string {
	return db.dir
}

// Put puts the given key / value to the queue
func (db *BadgerDB) Put(key []byte, value []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(snappy.Encode(nil, key), snappy.Encode(nil, value))
		return err
	})
}

// Get returns the given key
func (db *BadgerDB) Get(key []byte) (data []byte, err error) {
	err = db.db.View(func(txn *badger.Txn) error {
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












