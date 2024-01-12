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

func (k KeccakHash) Bytes() []byte {
	return k.bytes[:]
}

func (k KeccakHash) ComparableKey() string {
	return string(k.Bytes())
}

func KeccakHashFromBytes(b []byte) KeccakHash {
	var newBytes [KeccakHasherLength]byte
	copy(newBytes[:], b)
	return KeccakHash{
		bytes: newBytes,
	}
}

var _ hashdb.HashOut = KeccakHash{}

type KeccakHasher struct{}

func (k KeccakHasher) Length() int {
	return KeccakHasherLength
}

func (k KeccakHasher) FromBytes(in []byte) KeccakHash {
	var buf = [KeccakHasherLength]byte{}
	copy(buf[:], in)
	return NewKeccakHash(buf)
}

func (k KeccakHasher) Hash(in []byte) KeccakHash {
	h := sha3.NewLegacyKeccak256()

	_, err := h.Write(in)
	if err != nil {
		panic("Unexpected error hashing bytes")
	}

	hash := h.Sum(nil)
	return k.FromBytes(hash)
}

func NewKeccakHasher() KeccakHasher {
	return KeccakHasher{}
}

var _ hashdb.Hasher[KeccakHash] = (*KeccakHasher)(nil)
