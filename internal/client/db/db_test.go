// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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

// Check for interface fulfilment
var (
	_ blockchain.HeaderBackend[hash.H256, uint] = &blockchainDB[
		hash.H256, uint, noopExtrinsic, *generic.Header[uint, hash.H256, runtime.BlakeTwo256]]{}
	_ blockchain.HeaderMetadata[hash.H256, uint] = &blockchainDB[
		hash.H256, uint, noopExtrinsic, *generic.Header[uint, hash.H256, runtime.BlakeTwo256]]{}
	_ blockchain.Backend[hash.H256, uint] = &blockchainDB[
		hash.H256, uint, noopExtrinsic, *generic.Header[uint, hash.H256, runtime.BlakeTwo256]]{}
)
