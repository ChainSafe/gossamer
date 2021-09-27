package trie

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/chaindb"
)

var (
	// ErrProofNodeNotFound when a needed proof node is not in the database
	ErrProofNodeNotFound = errors.New("cannot find a trie node in the database")
)

// lookup struct holds the state root and database reference
// used to retrieve trie information from database
type lookup struct {
	// root to start the lookup
	root []byte
	db   chaindb.Database
}

// newLookup returns a Lookup to helps the proof generator
func newLookup(rootHash []byte, db chaindb.Database) *lookup {
	lk := &lookup{db: db}
	lk.root = make([]byte, len(rootHash))
	copy(lk.root, rootHash)

	return lk
}

// find will return the desired value or nil if key cannot be found and will record visited nodes
func (l *lookup) find(key []byte, recorder *recorder) ([]byte, error) {
	partial := key
	hash := l.root

	for {
		nodeData, err := l.db.Get(hash)
		if err != nil {
			return nil, ErrProofNodeNotFound
		}

		nodeHash := make([]byte, len(hash))
		copy(nodeHash, hash)

		recorder.record(nodeHash, nodeData)

		decoded, err := decodeBytes(nodeData)
		if err != nil {
			return nil, err
		}

		switch currNode := decoded.(type) {
		case nil:
			return nil, nil

		case *leaf:
			if bytes.Equal(currNode.key, partial) {
				return currNode.value, nil
			}
			return nil, nil

		case *branch:
			switch len(partial) {
			case 0:
				return currNode.value, nil
			default:
				if !bytes.HasPrefix(partial, currNode.key) {
					return nil, nil
				}

				if bytes.Equal(partial, currNode.key) {
					return currNode.value, nil
				}

				length := lenCommonPrefix(currNode.key, partial)
				switch child := currNode.children[partial[length]].(type) {
				case nil:
					return nil, nil
				default:
					partial = partial[length+1:]
					copy(hash, child.getHash())
				}
			}
		}
	}
}
