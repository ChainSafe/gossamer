package trie

import (
	"fmt"
)

var ChildStorageKeyPrefix = []byte(":child_storage:")

func (t *Trie) PutChild(child *Trie) error {
	childHash, err := child.Hash()
	if err != nil {
		return err
	}

	childBytes := [32]byte(childHash)
	key := append(ChildStorageKeyPrefix, childBytes[:]...)
	exists, err := t.Get(key)
	if err != nil {
		return err
	}

	if exists != nil {
		return fmt.Errorf("child already exists at %s%x", ChildStorageKeyPrefix, childHash)
	}

	return t.Put(key, childBytes[:])
}