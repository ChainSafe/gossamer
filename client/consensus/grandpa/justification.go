// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"reflect"

	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

var (
	errInvalidAuthoritiesSet    = errors.New("current state of blockchain has invalid authorities set")
	errBadJustification         = errors.New("bad justification for header")
	errBlockNotDescendentOfBase = errors.New("block not descendent of base")
)

// Justification A GRANDPA justification for block finality, it includes a commit message and
// an ancestry proof including all headers routing all precommit target blocks
// to the commit target block. Due to the current voting strategy the precommit
// targets should be the same as the commit target, since honest voters don't
// vote past authority set change blocks.
//
// This is meant to be stored in the db and passed around the network to other
// nodes, and are used by syncing nodes to prove authority set handoffs.
type Justification[
	Hash constraints.Ordered,
	N constraints.Unsigned,
	S comparable,
	ID AuthorityID,
	H Header[Hash, N]] struct {
	Round           uint64
	Commit          finalityGrandpa.Commit[Hash, N, S, ID]
	VotesAncestries []H
}

// NewJustificationFromCommit Create a GRANDPA justification from the given commit. This method
// assumes the commit is valid and well-formed.
func NewJustificationFromCommit[
	Hash constraints.Ordered,
	N constraints.Unsigned,
	S comparable,
	ID AuthorityID,
	H Header[Hash, N]](
	client HeaderBackend[Hash, N, H],
	round uint64,
	commit finalityGrandpa.Commit[Hash, N, S, ID]) (Justification[Hash, N, S, ID, H], error) {
	votesAncestriesHashes := make(map[Hash]struct{})
	voteAncestries := make([]H, 0)

	// we pick the precommit for the lowest block as the base that
	// should serve as the root block for populating ancestry (i.e.
	// collect all headers from all precommit blocks to the base)
	var minPrecommit *hashNumber[Hash, N]
	for _, signed := range commit.Precommits {
		precommit := signed.Precommit
		if minPrecommit == nil {
			minPrecommit = &hashNumber[Hash, N]{
				hash:   precommit.TargetHash,
				number: precommit.TargetNumber,
			}
		} else if precommit.TargetNumber < minPrecommit.number {
			minPrecommit = &hashNumber[Hash, N]{
				hash:   precommit.TargetHash,
				number: precommit.TargetNumber,
			}
		}
	}
	if minPrecommit == nil {
		return Justification[Hash, N, S, ID, H]{},
			fmt.Errorf("%w: invalid precommits for target commit", errBadJustification)
	}

	baseNumber := minPrecommit.number
	baseHash := minPrecommit.hash
	for _, signed := range commit.Precommits {
		currentHash := signed.Precommit.TargetHash
		for {
			if currentHash == baseHash {
				break
			}

			header, err := client.Header(currentHash)
			if err != nil || header == nil {
				return Justification[Hash, N, S, ID, H]{},
					fmt.Errorf("%w: invalid precommits for target commit", errBadJustification)
			}

			currentHeader := *header

			// NOTE: this should never happen as we pick the lowest block
			// as base and only traverse backwards from the other blocks
			// in the commit. but better be safe to avoid an unbound loop.
			if currentHeader.Number() <= baseNumber {
				return Justification[Hash, N, S, ID, H]{},
					fmt.Errorf("%w: invalid precommits for target commit", errBadJustification)
			}
			parentHash := currentHeader.ParentHash()

			_, ok := votesAncestriesHashes[currentHash]
			if !ok {
				voteAncestries = append(voteAncestries, currentHeader)
			}

			votesAncestriesHashes[currentHash] = struct{}{}
			currentHash = parentHash
		}
	}

	return Justification[Hash, N, S, ID, H]{
		Round:           round,
		Commit:          commit,
		VotesAncestries: voteAncestries,
	}, nil
}

// Decode a GRANDPA justification and validate the commit and the votes'
// ancestry proofs finalize the given block.
func decodeAndVerifyFinalizes[Hash constraints.Ordered,
	N constraints.Unsigned,
	S comparable,
	ID AuthorityID,
	H Header[Hash, N]](
	encoded []byte,
	finalizedTarget hashNumber[Hash, N],
	setID uint64,
	voters finalityGrandpa.VoterSet[ID]) (Justification[Hash, N, S, ID, H], error) {
	justification := Justification[Hash, N, S, ID, H]{
		VotesAncestries: make([]H, 0),
	}
	err := scale.Unmarshal(encoded, &justification)
	if err != nil {
		return Justification[Hash, N, S, ID, H]{}, fmt.Errorf("error decoding justification for header: %s", err)
	}

	decodedTarget := hashNumber[Hash, N]{
		hash:   justification.Commit.TargetHash,
		number: justification.Commit.TargetNumber,
	}

	if decodedTarget != finalizedTarget {
		return Justification[Hash, N, S, ID, H]{}, fmt.Errorf("invalid commit target in grandpa justification")
	}

	return justification, justification.verifyWithVoterSet(setID, voters)
}

// Verify Validate the commit and the votes' ancestry proofs.
func (j *Justification[Hash, N, S, ID, H]) Verify(setID uint64, authorities AuthorityList[ID]) error {
	var weights []finalityGrandpa.IDWeight[ID]
	for _, authority := range authorities {
		weight := finalityGrandpa.IDWeight[ID]{
			ID:     authority.Key,
			Weight: finalityGrandpa.VoterWeight(authority.Weight),
		}
		weights = append(weights, weight)
	}

	voters := finalityGrandpa.NewVoterSet[ID](weights)
	if voters != nil {
		err := j.verifyWithVoterSet(setID, *voters)
		return err
	}
	return fmt.Errorf("%w", errInvalidAuthoritiesSet)
}

// Validate the commit and the votes' ancestry proofs.
func (j *Justification[Hash, N, S, ID, H]) verifyWithVoterSet(
	setID uint64,
	voters finalityGrandpa.VoterSet[ID]) error {
	ancestryChain := newAncestryChain[Hash, N](j.VotesAncestries)
	commitValidationResult, err := finalityGrandpa.ValidateCommit[Hash, N, S, ID](j.Commit, voters, ancestryChain)
	if err != nil {
		return fmt.Errorf("%w: invalid commit in grandpa justification", errBadJustification)
	}

	if !commitValidationResult.Valid() {
		return fmt.Errorf("%w: invalid commit in grandpa justification", errBadJustification)
	}

	// we pick the precommit for the lowest block as the base that
	// should serve as the root block for populating ancestry (i.e.
	// collect all headers from all precommit blocks to the base)
	precommits := j.Commit.Precommits
	var minPrecommit *finalityGrandpa.SignedPrecommit[Hash, N, S, ID]
	if len(precommits) == 0 {
		panic("can only fail if precommits is empty; commit has been validated above; " +
			"valid commits must include precommits")
	}
	for _, precommit := range precommits {
		currPrecommit := precommit
		if minPrecommit == nil {
			minPrecommit = &currPrecommit
		} else if currPrecommit.Precommit.TargetNumber <= minPrecommit.Precommit.TargetNumber {
			minPrecommit = &currPrecommit
		}
	}

	baseHash := minPrecommit.Precommit.TargetHash
	visitedHashes := make(map[Hash]struct{})
	for _, signed := range precommits {
		mgs := finalityGrandpa.Message[Hash, N]{Value: signed.Precommit}
		isValidSignature, err := checkMessageSignature[Hash, N, ID](mgs, signed.ID, signed.Signature, j.Round, setID)
		if err != nil {
			return err
		}

		if !isValidSignature {
			return fmt.Errorf("%w: invalid signature for precommit in grandpa justification",
				errBadJustification)
		}

		if baseHash == signed.Precommit.TargetHash {
			continue
		}

		route, err := ancestryChain.Ancestry(baseHash, signed.Precommit.TargetHash)
		if err != nil {
			return fmt.Errorf("%w: invalid precommit ancestry proof in grandpa justification",
				errBadJustification)
		}

		// ancestry starts from parent HashField but the precommit target HashField has been
		// visited
		visitedHashes[signed.Precommit.TargetHash] = struct{}{}
		for _, hash := range route {
			visitedHashes[hash] = struct{}{}
		}
	}

	ancestryHashes := make(map[Hash]struct{})
	for _, header := range j.VotesAncestries {
		hash := header.Hash()
		ancestryHashes[hash] = struct{}{}
	}

	if len(visitedHashes) != len(ancestryHashes) {
		return fmt.Errorf("%w: invalid precommit ancestries in grandpa justification with unused headers",
			errBadJustification)
	}

	// Check if maps are equal
	if !reflect.DeepEqual(ancestryHashes, visitedHashes) {
		return fmt.Errorf("%w: invalid precommit ancestries in grandpa justification with unused headers",
			errBadJustification)
	}

	return nil
}

// Target The target block NumberField and HashField that this justifications proves finality for
func (j *Justification[Hash, N, S, ID, H]) Target() hashNumber[Hash, N] {
	return hashNumber[Hash, N]{
		number: j.Commit.TargetNumber,
		hash:   j.Commit.TargetHash,
	}
}

// ancestryChain A utility trait implementing `finality_grandpa::Chain` using a given set of headers.
// This is useful when validating commits, using the given set of headers to
// verify a valid ancestry route to the target commit block.
type ancestryChain[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N]] struct {
	ancestry map[Hash]H
}

func newAncestryChain[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N]](
	headers []H) ancestryChain[Hash, N, H] {
	ancestry := make(map[Hash]H)
	for _, header := range headers {
		hash := header.Hash()
		ancestry[hash] = header
	}
	return ancestryChain[Hash, N, H]{
		ancestry: ancestry,
	}
}

func (ac ancestryChain[Hash, N, H]) Ancestry(base Hash, block Hash) ([]Hash, error) {
	route := make([]Hash, 0)
	currentHash := block

	for {
		if currentHash == base {
			break
		}

		br, ok := ac.ancestry[currentHash]
		if !ok {
			return nil, fmt.Errorf("%w", errBlockNotDescendentOfBase)
		}
		block = br.ParentHash()
		currentHash = block
		route = append(route, currentHash)
	}

	if len(route) != 0 {
		route = route[:len(route)-1]
	}
	return route, nil
}

func (ac ancestryChain[Hash, N, H]) IsEqualOrDescendantOf(base Hash, block Hash) bool {
	if base == block {
		return true
	}

	_, err := ac.Ancestry(base, block)
	return err == nil
}
