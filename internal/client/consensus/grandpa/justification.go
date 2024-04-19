// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	pgrandpa "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	errInvalidAuthoritiesSet    = errors.New("current state of blockchain has invalid authorities set")
	errBadJustification         = errors.New("bad justification for header")
	errBlockNotDescendentOfBase = errors.New("block not descendent of base")
)

// A GRANDPA justification for block finality, it includes a commit message and
// an ancestry proof including all headers routing all precommit target blocks
// to the commit target block. Due to the current voting strategy the precommit
// targets should be the same as the commit target, since honest voters don't
// vote past authority set change blocks.
//
// This is meant to be stored in the db and passed around the network to other
// nodes, and are used by syncing nodes to prove authority set handoffs.
// #[derive(Clone, Encode, Decode, PartialEq, Eq, Debug)]
// pub struct GrandpaJustification<Block: BlockT> {
type GrandpaJustification[Hash runtime.Hash, N runtime.Number] struct {
	// The GRANDPA justification for block finality.
	Justification pgrandpa.GrandpaJustification[Hash, N]
}

// Type used for decoding grandpa justifications (can pass in generic Header type)
type decodeGrandpaJustification[
	Hash runtime.Hash,
	N runtime.Number,
] GrandpaJustification[Hash, N]

func decodeJustification[
	Hash runtime.Hash,
	N runtime.Number,
](encodedJustification []byte) (*GrandpaJustification[Hash, N], error) {
	newJustificaiton := decodeGrandpaJustification[Hash, N]{}
	err := scale.Unmarshal(encodedJustification, &newJustificaiton)
	if err != nil {
		return nil, err
	}
	return newJustificaiton.GrandpaJustification(), nil
}

func (dgj *decodeGrandpaJustification[Ordered, N]) UnmarshalSCALE(reader io.Reader) (err error) {
	type roundCommitHeader struct {
		Round   uint64
		Commit  pgrandpa.Commit[Ordered, N]
		Headers []runtime.Header[N, Ordered]
	}
	rch := roundCommitHeader{}
	decoder := scale.NewDecoder(reader)
	err = decoder.Decode(&rch)
	if err != nil {
		return
	}

	dgj.Justification.Round = rch.Round
	dgj.Justification.Commit = rch.Commit
	dgj.Justification.VoteAncestries = make([]runtime.Header[N, Ordered], len(rch.Headers))
	for i, header := range rch.Headers {
		dgj.Justification.VoteAncestries[i] = header
	}
	return
}

func (dgj decodeGrandpaJustification[Hash, N]) GrandpaJustification() *GrandpaJustification[Hash, N] {
	return &GrandpaJustification[Hash, N]{
		Justification: pgrandpa.GrandpaJustification[Hash, N]{
			Round:          dgj.Justification.Round,
			Commit:         dgj.Justification.Commit,
			VoteAncestries: dgj.Justification.VoteAncestries,
		},
	}
}

// NewJustificationFromCommit Create a GRANDPA justification from the given commit. This method
// assumes the commit is valid and well-formed.
func NewJustificationFromCommit[
	Hash runtime.Hash,
	N runtime.Number,
](
	client blockchain.HeaderBackend[Hash, N],
	round uint64,
	commit pgrandpa.Commit[Hash, N],
) (GrandpaJustification[Hash, N], error) {
	votesAncestriesHashes := make(map[Hash]struct{})
	voteAncestries := make([]runtime.Header[N, Hash], 0)

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
		return GrandpaJustification[Hash, N]{},
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
				return GrandpaJustification[Hash, N]{},
					fmt.Errorf("%w: invalid precommits for target commit", errBadJustification)
			}

			currentHeader := *header

			// NOTE: this should never happen as we pick the lowest block
			// as base and only traverse backwards from the other blocks
			// in the commit. but better be safe to avoid an unbound loop.
			if currentHeader.Number() <= baseNumber {
				return GrandpaJustification[Hash, N]{},
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

	return GrandpaJustification[Hash, N]{
		Justification: pgrandpa.GrandpaJustification[Hash, N]{
			Round:          round,
			Commit:         commit,
			VoteAncestries: voteAncestries,
		},
	}, nil
}

// Decode a GRANDPA justification and validate the commit and the votes'
// ancestry proofs finalize the given block.
func decodeAndVerifyFinalizes[
	Hash runtime.Hash,
	N runtime.Number,
	S comparable,
	ID pgrandpa.AuthorityID,
](
	encoded []byte,
	finalizedTarget hashNumber[Hash, N],
	setID uint64,
	voters grandpa.VoterSet[string],
) (GrandpaJustification[Hash, N], error) {
	justification, err := decodeJustification[Hash, N](encoded)
	if err != nil {
		return GrandpaJustification[Hash, N]{}, fmt.Errorf("error decoding justification for header: %s", err)
	}

	decodedTarget := hashNumber[Hash, N]{
		hash:   justification.Justification.Commit.TargetHash,
		number: justification.Justification.Commit.TargetNumber,
	}

	if decodedTarget != finalizedTarget {
		return GrandpaJustification[Hash, N]{}, fmt.Errorf("invalid commit target in grandpa justification")
	}

	return *justification, justification.verifyWithVoterSet(setID, voters)
}

// Verify Validate the commit and the votes' ancestry proofs.
func (j *GrandpaJustification[Hash, N]) Verify(setID uint64, authorities pgrandpa.AuthorityList) error {
	var weights []grandpa.IDWeight[string]
	for _, authority := range authorities {
		weight := grandpa.IDWeight[string]{
			ID:     string(authority.AuthorityID.ToRawVec()),
			Weight: uint64(authority.AuthorityWeight),
		}
		weights = append(weights, weight)
	}

	voters := grandpa.NewVoterSet[string](weights)
	if voters != nil {
		err := j.verifyWithVoterSet(setID, *voters)
		return err
	}
	return fmt.Errorf("%w", errInvalidAuthoritiesSet)
}

// Validate the commit and the votes' ancestry proofs.
func (j *GrandpaJustification[Hash, N]) verifyWithVoterSet(
	setID uint64,
	voters grandpa.VoterSet[string],
) error {
	ancestryChain := newAncestryChain[Hash, N](j.Justification.VoteAncestries)
	signedPrecommits := make([]grandpa.SignedPrecommit[Hash, N, string, string], 0)
	for _, pc := range j.Justification.Commit.Precommits {
		signedPrecommits = append(signedPrecommits, grandpa.SignedPrecommit[Hash, N, string, string]{
			Precommit: pc.Precommit,
			Signature: string(pc.Signature[:]),
			ID:        string(pc.ID.ToRawVec()),
		})
	}
	commitValidationResult, err := grandpa.ValidateCommit[Hash, N, string, string](
		grandpa.Commit[Hash, N, string, string]{
			TargetHash:   j.Justification.Commit.TargetHash,
			TargetNumber: j.Justification.Commit.TargetNumber,
			Precommits:   signedPrecommits,
		},
		voters,
		ancestryChain,
	)
	if err != nil {
		return fmt.Errorf("%w: invalid commit in grandpa justification", errBadJustification)
	}

	if !commitValidationResult.Valid() {
		return fmt.Errorf("%w: invalid commit in grandpa justification", errBadJustification)
	}

	// we pick the precommit for the lowest block as the base that
	// should serve as the root block for populating ancestry (i.e.
	// collect all headers from all precommit blocks to the base)
	precommits := j.Justification.Commit.Precommits
	var minPrecommit *grandpa.SignedPrecommit[Hash, N, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]
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
		msg := grandpa.NewMessage(signed.Precommit)
		isValidSignature := pgrandpa.CheckMessageSignature[Hash, N](
			msg,
			signed.ID,
			signed.Signature,
			pgrandpa.RoundNumber(j.Justification.Round),
			pgrandpa.SetID(setID),
		)

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
	for _, header := range j.Justification.VoteAncestries {
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
func (j *GrandpaJustification[Hash, N]) Target() hashNumber[Hash, N] {
	return hashNumber[Hash, N]{
		number: j.Justification.Commit.TargetNumber,
		hash:   j.Justification.Commit.TargetHash,
	}
}

// ancestryChain A utility trait implementing `finality_grandpa::Chain` using a given set of headers.
// This is useful when validating commits, using the given set of headers to
// verify a valid ancestry route to the target commit block.
type ancestryChain[Hash runtime.Hash, N runtime.Number] struct {
	ancestry map[Hash]runtime.Header[N, Hash]
}

func newAncestryChain[Hash runtime.Hash, N runtime.Number](
	headers []runtime.Header[N, Hash],
) ancestryChain[Hash, N] {
	ancestry := make(map[Hash]runtime.Header[N, Hash])
	for _, header := range headers {
		hash := header.Hash()
		ancestry[hash] = header
	}
	return ancestryChain[Hash, N]{
		ancestry: ancestry,
	}
}

func (ac ancestryChain[Ordered, N]) Ancestry(base Ordered, block Ordered) ([]Ordered, error) {
	route := make([]Ordered, 0)
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

func (ac ancestryChain[Ordered, N]) IsEqualOrDescendantOf(base Ordered, block Ordered) bool {
	if base == block {
		return true
	}

	_, err := ac.Ancestry(base, block)
	return err == nil
}
