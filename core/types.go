package core

import (
	"math/big"

	"github.com/ChainSafe/gossamer/common"
)

// TODO: Unsure if this differs from the block defined with the header in common/types.go
type Block struct {
	SlotNumber		*big.Int
	PreviousHash	common.Hash
	//VrfOutput		VRFOutput
	//Transactions 	[]Transaction
	//Signature		Signature
	BlockNumber		*big.Int
	Hash			common.Hash
}