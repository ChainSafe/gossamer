package db

import (
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
)

type noopExtrinsic struct{}

func (noopExtrinsic) IsSigned() *bool {
	return nil
}

// Check for interface fulfillment
var (
	_ blockchain.HeaderBackend[hash.H256, uint]  = &BlockchainDB[hash.H256, uint, noopExtrinsic, *generic.Header[uint, hash.H256, runtime.BlakeTwo256]]{}
	_ blockchain.HeaderMetaData[hash.H256, uint] = &BlockchainDB[hash.H256, uint, noopExtrinsic, *generic.Header[uint, hash.H256, runtime.BlakeTwo256]]{}
	_ blockchain.Backend[hash.H256, uint]        = &BlockchainDB[hash.H256, uint, noopExtrinsic, *generic.Header[uint, hash.H256, runtime.BlakeTwo256]]{}
)
