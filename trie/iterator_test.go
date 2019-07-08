package trie

import (
	"fmt"
	"testing"
)

func printNode(current node, withEncoding bool) {
	h, err := NewHasher()
	if err != nil {
		fmt.Printf("new hasher err %s\n", err)
	}

	var encoding []byte
	var hash []byte
	if withEncoding && current != nil {
		encoding, err = current.Encode()
		if err != nil {
			fmt.Printf("encoding err %s\n", err)
		}
		hash, err = h.Hash(current)
		if err != nil {
			fmt.Printf("hashing err %s\n", err)
		}
	}

	switch c := current.(type) {
	case *branch:
		fmt.Printf("branch key %x children %b value %x\n", nibblesToKeyLE(c.key), c.childrenBitmap(), c.value)
		if withEncoding {
			fmt.Printf("branch encoding ")
			printHexBytes(encoding)
			fmt.Printf("branch hash ")
			printHexBytes(hash)
		}
	case *leaf:
		fmt.Printf("leaf key %x value %x\n", nibblesToKeyLE(c.key), c.value)
		if withEncoding {
			fmt.Printf("leaf encoding ")
			printHexBytes(encoding)
			fmt.Printf("leaf hash ")
			printHexBytes(hash)
		}
	}
}

func TestEntries(t *testing.T) {
	trie := newEmpty()

	tests := []trieTest{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
	}

	for _, test := range tests {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Fatal(err)
		}
	}

	entries := trie.Entries()
	for k, v := range entries {
		t.Logf("key %x value %s", []byte(k), v)
	}

	if len(entries) != len(tests) {
		t.Fatal("length of trie.Entries does not equal length of values put into trie")
	}

	for _, test := range tests {
		if entries[string(test.key)] == nil {
			t.Fatal("did not get entry in trie")
		}
	}
}
