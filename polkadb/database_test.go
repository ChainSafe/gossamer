package polkadb_test

import (
	"io/ioutil"
	"os"
	"github.com/chainsafe/go-pre/polkadb"
)

func newTestBadgerDB() (*polkadb.BadgerDB, func()) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger-test")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}
	db, err := polkadb.NewBadgerDB(dir)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}
	return db, func() {
		db.Close()
		os.RemoveAll(dir)
	}
}

