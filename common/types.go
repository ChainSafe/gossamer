package common

import (
	"math/big"
)

type Hash [32]byte

type BlockHeader struct {
	ParentHash     Hash
	Number         *big.Int
	StateRoot      Hash
	ExtrinsicsRoot Hash
	Digest         []byte
}
