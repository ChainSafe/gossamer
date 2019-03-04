package polkadb

import (
	"io/ioutil"
	"os"
	"testing"
	"bytes"
	"fmt"
)

func newTestBadgerDB() (*BadgerDB, func()) {
	dir, err := ioutil.TempDir(os.TempDir(), "badger-test")
	fmt.Println(dir)
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}
	db, err := NewBadgerDB(dir)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}
	return db, func() {
		db.Close()
		os.RemoveAll(dir)
	}
}

func TestBadgerDB_PutGetDel(t *testing.T) {
	db, remove := newTestBadgerDB()
	defer remove()
	testPutGetter(db, t)
	testDelGet(db ,t)
}

func testPutGetter(db Database, t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"camel", "camel"},
		{"walrus", "walrus"},
		{"296204", "296204"},
		{"\x00123\x00", "\x00123\x00"},
	}
	for _, v := range tests {
		err := db.Put([]byte(v.input), []byte(v.input))
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}
	}
	for _, v := range tests {
		data, err := db.Get([]byte(v.input))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte(v.expected)) {
			t.Fatalf("get returned wrong result, got %q expected %q", string(data), v.expected)
		}
	}

	for _, v := range tests {
		exists, err := db.Has([]byte(v.input))
		if err != nil {
			t.Fatalf("has operation failed: %v", err)
		}
		if !exists {
			t.Fatalf("has operation returned wrong result, got %t expected %t", exists, true)
		}
	}

	for _, v := range tests {
		err := db.Put([]byte(v.input), []byte("?"))
		if err != nil {
			t.Fatalf("put override failed: %v", err)
		}
	}

	for _, v := range tests {
		data, err := db.Get([]byte(v.input))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte("?")) {
			t.Fatalf("get returned wrong result, got %q expected ?", string(data))
		}
	}
}

func testDelGet(db Database, t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"camel", "camel"},
		{"walrus", "walrus"},
		{"296204", "296204"},
		{"\x00123\x00", "\x00123\x00"},
	}

	for _, v := range tests {
		err := db.Del([]byte(v.input))
		if err != nil {
			t.Fatalf("delete %q failed: %v", v.input, err)
		}
	}

	for _, v := range tests {
		d, err := db.Get([]byte(v.input))
		if err != nil {
			t.Fatalf("got deleted value %q failed: %v", v.input, err)
		}
		if len(d) > 1 {
			t.Fatalf("failed to delete value %q", v.input)
		}
	}
}