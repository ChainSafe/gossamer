package trie

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/chaindb"
)

var (
	// ErrProofNodeNotFound when a needed proof node is not in the database
	ErrProofNodeNotFound = errors.New("cannot found the state root on storage")
)

// Lookup struct holds the state root and database reference
// used to retrieve trie information from database
type Lookup struct {
	// root to start the lookup
	root []byte
	db   chaindb.Database
}

// NewLookup returns a Lookup to helps the proof generator
func NewLookup(h []byte, db chaindb.Database) *Lookup {
	lk := &Lookup{db: db}
	lk.root = make([]byte, len(h))
	copy(lk.root, h)

	return lk
}

// Find will return the desired value or nil if key cannot be found and will record visited nodes
func (l *Lookup) Find(key []byte, recorder *Recorder) ([]byte, error) {
	partial := key
	hash := l.root

	for {
		nodeData, err := l.db.Get(hash[:])
		if err != nil {
			return nil, ErrProofNodeNotFound
		}

		nodeHash := make([]byte, len(hash))
		copy(nodeHash, hash)

		recorder.Record(nodeHash, nodeData)

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
					copy(hash[:], child.getHash())
				}
			}
		}
	}
}
