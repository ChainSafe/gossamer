package trie

import (
	"fmt"

	"github.com/ChainSafe/gossamer/common"
)

var ChildStorageKeyPrefix = []byte(":child_storage:")

func (t *Trie) PutChild(child *Trie, childKey []byte) error {
	childHash, err := child.Hash()
	if err != nil {
		return err
	}

	key := append(ChildStorageKeyPrefix, childKey[:]...)
	exists, err := t.Get(key)
	if err != nil {
		return err
	}

	if exists != nil {
		return fmt.Errorf("child already exists at %s%x", ChildStorageKeyPrefix, childHash)
	}

	value := [32]byte(childHash)
	return t.Put(key, value[:])
}

func (t *Trie) PutIntoChild(storageKey, key, value []byte) error {
	childHash, err := t.Get(storageKey)
	if err != nil {
		return err
	}

	hash := [32]byte{}
	copy(hash[:], childHash)
	childTrie := t.children[common.Hash(hash)]

	return childTrie.Put(key, value)
}

//func (t *Trie)