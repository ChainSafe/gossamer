package common

import (
	"math/big"
)

// Hash used to store a blake2b hash
type Hash [32]byte

// BlockHeader is the header of a Polkadot block
type BlockHeader struct {
	ParentHash     Hash     // the block hash of the block's parent
	Number         *big.Int // block number
	StateRoot      Hash     // the root of the state trie
	ExtrinsicsRoot Hash     // the root of the extrinsics trie
	Digest         []byte   // any addition block info eg. logs, seal
}

func NewHash(in []byte) (res Hash) {
	res = [32]byte{}
	copy(res[:], in)
	return res
}

func (h Hash) ToBytes() []byte {
	b := [32]byte(h)
	return b[:]
}