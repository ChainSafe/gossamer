package trie

import (
	"bytes"
)

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
