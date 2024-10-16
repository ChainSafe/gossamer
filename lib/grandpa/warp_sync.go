// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	consensus_grandpa "github.com/ChainSafe/gossamer/internal/client/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa/app"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const MaxWarpSyncProofSize = 8 * 1024 * 1024

var (
	errMissingStartBlock      = fmt.Errorf("missing start block")
	errStartBlockNotFinalized = fmt.Errorf("start block is not finalized")
)

type WarpSyncFragment struct {
	// The last block that the given authority set finalized. This block should contain a digest
	// signalling an authority set change from which we can fetch the next authority set.
	Header types.Header
	// A justification for the header above which proves its finality. In order to validate it the
	// verifier must be aware of the authorities and set id for which the justification refers to.
	Justification consensus_grandpa.GrandpaJustification[hash.H256, uint64]
}

type WarpSyncProof struct {
	Proofs []WarpSyncFragment
	// indicates whether the warp sync has been completed
	IsFinished   bool
	proofsLength int
}

func NewWarpSyncProof() WarpSyncProof {
	return WarpSyncProof{
		Proofs:       make([]WarpSyncFragment, 0),
		IsFinished:   false,
		proofsLength: 0,
	}
}

func (w *WarpSyncProof) addFragment(fragment WarpSyncFragment) (limitReached bool, err error) {
	encodedFragment, err := scale.Marshal(fragment)
	if err != nil {
		return false, err
	}

	if w.proofsLength+len(encodedFragment) >= MaxWarpSyncProofSize {
		return true, nil
	}

	w.proofsLength += len(encodedFragment)
	w.Proofs = append(w.Proofs, fragment)

	return false, nil
}

func (w *WarpSyncProof) lastProofBlockNumber() uint64 {
	if len(w.Proofs) == 0 {
		return 0
	}
	return w.Proofs[len(w.Proofs)-1].Justification.Justification.Commit.TargetNumber + 1
}

// The warp proof is verified by traversing the warp proof fragments,
// then we verify the justifications against the authorities based on the
// genesis authorities and the scheduled changes.
// If we are able to verify all the fragments, then the warp proof is valid.
func (w *WarpSyncProof) verify(
	setId grandpa.SetID,
	authorities primitives.AuthorityList,
	hardForks map[string]SetIdAuthorityList,
) (*SetIdAuthorityList, error) {
	currentSetId := setId
	currentAuthorities := authorities

	for fragmentNumber, proof := range w.Proofs {
		headerHash := proof.Header.Hash()
		number := proof.Header.Number

		hardForkKey := fmt.Sprintf("%v-%v", headerHash, number)
		if fork, ok := hardForks[hardForkKey]; ok {
			currentSetId = fork.SetID
			currentAuthorities = fork.AuthorityList
		} else {
			err := proof.Justification.Verify(uint64(currentSetId), currentAuthorities)
			if err != nil {
				return nil, err
			}

			if !bytes.Equal(proof.Justification.Target().Hash.Bytes(), headerHash.ToBytes()) {
				return nil, fmt.Errorf("mismatch between header and justification")
			}

			scheduledChange, err := findScheduledChange(proof.Header)
			if err != nil {
				return nil, fmt.Errorf("finding scheduled change: %w", err)
			}

			if scheduledChange != nil {
				auths, err := grandpaAuthoritiesRawToAuthorities(scheduledChange.Auths)
				if err != nil {
					return nil, fmt.Errorf("cannot parse GRANPDA raw authorities: %w", err)
				}

				currentSetId += 1
				currentAuthorities = auths
			} else if fragmentNumber != len(w.Proofs)-1 || !w.IsFinished {
				return nil, fmt.Errorf("Header is missing authority set change digest")
			}
		}
	}

	return &SetIdAuthorityList{currentSetId, currentAuthorities}, nil
}

type WarpSyncProofProvider struct {
	blockState   BlockState
	grandpaState GrandpaState
	hardForks    map[string]SetIdAuthorityList
}

func NewWarpSyncProofProvider(blockState BlockState, grandpaState GrandpaState) *WarpSyncProofProvider {
	return &WarpSyncProofProvider{
		blockState:   blockState,
		grandpaState: grandpaState,
	}
}

type SetIdAuthorityList struct {
	grandpa.SetID
	primitives.AuthorityList
}

// Generate build a warp sync encoded proof starting from the given block hash
func (p *WarpSyncProofProvider) Generate(start common.Hash) ([]byte, error) {
	// Get and traverse all GRANDPA authorities changes from the given block hash
	beginBlockHeader, err := p.blockState.GetHeader(start)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errMissingStartBlock, err)
	}

	lastFinalizedBlockHeader, err := p.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return nil, fmt.Errorf("getting best block header: %w", err)
	}

	if beginBlockHeader.Number > lastFinalizedBlockHeader.Number {
		return nil, errStartBlockNotFinalized
	}

	authoritySetChanges, err := p.grandpaState.GetAuthoritiesChangesFromBlock(beginBlockHeader.Number)
	if err != nil {
		return nil, err
	}

	limitReached := false
	finalProof := NewWarpSyncProof()
	for _, blockNumber := range authoritySetChanges {
		header, err := p.blockState.GetHeaderByNumber(blockNumber)
		if err != nil {
			return nil, err
		}

		scheduledChange, err := findScheduledChange(*header)
		if err != nil {
			return nil, fmt.Errorf("finding scheduled change: %w", err)
		}

		// the last block in a set is the one that triggers a change to the next set,
		// therefore the block must have a digest that signals the authority set change
		if scheduledChange == nil {
			// if it doesn't contain a signal for standard change then the set must have changed
			// through a forced changed, in which case we stop collecting proofs as the chain of
			// trust in authority handoffs was broken.
			break
		}

		encJustification, err := p.blockState.GetJustification(header.Hash()) // get the justification of such block
		if err != nil {
			return nil, err
		}

		justification, err := consensus_grandpa.DecodeJustification[hash.H256, uint64, runtime.BlakeTwo256](encJustification)
		if err != nil {
			return nil, err
		}

		fragment := WarpSyncFragment{Header: *header, Justification: *justification}

		// check the proof size
		limitReached, err = finalProof.addFragment(fragment)
		if err != nil {
			return nil, err
		}

		if limitReached {
			break
		}
	}

	// If the limit is not reached then retrieve the latest (best) justification
	// and append in the proofs
	if !limitReached {
		// the existing best justification must be for a block higher than the
		// last authority set change. if we didn't prove any authority set
		// change then we fallback to make sure it's higher or equal to the
		// initial warp sync block.
		lastFinalizedBlockHeader, err := p.blockState.GetHighestFinalisedHeader()
		if err != nil {
			return nil, fmt.Errorf("getting best block header: %w", err)
		}

		latestJustification, err := p.blockState.GetJustification(lastFinalizedBlockHeader.Hash())
		if err != nil {
			return nil, err
		}

		justification, err := consensus_grandpa.DecodeJustification[hash.H256, uint64, runtime.BlakeTwo256](
			latestJustification,
		)
		if err != nil {
			return nil, err
		}

		if justification.Justification.Commit.TargetNumber >= finalProof.lastProofBlockNumber() {
			fragment := WarpSyncFragment{Header: *lastFinalizedBlockHeader, Justification: *justification}
			_, err = finalProof.addFragment(fragment)
			if err != nil {
				return nil, err
			}
		}

		finalProof.IsFinished = true
	}

	// Encode and return the proof
	return scale.Marshal(finalProof)
}

// Verify checks the validity of the given warp sync proof
func (p *WarpSyncProofProvider) Verify(
	encodedProof []byte,
	setId grandpa.SetID,
	authorities primitives.AuthorityList,
) (*network.WarpSyncVerificationResult, error) {
	var proof WarpSyncProof
	err := scale.Unmarshal(encodedProof, &proof)
	if err != nil {
		return nil, fmt.Errorf("decoding warp sync proof: %w", err)
	}

	if len(proof.Proofs) == 0 {
		return nil, fmt.Errorf("empty warp sync proof")
	}

	lastProof := proof.Proofs[len(proof.Proofs)-1]
	lastHeader := lastProof.Header

	nextSetAndAuthorities, err := proof.verify(setId, authorities, p.hardForks)
	if err != nil {
		return nil, fmt.Errorf("verifying warp sync proof: %w", err)
	}

	return &network.WarpSyncVerificationResult{
		SetId:         nextSetAndAuthorities.SetID,
		AuthorityList: nextSetAndAuthorities.AuthorityList,
		Header:        lastHeader,
		Completed:     proof.IsFinished,
	}, nil
}

func findScheduledChange(
	header types.Header,
) (*types.GrandpaScheduledChange, error) {
	for _, digestItem := range header.Digest {
		digestValue, err := digestItem.Value()
		if err != nil {
			return nil, fmt.Errorf("getting digest value: %w", err)
		}

		switch val := digestValue.(type) {
		case types.ConsensusDigest:
			consensusDigest := types.GrandpaConsensusDigest{}
			if val.ConsensusEngineID == types.GrandpaEngineID {
				err := scale.Unmarshal(val.Data, &consensusDigest)
				if err != nil {
					return nil, err
				}

				scheduledChange, err := consensusDigest.Value()
				if err != nil {
					return nil, err
				}

				parsedScheduledChange, _ := scheduledChange.(types.GrandpaScheduledChange)
				return &parsedScheduledChange, nil
			}
		}
	}
	return nil, nil
}

func grandpaAuthoritiesRawToAuthorities(adr []types.GrandpaAuthoritiesRaw) (primitives.AuthorityList, error) {
	ad := make([]primitives.AuthorityIDWeight, len(adr))
	for i, r := range adr {
		ad[i] = primitives.AuthorityIDWeight{}

		key, err := ed25519.NewPublicKey(r.Key[:])
		if err != nil {
			return nil, err
		}

		keyBytes := key.AsBytes()
		pkey, err := app.NewPublic(keyBytes[:])
		if err != nil {
			return nil, err
		}

		ad[i].AuthorityID = pkey
		ad[i].AuthorityWeight = primitives.AuthorityWeight(r.ID)
	}

	return ad, nil
}
