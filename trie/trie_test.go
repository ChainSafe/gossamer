package trie

import "github.com/chainsafe/go-pre/databases"

func newEmpty() *Trie {
	var test [32]byte
	t := NewTrie(NewDatabase(databases.NewMemDatabase()), test)
	return t
}



