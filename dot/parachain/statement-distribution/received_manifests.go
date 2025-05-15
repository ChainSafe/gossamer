// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"errors"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	errManifestImportConflicting = errors.New("the manifest conflicts with another, previously sent manifest")

	errManifestImportOverflow = errors.New(
		"the manifest has overflowed beyond the limits of what the counterparty was allowed to send us",
	)
)

// manifestSummary represents a summary of a manifest being sent by a counterparty.
type manifestSummary struct {
	// claimedParentHash is the claimed parent head data hash of the candidate.
	claimedParentHash common.Hash
	// claimedGroupIndex is the claimed group index assigned to the candidate.
	claimedGroupIndex parachaintypes.GroupIndex
	// statementKnowledge is a statement filter sent alongside the candidate, communicating knowledge.
	statementKnowledge statementFilter
}

// receivedManifests contains the knowledge we are aware of counterparties having of manifests.
type receivedManifests struct {
	received map[parachaintypes.CandidateHash]manifestSummary

	// secondedCounts is a limit of how many seconded statement
	// a given candidate can have, defined per session
	secondedCounts map[parachaintypes.GroupIndex][]uint
}

func (rm *receivedManifests) candidateStatementFilter(candidateHash parachaintypes.CandidateHash) *statementFilter {
	if rm.received == nil {
		return nil
	}

	manifestSummary, ok := rm.received[candidateHash]
	if !ok {
		return nil
	}

	filter := manifestSummary.statementKnowledge.clone()
	return &filter
}

// importReceived attempts to import a received manifest from a counterparty.
//
// This will reject manifests which are either duplicate, conflicting,
// or imply an irrational amount of `Seconded` statements.
//
// This assumes that the manifest has already been checked for
// validity - i.e. that the bitvecs match the claimed group in size
// and that the manifest includes at least one `Seconded`
// attestation and includes enough attestations for the candidate
// to be backed.
//
// This also should only be invoked when we are intended to track
// the knowledge of this peer as determined by the [SessionTopology].
func (rm *receivedManifests) importReceived(
	groupSize uint,
	secondingLimit uint,
	candidateHash parachaintypes.CandidateHash,
	manifestSummary manifestSummary,
) error {
	previousSummary, ok := rm.received[candidateHash]
	if !ok {
		withinLimits := updatingEnsureWithinSecondingLimit(
			rm.secondedCounts,
			manifestSummary.claimedGroupIndex,
			groupSize,
			secondingLimit,
			manifestSummary.statementKnowledge.secondedInGroup,
		)

		if withinLimits {
			rm.received[candidateHash] = manifestSummary
			return nil
		} else {
			return errManifestImportOverflow
		}
	}

	if previousSummary.claimedGroupIndex != manifestSummary.claimedGroupIndex {
		return errManifestImportConflicting
	}

	if !manifestSummary.statementKnowledge.secondedInGroup.Contains(
		previousSummary.statementKnowledge.secondedInGroup,
	) {
		return errManifestImportConflicting
	}

	if !manifestSummary.statementKnowledge.validatedInGroup.Contains(
		previousSummary.statementKnowledge.validatedInGroup,
	) {
		return errManifestImportConflicting
	}

	freshSeconded := manifestSummary.statementKnowledge.secondedInGroup.Or(
		previousSummary.statementKnowledge.secondedInGroup,
	)

	withinLimits := updatingEnsureWithinSecondingLimit(
		rm.secondedCounts,
		manifestSummary.claimedGroupIndex,
		groupSize,
		secondingLimit,
		freshSeconded,
	)

	if !withinLimits {
		return errManifestImportOverflow
	}

	// All checks passed. Overwrite: guaranteed to be superset.
	rm.received[candidateHash] = manifestSummary
	return nil
}

// updatingEnsureWithinSecondingLimit updates validator-seconded records but only
// if the new statements are OK. returns `true` if alright and `false` otherwise.
//
// The seconding limit is a per-validator limit. It ensures an upper bound on the total number of
// candidates entering the system.
//
// The function mutates secondedCounts.
func updatingEnsureWithinSecondingLimit(
	secondedCounts map[parachaintypes.GroupIndex][]uint,
	groupIndex parachaintypes.GroupIndex,
	groupSize uint,
	secondingLimit uint,
	newSeconded parachaintypes.BitVec,
) bool {
	if secondingLimit == 0 {
		return false
	}

	counts, ok := secondedCounts[groupIndex]
	if !ok {
		counts = make([]uint, groupSize)
		secondedCounts[groupIndex] = counts
	}

	nsBits := newSeconded.Bits()
	for i, bit := range nsBits {
		if !bit {
			continue
		}

		if i < len(counts) && counts[i] == secondingLimit {
			return false
		}
	}

	for i, bit := range nsBits {
		if !bit {
			continue
		}

		if i < len(counts) {
			counts[i] += 1
		} else {
			// polkadot-sdk does not contain this case and assumes groupSize == len(newSeconded)
			// https://github.com/paritytech/polkadot-sdk/blob/3b4c48e7e3bba96091024643407994938607e3b9/polkadot/node/network/statement-distribution/src/v2/grid.rs#L865
			logger.Warnf(
				"unexpectedly got more new seconded statements (%d) than members in group (%d)",
				len(nsBits),
				groupSize,
			)
		}
	}
	secondedCounts[groupIndex] = counts

	return true
}
