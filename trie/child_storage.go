package trie

import (
	"fmt"

	"github.com/ChainSafe/gossamer/common"
)

var ChildStorageKeyPrefix = []byte(":child_storage:")

func (t *Trie) PutChild(childKey []byte, child *Trie) error {
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
	err = t.Put(key, value[:])
	if err != nil {
		return err
	}

	t.children[common.Hash(childHash)] = child
	return nil
}

func (t *Trie) GetChild(storageKey []byte) (*Trie, error) {
	childHash, err := t.Get(storageKey)
	if err != nil {
		return nil, err
	}

	fmt.Println(childHash)

	hash := [32]byte{}
	copy(hash[:], childHash)
	return t.children[common.Hash(hash)], nil
}

func (t *Trie) PutIntoChild(storageKey, key, value []byte) error {
	childTrie, err := t.GetChild(storageKey)
	if err != nil {
		return err
	}

	return childTrie.Put(key, value)
}

func (t *Trie) GetFromChild(storageKey, key []byte) ([]byte, error) {
	childTrie, err := t.GetChild(storageKey)
	if err != nil {
		return nil, err
	}

	return childTrie.Get(key)
}