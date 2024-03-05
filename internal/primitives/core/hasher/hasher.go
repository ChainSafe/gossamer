package hasher

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hashing"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// / Blake2-256 Hash implementation.
type BlakeTwo256 struct{}

// / Produce the hash of some byte-slice.
func (bt256 BlakeTwo256) Hash(s []byte) hash.H256 {
	h := hashing.Blake2_256(s)
	return hash.H256(h[:])
}

// / Produce the hash of some codec-encodable value.
func (bt256 BlakeTwo256) HashOf(s any) hash.H256 {
	bytes := scale.MustMarshal(s)
	return bt256.Hash(bytes)
}

// / Blake2-256 Hash implementation.
type Keccak256 struct{}

// / Produce the hash of some byte-slice.
func (k256 Keccak256) Hash(s []byte) hash.H256 {
	h := hashing.Keccak256(s)
	return hash.H256(h[:])
}

// / Produce the hash of some codec-encodable value.
func (k256 Keccak256) HashOf(s any) hash.H256 {
	bytes := scale.MustMarshal(s)
	return k256.Hash(bytes)
}
