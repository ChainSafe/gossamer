package trie

import (
	"bytes"
	"testing"

	db "github.com/ChainSafe/gossamer/polkadb"
)

func TestWriteToDB(t *testing.T) {
	db, err := db.NewBadgerDB("./gossamer-data")
	if err != nil {
		t.Fatalf("Fail: could not create badgerDB")
	}
	trie := &Trie{
		db: &Database{db: db},
		root: nil,
	}

	rt := generateRandTest(20000)
	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}

		val, err := trie.Get(test.key)
		if err != nil {
			t.Errorf("Fail to get key %x: %s", test.key, err.Error())
		} else if !bytes.Equal(val, test.value) {
			t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
		}
	}

	err = trie.WriteToDB()
	if err != nil {
		t.Errorf("Fail: could not write to DB: %s", err)
	}
}