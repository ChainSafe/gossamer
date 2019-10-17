package main

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/trie"
)

func TestLoadTrie(t *testing.T) {
	data := map[string]string{"0x1234": "0x5678", "0xaabbcc": "0xddeeff"}
	testTrie := &trie.Trie{}

	err := loadTrie(testTrie, data)
	if err != nil {
		t.Fatal(err)
	}

	expectedTrie := &trie.Trie{}
	var keyBytes, valueBytes []byte
	for key, value := range data {
		keyBytes, err = common.HexToBytes(key)
		if err != nil {
			t.Fatal(err)
		}
		valueBytes, err = common.HexToBytes(value)
		if err != nil {
			t.Fatal(err)
		}
		err = expectedTrie.Put(keyBytes, valueBytes)
		if err != nil {
			t.Fatal(err)
		}
	}

	testhash, err := testTrie.Hash()
	if err != nil {
		t.Fatal(err)
	}
	expectedhash, err := expectedTrie.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(testhash[:], expectedhash[:]) {
		t.Fatalf("Fail: got %x expected %x", testhash, expectedhash)
	}
}
