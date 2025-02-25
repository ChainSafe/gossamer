package provisioner

import (
	"fmt"

	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance parachain.RuntimeInstance, err error)
}

// GetOnchainDisputes gets the on-chain disputes at a given block number and returns them as a map
// for efficient searching. It takes a relay parent hash and returns a map of session index and
// candidate hash tuples to dispute states.
func GetOnchainDisputes(
	blockstate BlockState,
	relayParent common.Hash,
) (map[parachaintypes.DisputeKey]parachaintypes.DisputeState, error) {
	rt, err := blockstate.GetRuntime(relayParent)
	if err != nil {
		return nil, fmt.Errorf("getting runtime for relay parent %s: %w", relayParent, err)
	}

	disputes, err := rt.ParachainHostDisputes()
	if err != nil {
		return nil, fmt.Errorf("getting disputes from runtime: %w", err)
	}

	return disputes, nil
}
