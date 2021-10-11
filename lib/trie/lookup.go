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

func findAndRecord(t *Trie, key []byte, recorder *recorder) []byte {
	l, err := find(t.root, key, recorder)
	if l == nil || err != nil {
		return nil
	}

	return l.value
}

func find(parent node, key []byte, recorder *recorder) (*leaf, error) {
	enc, hash, err := parent.encodeAndHash()
	if err != nil {
		return nil, err
	}

	recorder.record(hash, enc)

	switch p := parent.(type) {
	case *branch:
		length := lenCommonPrefix(p.key, key)

		// found the value at this node
		if bytes.Equal(p.key, key) || len(key) == 0 {
			return &leaf{key: p.key, value: p.value, dirty: false}, nil
		}

		// did not find value
		if bytes.Equal(p.key[:length], key) && len(key) < len(p.key) {
			return nil, nil
		}

		return find(p.children[key[length]], key[length+1:], recorder)
	case *leaf:
		enc, hash, err := p.encodeAndHash()
		if err != nil {
			return nil, err
		}

		recorder.record(hash, enc)
		if bytes.Equal(p.key, key) {
			return p, nil
		}
	default:
		return nil, nil
	}

	return nil, nil
}
