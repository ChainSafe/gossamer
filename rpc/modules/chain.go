package modules

import (
	"math/big"

	"github.com/ChainSafe/gossamer/common"
)

type ChainHashRequest common.Hash

type ChainBlockNumberRequest *big.Int

// TODO: Waiting on Block type https://github.com/ChainSafe/gossamer/pull/233
type ChainBlockResponse struct {}

type ChainBlockHeaderResponse struct{}

type ChainBlockHashResponse common.Hash