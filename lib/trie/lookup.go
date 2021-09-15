package trie

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/chaindb"
)

var (
	ErrInvalidStateRoot = errors.New("cannot found the state root on storage")
)

type Lookup struct {
	// root to start the lookup
	hash []byte
	db   chaindb.Database
}

// NewLookup returns a Lookup to helps the proof generator
func NewLookup(h []byte, db chaindb.Database) *Lookup {
	lk := &Lookup{db: db}
	lk.hash = make([]byte, len(h))
	copy(lk.hash, h)

	return lk
}

// Find will return the desired value or nil if key cannot be found and will record visited nodes
func (l *Lookup) Find(key []byte, recorder *Recorder) ([]byte, error) {
	partial := key
	hash := l.hash

	for {
		nodeData, err := l.db.Get(hash[:])
		if err != nil {
			return nil, ErrInvalidStateRoot
		}

		recorder.Record(hash, nodeData)

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
