package common

import (
	"math/big"
	scale "github.com/ChainSafe/gossamer/codec"
)

type BlockHeader struct {
	parentHash [32]byte
	number *big.Int
	stateRoot [32]byte
	extrinsicsRoot [32]byte
	digest []byte
}

func (h *BlockHeader) Hash() ([32]byte, error) {
	encHeader, err := scale.Encode(h)
	if err != nil {
		return [32]byte{}, err
	}
	return Hash(encHeader)
} 
