package util

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

// NOTE: In Polkadot-SDK, chainAPI is a parachain subsystem which communicates via overseer messages.

// FetchAncestorBlockHashes fetches the hashes of the ancestors of a given leaf hash
// up to a specified number of blocks.
func FetchAncestorBlockHashes(
	leafHash common.Hash,
	numOfBlocks uint32,
	blockState BlockState,
) ([]common.Hash, error) {
	ancestors := make([]common.Hash, 0, numOfBlocks)
	currentHash := leafHash

	for i := uint32(0); i < numOfBlocks; i++ {
		header, err := blockState.GetHeader(currentHash)
		if err != nil {
			return nil, fmt.Errorf("getting header for leaf ancestor %v: %w", currentHash, err)
		}

		// Stop at genesis block
		if header.Number == 0 {
			break
		}

		parentHash := header.ParentHash
		ancestors = append(ancestors, parentHash)
		currentHash = parentHash
	}

	return ancestors, nil
}
