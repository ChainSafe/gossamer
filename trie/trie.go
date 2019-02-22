package trie

import (
	"log"
	"fmt"
)

// Trie is a Merkle Patricia Trie.
// The zero value is an empty trie with no database.
// Use NewTrie to create a trie that sits on top of a database.
type Trie struct {
	db *Database
	root [32]byte
}

// NewTrie creates a trie with an existing root node from db.
func NewTrie(db *Database, root [32]byte) *Trie {
	return &Trie{
		db:	  db,
		root: root,
	}
}

// Put associates key with value in the trie.
func(t *Trie) Put(k, v  []byte){
	if err := t.TryPut(k,v); err != nil {
		log.Println("Error in trie", err.Error())
	}
}

func(t *Trie) TryPut(k, v []byte) error {
	return nil
}

func(t *Trie) set(n node, k, v []byte) (bool, error) {
	switch n := n.(type) {
	case *extension:
		fmt.Println(n)
		return false, nil
	case *branch:
		return false, nil
	case *leaf:
		return false, nil
	default:
		fmt.Println("ERROR")
	}
	return false, nil
}

// Get returns the value for key stored in the trie.
func(t *Trie) Get(key []byte){}

// Delete removes any existing value for key from the trie.
func(t *Trie) Del(key []byte){}
