package trie

import (
	"hash"

	"golang.org/x/crypto/blake2s"
)

type Hasher struct {
	hash hash.Hash
}

func newHasher() (*Hasher, error) {
	h, err := blake2s.New256(nil)
	if err != nil {
		return nil, err
	}

	return &Hasher{
		hash: h,
	}, nil
}

// Hash encodes the node and then hashes it if its encoded length is > 32 bytes
func Hash(n node) (h []byte, err error) {
	hasher, err := newHasher()
	if err != nil {
		return nil, err
	}

	encNode, err := n.Encode()
	if err != nil {
		return nil, err
	}

	// if length of encoded leaf is less than 32 bytes, do not hash
	if len(encNode) < 32 {
		return encNode, nil
	}

	// otherwise, hash encoded node
	_, err = hasher.hash.Write(encNode)
	if err == nil {
		h = hasher.hash.Sum(nil)
	}

	return h, err
}
