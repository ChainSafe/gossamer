package trie

import (
	"bytes"
	"errors"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	ErrInvalidStateRoot   = errors.New("cannot found the state root on storage")
	ErrIncompleteDatabase = errors.New("cannot found the node hash on storage")
)

type Lookup struct {
	recorder *NodeRecorder
	// root to start the lookup
	hash common.Hash
	db   chaindb.Database
}

// NewLookup returns a Lookup to helps the proof generator
func NewLookup(h common.Hash, db chaindb.Database, r *NodeRecorder) *Lookup {
	return &Lookup{
		db:       db,
		hash:     h,
		recorder: r,
	}
}

// Find will return the desired value or nil if key cannot be found and will record visited nodes
func (l *Lookup) Find(nKeys []byte) ([]byte, error) {
	partial := nKeys
	hash := l.hash

	var depth uint32

	for {
		nodeData, err := l.db.Get(hash[:])
		if err != nil && depth == 0 {
			return nil, ErrInvalidStateRoot
		} else if err != nil {
			return nil, ErrIncompleteDatabase
		}

		l.recorder.Record(hash, nodeData, depth)

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

				switch child := currNode.children[partial[0]].(type) {
				case nil:
					return nil, nil
				default:
					partial = partial[1:]
					copy(hash[:], child.getHash())
				}
			}
		}

		depth++
	}
}
