package trie

import (
	"bytes"
)

// findAndRecord search for a desired key recording all the nodes in the path including the desired node
func findAndRecord(t *Trie, key []byte, recorder *recorder) error {
	return find(t.root, key, recorder)
}

func find(parent node, key []byte, recorder *recorder) error {
	enc, hash, err := parent.encodeAndHash()
	if err != nil {
		return err
	}
	recorder.record(hash, enc)

	switch p := parent.(type) {
	case *branch:
		length := lenCommonPrefix(p.key, key)

		// found the value at this node
		if bytes.Equal(p.key, key) || len(key) == 0 {
			return nil
		}

		// did not find value
		if bytes.Equal(p.key[:length], key) && len(key) < len(p.key) {
			return nil
		}

		return find(p.children[key[length]], key[length+1:], recorder)
	case *leaf:
		if bytes.Equal(p.key, key) {
			return nil
		}
	}

	return nil
}
