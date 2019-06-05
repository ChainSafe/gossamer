package common

import (
	"math/big"
	//log "github.com/inconshreveable/log15"
	//scale "github.com/ChainSafe/gossamer/codec"
)

type BlockHeader struct {
	ParentHash [32]byte
	Number *big.Int
	StateRoot [32]byte
	ExtrinsicsRoot [32]byte
	Digest []byte
}

// func (h *BlockHeader) Hash() ([32]byte, error) {
// 	encHeader, err := scale.Encode(*h)
// 	log.Debug("BlockHeader.Hash", "encHeader", encHeader)
// 	if err != nil {
// 		return [32]byte{}, err
// 	}
// 	return Hash(encHeader)
// } 
