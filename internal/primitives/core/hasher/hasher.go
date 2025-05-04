package hasher

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/crypto/hashing"
)

type Blake2Hasher struct{}

func (bt256 Blake2Hasher) Hash(s []byte) hash.H256 {
	h := hashing.BlakeTwo256(s)
	return hash.H256(h[:])
}
