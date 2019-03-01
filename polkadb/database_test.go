package polkadb_test

import (
	"io/ioutil"
	"os"
	"github.com/chainsafe/go-pre/polkadb"
	"testing"
	"bytes"
)


var test_values = []string{"noot", "dutter", "204294", "\x00123\x00"}

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

func TestBadgerDB_PutGet(t *testing.T) {
	db, remove := newTestBadgerDB()
	defer remove()
	testPutGet(db, t)
}

func testPutGet(db polkadb.Database, t *testing.T) {
	t.Parallel()

	for _, v := range test_values {
		err := db.Put([]byte(v), []byte(v))
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}
	}

	for _, v := range test_values {
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte(v)) {
			t.Fatalf("get returned wrong result, got %q expected %q", string(data), v)
		}
	}

	for _, v := range test_values {
		err := db.Put([]byte(v), []byte("?"))
		if err != nil {
			t.Fatalf("put override failed: %v", err)
		}
	}

	for _, v := range test_values {
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte("?")) {
			t.Fatalf("get returned wrong result, got %q expected ?", string(data))
		}
	}

	for _, v := range test_values {
		orig, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		orig[0] = byte(0xff)
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte("?")) {
			t.Fatalf("get returned wrong result, got %q expected ?", string(data))
		}
	}

	for _, v := range test_values {
		err := db.Del([]byte(v))
		if err != nil {
			t.Fatalf("delete %q failed: %v", v, err)
		}
	}

	for _, v := range test_values {
		_, err := db.Get([]byte(v))
		if err == nil {
			t.Fatalf("got deleted value %q", v)
		}
	}
}