package polkadb

import (
	"testing"
	"io/ioutil"
	"github.com/stretchr/testify/require"
	"fmt"
)

// runBadgerTest opens a badger db and runs a a test on it.
func runBadgerTest(t *testing.T, test func(t *testing.T, db *BadgerDB)) {
	dir, err := ioutil.TempDir(".", "badger-test")
	if err != nil {
		t.Error(err)
	}
	db, err := NewBadgerDB(dir)
	if err != nil {
		t.Error(err)
	}
	test(t, db)
}

func TestWrite(t *testing.T) {
	runBadgerTest(t, func(t *testing.T, db *BadgerDB) {
		for i := 0; i < 100; i++ {
			txnSet(t, db, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("val%d", i)), 0x00)
		}
	})
}

func txnSet(t *testing.T, kv *BadgerDB, key []byte, val []byte, meta byte) {
	txn := kv.db.NewTransaction(true)
	require.NoError(t, txn.SetWithMeta(key, val, meta))
	require.NoError(t, txn.Commit())
}