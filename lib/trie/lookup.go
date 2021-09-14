package trie

import (
	"bytes"
	"errors"
	"fmt"

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
	return &Lookup{
		db:   db,
		hash: h,
	}
}

// Find will return the desired value or nil if key cannot be found and will record visited nodes
func (l *Lookup) Find(key []byte) ([]byte, []NodeRecord, error) {
	partial := key
	hash := l.hash

	nr := make([]NodeRecord, 0)

	for {
		nodeData, err := l.db.Get(hash[:])
		if err != nil {
			return nil, nil, ErrInvalidStateRoot
		}

		nodeRecord := NodeRecord{Hash: hash, RawData: nodeData}
		fmt.Println(len(nr))
		nr = append(nr, nodeRecord)

		decoded, err := decodeBytes(nodeData)
		if err != nil {
			return nil, nr, err
		}

		switch currNode := decoded.(type) {
		case nil:
			return nil, nr, nil

		case *leaf:
			if bytes.Equal(currNode.key, partial) {
				return currNode.value, nr, nil
			}
			return nil, nr, nil

		case *branch:
			switch len(partial) {
			case 0:
				return currNode.value, nr, nil
			default:
				if !bytes.HasPrefix(partial, currNode.key) {
					return nil, nr, nil
				}

				if bytes.Equal(partial, currNode.key) {
					return currNode.value, nr, nil
				}

				length := lenCommonPrefix(currNode.key, partial)
				switch child := currNode.children[partial[length]].(type) {
				case nil:
					return nil, nr, nil
				default:
					partial = partial[length+1:]
					copy(hash[:], child.getHash())
				}
			}
		}
	}
}
