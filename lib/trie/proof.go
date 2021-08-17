package trie

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	ErrEmptyTrieRoot = errors.New("provided trie must have a root")
	ErrEmptyNibbles  = errors.New("empty nibbles provided from key")
	ErrInvalidProof  = errors.New("provided key is not present at trie")
)

func (t *Trie) Prove(key []byte, fromLvl uint, db chaindb.Writer) error {
	key = keyToNibbles(key)
	if len(key) == 0 {
		return ErrEmptyNibbles
	}

	var nodes []node
	currNode := t.root

proveLoop:
	for {
		switch n := currNode.(type) {
		case nil:
			break proveLoop

		case *leaf:
			nodes = append(nodes, n)

			if bytes.Equal(n.key, key) {
				break proveLoop
			}

			return errors.New("could not found key")

		case *branch:
			nodes = append(nodes, n)
			if bytes.Equal(n.key, key) || len(key) == 0 {
				break proveLoop
			}

			length := lenCommonPrefix(n.key, key)
			if length > 0 && len(key) < len(n.key) {
				return errors.New("could not found key")
			}

			currNode = n.children[key[length]]
			key = key[length+1:]
		}
	}

	for _, n := range nodes {
		if fromLvl > 0 {
			fromLvl--
			continue
		}

		var (
			hashNode    []byte
			encHashNode []byte
			err         error
		)

		if encHashNode, hashNode, err = n.encodeAndHash(); err != nil {
			return fmt.Errorf("problems while encoding and hashing the node: %w", err)
		}

		fmt.Printf("0x%x\n", hashNode[:])

		if err = db.Put(hashNode, encHashNode); err != nil {
			return err
		}
	}

	return nil
}

func VerifyProof(rootHash common.Hash, key []byte, db chaindb.Reader) ([]byte, error) {
	key = keyToNibbles(key)
	wantedHash := rootHash

	for {
		fmt.Printf("verifying 0x%x\n", wantedHash)
		enc, err := db.Get(wantedHash[:])
		if err != nil {
			return nil, fmt.Errorf("could not get hash %s while verifying proof: %w", wantedHash, err)
		}

		node, err := decodeBytes(enc)
		if err != nil {
			return nil, fmt.Errorf("could not decode node bytes: %w", err)
		}

		switch n := node.(type) {
		case nil:
			return nil, ErrInvalidProof
		case *leaf:
			if bytes.Equal(n.key, key) {
				return n.value, nil
			}

			return nil, ErrInvalidProof
		case *branch:
			if bytes.Equal(n.key, key) {
				return n.value, nil
			}

			if len(key) == 0 {
				return nil, ErrInvalidProof
			}

			length := lenCommonPrefix(n.key, key)
			node = n.children[key[length]]
			if node == nil {
				return nil, fmt.Errorf("key: 0x%x not found on branch to key 0x%x,", key[0], key[:])
			}

			key = key[length+1:]
			copy(wantedHash[:], node.getHash())
		}
	}
}
