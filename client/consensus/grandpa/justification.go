// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
	"reflect"
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
type Justification[H HashI, N constraints.Unsigned, S comparable, ID constraints.Ordered, Header HeaderI[H, N]] struct {
	Round           uint64
	Commit          finalityGrandpa.Commit[H, N, S, ID]
	VotesAncestries []Header
}

// Create a GRANDPA justification from the given commit. This method
// assumes the commit is valid and well-formed.
func fromCommit[H HashI, N constraints.Unsigned, S comparable, ID constraints.Ordered, Header HeaderI[H, N]](client HeaderBackend[H, N, Header], round uint64, commit finalityGrandpa.Commit[H, N, S, ID]) (Justification[H, N, S, ID, Header], error) {
	votesAncestriesHashes := make(map[H]struct{})
	voteAncestries := make([]Header, 0)

	// we pick the precommit for the lowest block as the base that
	// should serve as the root block for populating ancestry (i.e.
	// collect all headers from all precommit blocks to the base)
	var minPrecommit *hashNumber[H, N]
	for _, signed := range commit.Precommits {
		precommit := signed.Precommit
		if minPrecommit == nil {
			minPrecommit = &hashNumber[H, N]{
				hash:   precommit.TargetHash,
				number: precommit.TargetNumber,
			}
		} else if precommit.TargetNumber < minPrecommit.number {
			minPrecommit = &hashNumber[H, N]{
				hash:   precommit.TargetHash,
				number: precommit.TargetNumber,
			}
		}
	}
	if minPrecommit == nil {
		return Justification[H, N, S, ID, Header]{}, fmt.Errorf("%w: invalid precommits for target commit", errBadJustification)
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
				return Justification[H, N, S, ID, Header]{}, fmt.Errorf("%w: invalid precommits for target commit", errBadJustification)
			}

			currentHeader := *header

			// NOTE: this should never happen as we pick the lowest block
			// as base and only traverse backwards from the other blocks
			// in the commit. but better be safe to avoid an unbound loop.
			if currentHeader.Number() <= baseNumber {
				return Justification[H, N, S, ID, Header]{}, fmt.Errorf("%w: invalid precommits for target commit", errBadJustification)
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

	return Justification[H, N, S, ID, Header]{
		Round:           round,
		Commit:          commit,
		VotesAncestries: voteAncestries,
	}, nil
}

// Decode a GRANDPA justification and validate the commit and the votes'
// ancestry proofs finalize the given block.
func decodeAndVerifyFinalizes[H HashI, N constraints.Unsigned, S comparable, ID constraints.Ordered, Header HeaderI[H, N]](encoded []byte, finalizedTarget hashNumber[H, N], setID uint64, voters finalityGrandpa.VoterSet[ID]) (Justification[H, N, S, ID, Header], error) {
	justification := Justification[H, N, S, ID, Header]{
		VotesAncestries: make([]Header, 0),
	}
	err := scale.Unmarshal(encoded, &justification)
	if err != nil {
		return Justification[H, N, S, ID, Header]{}, fmt.Errorf("error decoding justification for header: %s", err)
	}

	decodedTarget := hashNumber[H, N]{
		hash:   justification.Commit.TargetHash,
		number: justification.Commit.TargetNumber,
	}

	if decodedTarget != finalizedTarget {
		return Justification[H, N, S, ID, Header]{}, fmt.Errorf("invalid commit target in grandpa justification")
	}

	return justification, justification.verifyWithVoterSet(setID, voters)
}

// TODO get feedback on if I can avoid this, see below
//type NewAuthority[ID constraints.Ordered] struct {
//	Key    ID
//	Weight uint64
//}

// Validate the commit and the votes' ancestry proofs.
func (j *Justification[H, N, S, ID, Header]) verify(setID uint64, weights []finalityGrandpa.IDWeight[ID]) error {
	// TODO Get reviewer feedback on this. In substrate they pass in data then convert to IDWeight, however for
	// us we do no ever use this input type, so I think better to just directly take IDWeights as a param.
	// Can revert if people disagree
	//var weights []finalityGrandpa.IDWeight[ID]
	//for _, authority := range authorities {
	//	weight := finalityGrandpa.IDWeight[ID]{
	//		ID: authority.Key,
	//	}
	//	weights = append(weights, weight)
	//}
	voters := finalityGrandpa.NewVoterSet[ID](weights)
	if voters != nil {
		err := j.verifyWithVoterSet(setID, *voters)
		return err
	}
	return fmt.Errorf("%w", errInvalidAuthoritiesSet)
}

// Validate the commit and the votes' ancestry proofs.
func (j *Justification[H, N, S, ID, Header]) verifyWithVoterSet(setID uint64, voters finalityGrandpa.VoterSet[ID]) error {
	ancestryChain := newAncestryChain[H, N](j.VotesAncestries)
	commitValidationResult, err := finalityGrandpa.ValidateCommit[H, N, S, ID](j.Commit, voters, ancestryChain)
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
	//minPrecommit := &finalityGrandpa.SignedPrecommit[H, N, S, ID]{}
	var minPrecommit *finalityGrandpa.SignedPrecommit[H, N, S, ID]
	if len(precommits) == 0 {
		panic("can only fail if precommits is empty; commit has been validated above; valid commits must include precommits; qed.")
	}
	for _, precommit := range precommits {
		if minPrecommit == nil {
			minPrecommit = &precommit
		} else if precommit.Precommit.TargetNumber <= minPrecommit.Precommit.TargetNumber {
			minPrecommit = &precommit
		}
	}

	baseHash := minPrecommit.Precommit.TargetHash
	visitedHashes := make(map[H]struct{})
	for _, signed := range precommits {
		/*
				TODO signature is generic type any, but signature verification in Gossamer uses []byte
			    TODO need access to message.value both for encoding and so I can set it
		*/
		//if !checkMessageSignature[H, N](signed.Precommit, signed.ID, signed.Signature, j.Round, setID) {
		//	return fmt.Errorf("%w: invalid signature for precommit in grandpa justification", errBadJustification)
		//}

		if baseHash == signed.Precommit.TargetHash {
			continue
		}

		route, err := ancestryChain.Ancestry(baseHash, signed.Precommit.TargetHash)
		if err != nil {
			return fmt.Errorf("%w: invalid precommit ancestry proof in grandpa justification", errBadJustification)
		}

		// ancestry starts from parent HashField but the precommit target HashField has been
		// visited
		visitedHashes[signed.Precommit.TargetHash] = struct{}{}
		for _, hash := range route {
			visitedHashes[hash] = struct{}{}
		}
	}

	ancestryHashes := make(map[H]struct{})
	for _, header := range j.VotesAncestries {
		hash := header.Hash()
		ancestryHashes[hash] = struct{}{}
	}

	if len(visitedHashes) != len(ancestryHashes) {
		return fmt.Errorf("%w: invalid precommit ancestries in grandpa justification with unused headers", errBadJustification)
	}

	// Check if maps are equal
	if !reflect.DeepEqual(ancestryHashes, visitedHashes) {
		return fmt.Errorf("%w: invalid precommit ancestries in grandpa justification with unused headers", errBadJustification)
	}

	return nil
}

// The target block NumberField and HashField that this justifications proves finality for
func (j *Justification[H, N, S, ID, Header]) target() hashNumber[H, N] {
	return hashNumber[H, N]{
		number: j.Commit.TargetNumber,
		hash:   j.Commit.TargetHash,
	}
}

// ancestryChain A utility trait implementing `finality_grandpa::Chain` using a given set of headers.
// This is useful when validating commits, using the given set of headers to
// verify a valid ancestry route to the target commit block.
type ancestryChain[H HashI, N constraints.Unsigned, Header HeaderI[H, N]] struct {
	ancestry map[H]Header
}

func newAncestryChain[H HashI, N constraints.Unsigned, Header HeaderI[H, N]](headers []Header) ancestryChain[H, N, Header] {
	ancestry := make(map[H]Header)
	for _, header := range headers {
		hash := header.Hash()
		ancestry[hash] = header
	}
	return ancestryChain[H, N, Header]{
		ancestry: ancestry,
	}
}

func (ac ancestryChain[H, N, Header]) Ancestry(base H, block H) ([]H, error) {
	route := make([]H, 0)
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

		if !block.IsEmpty() {
			currentHash = block
			route = append(route, currentHash)
		} else {
			return nil, fmt.Errorf("%w", errBlockNotDescendentOfBase)
		}
	}

	if len(route) != 0 {
		route = route[:len(route)-1]
	}
	return route, nil
}

func (ac ancestryChain[H, N, Header]) IsEqualOrDescendantOf(base H, block H) bool {
	if base == block {
		return true
	}

	_, err := ac.Ancestry(base, block)
	return err == nil
}
