package client

import (
	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
)

type noopExtrinsic struct{}

func (noopExtrinsic) IsSigned() *bool {
	return nil
}

type TestClient struct {
	Client[
		hash.H256, runtime.BlakeTwo256, uint64, noopExtrinsic, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	]
}

var (
	_ api.BlockchainEvents[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]] = &TestClient{}
	_ api.PreCommitActions[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]] = &TestClient{}
)
