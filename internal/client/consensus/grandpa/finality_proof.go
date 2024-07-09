// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"

	"github.com/ChainSafe/gossamer/internal/client/api"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// GRANDPA block finality proof generation and check.
//
// Finality of block B is proved by providing:
// 1) the justification for the descendant block F;
// 2) headers sub-chain (B; F] if B != F;
// 3) proof of GRANDPA::authorities() if the set changes at block F.
//
// Since earliest possible justification is returned, the GRANDPA authorities set
// at the block F is guaranteed to be the same as in the block B (this is because block
// that enacts new GRANDPA authorities set always comes with justification). It also
// means that the `set_id` is the same at blocks B and F.
//
// Let U be the last finalized block known to caller. If authorities set has changed several
// times in the (U; F] interval, multiple finality proof fragments are returned (one for each
// authority set change) and they must be verified in-order.
//
// Finality proof provider can choose how to provide finality proof on its own. The incomplete
// finality proof (that finalises some block C that is ancestor of the B and descendant
// of the U) could be returned.

var (
	// The requested block has not yet been finalized
	errBlockNotYetFinalized = errors.New("block not yet finalized")
	// The requested block is not covered by authority set changes. Likely this means the block is
	// in the latest authority set, and the subscription API is more appropriate
	errBlockNotInAuthoritySetChanges = errors.New("block not covered by authority set changes")
)

const maxUnknownHeaders = 100_000

// FinalityProofProvider Finality proof provider for serving network requests.
type FinalityProofProvider[
	Hash runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[Hash],
] struct {
	backend            api.Backend[Hash, N, Hasher]
	sharedAuthoritySet *SharedAuthoritySet[Hash, N]
}

// NewFinalityProofProvider Create new finality proof provider using:
//
// - backend for accessing blockchain data;
// - authorityProvider for calling and proving runtime methods.
// - sharedAuthoritySet for accessing authority set data
func NewFinalityProofProvider[
	Hash runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[Hash],
](
	backend api.Backend[Hash, N, Hasher],
	sharedAuthSet *SharedAuthoritySet[Hash, N],
) *FinalityProofProvider[Hash, N, Hasher] {
	return &FinalityProofProvider[Hash, N, Hasher]{
		backend:            backend,
		sharedAuthoritySet: sharedAuthSet,
	}
}

// ProveFinality Prove finality for the given block number by returning a Justification for the last block of
// the authority set in bytes.
func (provider FinalityProofProvider[Hash, N, Hasher]) ProveFinality(block N) (*[]byte, error) {
	proof, err := provider.proveFinalityProof(block, true)
	if err != nil {
		return nil, err
	}

	if proof != nil {
		encodedProof, err := scale.Marshal(*proof)
		if err != nil {
			return nil, err
		}
		return &encodedProof, nil
	}

	return nil, nil
}

// Prove finality for the given block number by returning a Justification for the last block of
// the authority set.
//
// If `collectUnknownHeaders` is true, the finality proof will include all headers from the
// requested block until the block the justification refers to.
func (provider FinalityProofProvider[Hash, N, Hasher]) proveFinalityProof(
	block N,
	collectUnknownHeaders bool) (*FinalityProof[Hash, N], error) {
	if provider.sharedAuthoritySet == nil {
		return nil, nil
	}

	return proveFinality[Hash, N, Hasher](
		provider.backend,
		provider.sharedAuthoritySet.inner.AuthoritySetChanges,
		block,
		collectUnknownHeaders,
	)
}

// FinalityProof Finality for block B is proved by providing:
// 1) the justification for the descendant block F;
// 2) headers sub-chain (B; F] if B != F;
type FinalityProof[Hash runtime.Hash, N runtime.Number] struct {
	// The hash of block F for which justification is provided
	Block Hash
	// Justification of the block F
	Justification []byte
	// The set of headers in the range (B; F] that we believe are unknown to the caller. Ordered.
	UnknownHeaders []runtime.Header[N, Hash]
}

// Prove finality for the given block number by returning a justification for the last block of
// the authority set of which the given block is part of, or a justification for the latest
// finalized block if the given block is part of the current authority set.
//
// If `collectUnknownHeaders` is true, the finality proof will include all headers from the
// requested block until the block the justification refers to.
func proveFinality[
	Hash runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[Hash],
](
	backend api.Backend[Hash, N, Hasher],
	authSetChanges AuthoritySetChanges[N],
	block N,
	collectUnknownHeaders bool,
) (*FinalityProof[Hash, N], error) {
	// Early-return if we are sure that there are no blocks finalized that cover the requested
	// block.
	finalizedNumber := backend.Blockchain().Info().FinalizedNumber
	if finalizedNumber < block {
		logger.Tracef("requested finality proof for descendant of %v while we only have finalized %v", block, finalizedNumber)
		return nil, errBlockNotYetFinalized
	}

	authSetChangeID, err := authSetChanges.getSetID(block)
	if err != nil {
		return nil, err
	}

	var encJustification []byte
	var justBlock N

	switch val := authSetChangeID.(type) {
	case authoritySetChangeIDLatest:
		justification, err := BestJustification[Hash, N, Hasher](backend)
		if err != nil && !errors.Is(err, errValueNotFound) {
			return nil, err
		}

		if justification != nil {
			encJustification, err = scale.Marshal(*justification)
			if err != nil {
				return nil, err
			}
			justBlock = justification.Target().Number
		} else {
			logger.Trace("No justification found for the authoritySetChangeIDLatest finalized block. Returning empty proof")
			return nil, nil
		}
	case authoritySetChangeIDSet[N]:
		lastBlockForSetID, err := backend.Blockchain().ExpectBlockHashFromID(val.inner.BlockNumber)
		if err != nil {
			return nil, err
		}

		// If error or no justifications found, return empty proof
		justifications, err := backend.Blockchain().Justifications(lastBlockForSetID)
		if err != nil || justifications == nil {
			logger.Tracef("getting justifications when making finality proof for %v. Returning empty proof",
				block)
			return nil, nil //nolint
		}
		justification := justifications.IntoJustification(primitives.GrandpaEngineID)
		if justification != nil {
			encJustification = *justification
			justBlock = val.inner.BlockNumber
		} else {
			logger.Tracef("No justification found when making finality proof for %v. Returning empty proof",
				block)
			return nil, nil
		}
	case authoritySetChangeIDUnknown:
		logger.Tracef("authoritySetChanges does not cover the requested block %v due to missing data."+
			" You need to resync to populate AuthoritySetChanges properly", block)

		return nil, errBlockNotInAuthoritySetChanges
	default:
		panic("authoritySetChangeIDUnknown type for authSetChangeID")
	}

	var headers []runtime.Header[N, Hash]
	if collectUnknownHeaders {
		// Collect all headers from the requested block until the last block of the set
		current := block + 1
		for {
			if current > justBlock || len(headers) >= maxUnknownHeaders {
				break
			}
			hash, err := backend.Blockchain().ExpectBlockHashFromID(current)
			if err != nil {
				return nil, err
			}

			header, err := backend.Blockchain().ExpectHeader(hash)
			if err != nil {
				return nil, err
			}
			headers = append(headers, header)
			current += 1
		}
	}

	blockHash, err := backend.Blockchain().ExpectBlockHashFromID(justBlock)
	if err != nil {
		return nil, err
	}

	return &FinalityProof[Hash, N]{
		Block:          blockHash,
		Justification:  encJustification,
		UnknownHeaders: headers,
	}, nil
}
