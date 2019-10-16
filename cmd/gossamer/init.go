package main

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/trie"
)

func loadTrie(t *trie.Trie, data []map[string]string) error {
	for _, field := range data {
		for key, value := range field {
			keyBytes, err := common.HexToBytes(key)
			if err != nil {
				return err
			}
			valueBytes, err := common.HexToBytes(value)
			if err != nil {
				return err
			}
			err = t.Put(keyBytes, valueBytes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func commitToDb(t *trie.Trie) error {
	err := t.WriteToDB()
	if err != nil {
		return err
	}
	err = t.Commit()
	return err
}
