package common

import (
	"fmt"
	"math/big"
)

// Hash used to store a blake2b hash
type Hash [32]byte

func (h *Hash) String() string {
	return fmt.Sprintf("0x%x", h[:])
}

// BlockHeader is the header of a Polkadot block
type BlockHeader struct {
	ParentHash     Hash     // the block hash of the block's parent
	Number         *big.Int // block number
	StateRoot      Hash     // the root of the state trie
	ExtrinsicsRoot Hash     // the root of the extrinsics trie
	Digest         []byte   // any addition block info eg. logs, seal
}

