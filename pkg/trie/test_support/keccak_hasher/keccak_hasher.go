package keccak_hasher

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"golang.org/x/crypto/sha3"
)

const KeccakHasherLength = 32

type KeccakHash struct {
	bytes [KeccakHasherLength]byte
}

func NewKeccakHash(bytes [KeccakHasherLength]byte) KeccakHash {
	return KeccakHash{
		bytes: bytes,
	}
}

func (k KeccakHash) ToBytes() []byte {
	return k.bytes[:]
}

type KeccakHasher[H KeccakHash] struct{}

func (k *KeccakHasher[H]) Length() int {
	return KeccakHasherLength
}

func (k *KeccakHasher[H]) FromBytes(in []byte) H {
	var buf = [KeccakHasherLength]byte{}
	copy(buf[:], in)
	return H(NewKeccakHash(buf))
}

func (k *KeccakHasher[H]) Hash(in []byte) H {
	h := sha3.NewLegacyKeccak256()

	_, err := h.Write(in)
	if err != nil {
		panic("Unexpected error hashing bytes")
	}

	hash := h.Sum(nil)
	return k.FromBytes(hash)
}

func NewKeccakHasher[H KeccakHash]() KeccakHasher[H] {
	return KeccakHasher[H]{}
}

var _ hashdb.Hasher[KeccakHash] = (*KeccakHasher[KeccakHash])(nil)
