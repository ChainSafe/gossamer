// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const MAX_WARP_SYNC_PROOF_SIZE = 8 * 1024 * 1024

var (
	errMissingStartBlock      = fmt.Errorf("missing start block")
	errStartBlockNotFinalized = fmt.Errorf("start block is not finalized")
)

type BlockState interface {
	GetHeader(hash common.Hash) (*types.Header, error)
	BestBlockHeader() (*types.Header, error)
	GetHeaderByNumber(num uint) (*types.Header, error)
	GetJustification(hash common.Hash) ([]byte, error)
}

type GrandpaState interface {
	GetAuthoritesChangesFromBlock(blockNumber uint) ([]uint, error)
}

type WarpSyncFragment struct {
	// The last block that the given authority set finalized. This block should contain a digest
	// signalling an authority set change from which we can fetch the next authority set.
	Header types.Header
	// A justification for the header above which proves its finality. In order to validate it the
	// verifier must be aware of the authorities and set id for which the justification refers to.
	Justification GrandpaJustification[hash.H256, uint64]
}

type WarpSyncProof struct {
	Proofs []WarpSyncFragment
	// indicates whether the warp sync has been completed
	IsFinished bool
}

func (w *WarpSyncProof) Encode() ([]byte, error) {
	if w == nil {
		return nil, fmt.Errorf("cannot encode nil WarpSyncProof")
	}
	return scale.Marshal(*w)
}

type WarpSyncProofProvider struct {
	blockState   BlockState
	grandpaState GrandpaState
}

// Generate build a warp sync encoded proof starting from the given block hash
func (np *WarpSyncProofProvider) Generate(start common.Hash) ([]byte, error) {
	// Get and traverse all GRANDPA authorities changes from the given block hash
	beginBlockHeader, err := np.blockState.GetHeader(start)
	if err != nil {
		return nil, errMissingStartBlock
	}

	bestLastBlockHeader, err := np.blockState.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	if beginBlockHeader.Number > bestLastBlockHeader.Number {
		return nil, errStartBlockNotFinalized
	}

	authoritySetChanges, err := np.grandpaState.GetAuthoritesChangesFromBlock(beginBlockHeader.Number)
	if err != nil {
		return nil, err
	}

	proofsEncodedLen := 0
	proofs := make([]WarpSyncFragment, 0)
	limitReached := false
	for _, blockNumber := range authoritySetChanges {
		header, err := np.blockState.GetHeaderByNumber(blockNumber)
		if err != nil {
			return nil, err
		}

		encJustification, err := np.blockState.GetJustification(header.Hash()) // get the justification of such block
		if err != nil {
			return nil, err
		}

		justification, err := decodeJustification[hash.H256, uint64, runtime.BlakeTwo256](encJustification)
		if err != nil {
			return nil, err
		}

		fragment := WarpSyncFragment{Header: *header, Justification: *justification}

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
	// If the limit is not reached then retrieve the latest (best) justification
	// and append in the proofs
	if !limitReached {
		// the existing best justification must be for a block higher than the
		// last authority set change. if we didn't prove any authority set
		// change then we fallback to make sure it's higher or equal to the
		// initial warp sync block.
		bestLastBlockHeader, err := np.blockState.BestBlockHeader()
		if err != nil {
			return nil, err
		}
		latestJustification, err := np.blockState.GetJustification(bestLastBlockHeader.Hash())
		if err != nil {
			return nil, err
		}

		justification, err := decodeJustification[hash.H256, uint64, runtime.BlakeTwo256](latestJustification)
		if err != nil {
			return nil, err
		}

		limit := proofs[len(proofs)-1].Justification.Justification.Commit.TargetNumber + 1

		if justification.Justification.Commit.TargetNumber >= limit {
			fragment := WarpSyncFragment{Header: *bestLastBlockHeader, Justification: *justification}
			proofs = append(proofs, fragment)
		}

		isFinished = true
	}

	// Encode and return the proof
	finalProof := WarpSyncProof{Proofs: proofs, IsFinished: isFinished}
	return finalProof.Encode()
}
