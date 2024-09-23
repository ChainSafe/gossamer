package grandpa

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const MAX_WARP_SYNC_PROOF_SIZE = 8 * 1024 * 1024

type WarpSyncFragment struct {
	// The last block that the given authority set finalized. This block should contain a digest
	// signalling an authority set change from which we can fetch the next authority set.
	header types.Header
	// A justification for the header above which proves its finality. In order to validate it the
	// verifier must be aware of the authorities and set id for which the justification refers to.
	justification []byte
}

type WarpSyncProof struct {
	proofs []WarpSyncFragment
	// indicates whether the warp sync has been completed
	isFinished bool
}

type NetworkProvider struct {
	blockState   BlockState
	grandpaState GrandpaState
}

func (np *NetworkProvider) Generate(start common.Hash) ([]byte, error) {
	//Generate proof
	beginBlockHeader, err := np.blockState.GetHeader(start)
	if err != nil {
		return nil, err
	}
	authoritySetChanges, err := np.grandpaState.GetAuthoritesChangesFromBlock(beginBlockHeader.Number)
	if err != nil {
		return nil, err
	}

	proofsEncodedLen := 0
	proofs := make([]WarpSyncFragment, 0)
	limitReached := false
	for _, blockNumber := range authoritySetChanges {
		// the header should contains a standard scheduled change
		// otherwise  the set must have changed through a forced changed,
		// in which case we stop collecting proofs as the chain of trust in authority handoffs was broken.
		header, err := np.blockState.GetHeaderByNumber(blockNumber)
		if err != nil {
			return nil, err
		}

		justification, err := np.blockState.GetJustification(header.Hash()) // get the justification of such block
		if err != nil {
			return nil, err
		}
		fragment := WarpSyncFragment{header: *header, justification: justification}

		// check the proof size
		encodedFragment, err := scale.Marshal(fragment)
		if err != nil {
			return nil, err
		}

		if proofsEncodedLen+len(encodedFragment) >= MAX_WARP_SYNC_PROOF_SIZE {
			limitReached = true
			break
		}

		proofsEncodedLen += len(encodedFragment)
		proofs = append(proofs, fragment)
	}

	isFinished := false
	// If the limit is not reached then they retrieve the latest (best) justification
	// and append in the proofs
	if !limitReached {
		bestLastBlockHeader, err := np.blockState.BestBlockHeader()
		if err != nil {
			return nil, err
		}
		latestJustification, err := np.blockState.GetJustification(bestLastBlockHeader.Hash())
		if err != nil {
			return nil, err
		}

		fragment := WarpSyncFragment{header: *bestLastBlockHeader, justification: latestJustification}
		proofs = append(proofs, fragment)
		isFinished = true
	}

	//Encode proof
	finalProof := WarpSyncProof{proofs: proofs, isFinished: isFinished}
	encodedProof, err := scale.Marshal(finalProof)
	if err != nil {
		return nil, err
	}
	return encodedProof, nil
}
