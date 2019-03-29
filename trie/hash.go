package trie

import (
	"bytes"
	"encoding/hex"
	"errors"
	"hash"

	scale "github.com/ChainSafe/gossamer/codec"
	hexcodec "github.com/ChainSafe/gossamer/common/codec"
	"golang.org/x/crypto/blake2s"
)

type hasher struct {
	hash hash.Hash
}

func newHasher() (*hasher, error) {
	key, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	if err != nil {
		return nil, err
	}

	h, err := blake2s.New256(key)
	if err != nil {
		return nil, err
	}

	return &hasher{
		hash: h,
	}, nil
}

// Encode encodes the node with its respective type encoding
func Encode(n node) ([]byte, error) {
	switch n := n.(type) {
	case *leaf:
		return n.Encode()
	case *extension:
		return n.Encode()
	case *branch:
		return n.Encode()
	default:
		return nil, errors.New("cannot encode: invalid node")
	}
}

// Encode encodes a leaf node
func (l *leaf) Encode() ([]byte, error) {
	encLen, err := encodeLen(l)
	if err != nil {
		return nil, err
	}

	encHex := keyToHex(l.key)
	encHex = hexcodec.Encode(encHex[0 : len(encHex)-1])

	buf := bytes.Buffer{}
	encoder := &scale.Encoder{&buf}
	_, err = encoder.Encode(l.value)
	if err != nil {
		return nil, err
	}

	return append(append(encLen, encHex...), buf.Bytes()...), nil
}

// Encode encodes a branch node
func (b *branch) Encode() (h []byte, err error) {
	return nil, nil
}

// Encode encodes an extension node
func (e *extension) Encode() (h []byte, err error) {
	return nil, nil
}

// encodeLen encodes the length of the partial key an extension or leaf node
func encodeLen(n node) (encLen []byte, err error) {
	switch n := n.(type) {
	case *extension:
		encLen = []byte{getPrefix(n)}
		if len(n.key) < bigKeySize(n) {
			encLen = append(encLen, byte(len(n.key)))
		} else {
			encLen = append(encLen, []byte{127, byte(len(n.key) - bigKeySize(n))}...)
		}
	case *leaf:
		encLen = []byte{getPrefix(n)}
		if len(n.key) < bigKeySize(n) {
			encLen = append(encLen, byte(len(n.key)))
		} else {
			encLen = append(encLen, []byte{127, byte(len(n.key) - bigKeySize(n))}...)
		}
	default:
		err = errors.New("encodeLen error: invalid node")
	}

	return encLen, err
}

func (l *leaf) Hash() (h []byte, err error) {
	hasher, err := newHasher()
	if err != nil {
		return nil, err
	}

	encLeaf, err := l.Encode()
	if err != nil {
		return nil, err
	}

	// if length of encoded leaf is less than 32 bytes, do not hash
	if len(encLeaf) < 32 {
		return encLeaf, nil
	}

	// otherwise, hash encoded node
	_, err = hasher.hash.Write(encLeaf)
	if err != nil {
		return nil, err
	}

	return hasher.hash.Sum(nil), nil
}

func (b *branch) Hash() (h []byte, err error) {
	return nil, nil
}

func (e *extension) Hash() (h []byte, err error) {
	return nil, nil
}
