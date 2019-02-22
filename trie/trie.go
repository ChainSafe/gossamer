package trie

import (
	"log"
	"fmt"
)

type Trie struct {
	db *Database
	root [32]byte
}

func NewTrie(db *Database, root [32]byte) *Trie {
	return &Trie{
		db:	  db,
		root: root,
	}
}

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


func(t *Trie) Get(key []byte){}

func(t *Trie) Del(key []byte){}
