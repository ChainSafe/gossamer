package trie

import (
	"errors"
)

var (
	ErrInvalidProof = errors.New("provided key is not present at trie")
)

func VerifyProof(t Trie, key []byte) (value []byte, err error) {
	if t.root == nil {
		return nil, errors.New("cannot verify proof of an empty")
	}

	key = keyToNibbles(key)
	if len(key) == 0 {
		return nil, errors.New("cannot verify proof of an empty key")
	}

	retrievedNode := t.retrieve(t.root, key)
	if retrievedNode == nil {
		return nil, ErrInvalidProof
	}

	return retrievedNode.value, nil
}
