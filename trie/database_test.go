package trie

import (
	"bytes"
	"testing"

	db "github.com/ChainSafe/gossamer/polkadb"
)

func newTrie() (*Trie, error) {
	hasher, err := newHasher()
	if err != nil {
		return nil, err
	}

	db, err := db.NewBadgerDB("./gossamer_data")
	if err != nil {
		return nil, err
	}

	trie := &Trie{
		db: &Database{
			db:     db,
			hasher: hasher,
		},
		root: nil,
	}

	trie.db.batch = trie.db.db.NewBatch()

	return trie, nil
}

func TestWriteToDB(t *testing.T) {
	trie, err := newTrie()
	if err != nil {
		t.Fatal(err)
	}

	rt := generateRandTest(20000)
	var val []byte
	for _, test := range rt {
		err = trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}

		val, err = trie.Get(test.key)
		if err != nil {
			t.Errorf("Fail to get key %x: %s", test.key, err.Error())
		} else if !bytes.Equal(val, test.value) {
			t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
		}
	}

	err = trie.WriteToDB()
	if err != nil {
		t.Errorf("Fail: could not write to batch writer: %s", err)
	}

	err = trie.Commit()
	if err != nil {
		t.Errorf("Fail: could not commit (batch write) to DB: %s", err)
	}

	trie.db.db.Close()
}

func TestWriteDirty(t *testing.T) {
	trie, err := newTrie()
	if err != nil {
		t.Fatal(err)
	}

	dirtyNode := &leaf{key: generateRandBytes(10), value: generateRandBytes(10), dirty: true}
	written, err := trie.writeNodeToDB(dirtyNode)
	if err != nil {
		t.Errorf("Fail: could not write to db: %s", err)
	} else if !written {
		t.Errorf("Fail: did not write dirty node to db")
	}

	cleanNode := &leaf{key: generateRandBytes(10), value: generateRandBytes(10), dirty: false}
	written, err = trie.writeNodeToDB(cleanNode)
	if err != nil {
		t.Errorf("Fail: could not write to db: %s", err)
	} else if written {
		t.Errorf("Fail: wrote clean node to db")
	}
}
