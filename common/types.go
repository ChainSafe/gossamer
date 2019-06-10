package common

import (
	"math/big"
)

// Hash used to store a blake2b hash
type Hash [32]byte

// BlockHeader is the header of a Polkadot block
// ParentHash is the block hash of the block's parent
// Number is the block number
// StateRoot is the root of the state trie
// ExtrinsicsRoot is the root of the extrinsics trie
// Digest is any addition block info eg. logs
type BlockHeader struct {
	ParentHash     Hash
	Number         *big.Int
	StateRoot      Hash
	ExtrinsicsRoot Hash
	Digest         []byte
}
