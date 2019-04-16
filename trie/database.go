package trie

import (
	"github.com/ChainSafe/gossamer/polkadb"
	"sync"
)

// Database is a wrapper around a polkadb
type Database struct {
	db polkadb.Database
	lock sync.RWMutex
}
 
// WriteToDB writes the trie to the underlying database
// Stores the merkle value of the node as the key and the encoded node as the value
func (t *Trie) WriteToDB() error {
	return t.writeToDB(t.root)
}

func (t *Trie) writeToDB(n node) error {
	err := t.writeNodeToDB(n)
	if err != nil {
		return err
	}
	
	switch n := n.(type) {
	case *branch:
		for _, child := range n.children {
			if child != nil {
				err = t.writeToDB(child)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (t *Trie) writeNodeToDB(n node) error {
	encRoot, err := Encode(t.root)
	if err != nil {
		return err
	}

	hash, err := Hash(t.root)
	if err != nil {
		return err
	}

	t.db.lock.Lock()
	err = t.db.db.Put(hash, encRoot)
	t.db.lock.Unlock()
	return err
}