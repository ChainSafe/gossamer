// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package provisioner

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"time"

	disputemessages "github.com/ChainSafe/gossamer/dot/parachain/disputes-coordinator/messages"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/primitives"
)

const (
	MaxDisputeVotesForwardedToRuntime = 200_000
	VotesSelectionBatchSize           = 1_100
)

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance parachain.RuntimeInstance, err error)
}

// Implements the `select_disputes` function which selects dispute votes which should
// be sent to the Runtime.
//
// Provisioner fetches all disputes from `dispute-coordinator` and separates them in multiple
// partitions. Please refer to `struct PartitionedDisputes` for details.
//
// Besides the prioritisation described above the votes in each partition are filtered too.
// Provisioner fetches all onchain votes and filters them out from all partitions. As a result the
// Runtime receives only fresh votes
func SelectDisputes(
	overseerChan chan<- any,
	blockState BlockState,
	leaf *parachaintypes.ActivatedLeaf,
	maxDisputesVotes int,
	voteSelectionBatchSize int,
) parachaintypes.MultiDisputeStatementSet {
	onchain, err := getOnchainDisputes(blockState, leaf.Hash)
	if err != nil {
		logger.Warnf("getting on chain disputes: %s", err.Error())
		// Here we log the error and continue with an empty onchain set.
		onchain = make(map[parachaintypes.DisputeKey]parachaintypes.DisputeState)
	}

	recent, err := requestDisputes(overseerChan)
	if err != nil {
		recent = []disputemessages.RecentDispute{}
	}

	// Filter out unconfirmed disputes except if already known onchain.
	var filteredRecent []disputemessages.RecentDispute
	for _, d := range recent {
		if d.DisputeStatus.IsConfirmedConcluded() ||
			containsOnchain(onchain, d.SessionIndex, d.CandidateHash) {
			filteredRecent = append(filteredRecent, d)
		}
	}

	partitioned := partitionRecentDisputes(filteredRecent, onchain)
	voteResults, err := voteSelection(overseerChan, partitioned, onchain,
		maxDisputesVotes, voteSelectionBatchSize)
	if err != nil {
		voteResults = []voteSelectionResult{}
	}

	return newMultiDisputeStatementSet(voteResults)
}

type voteSelectionResult struct {
	key   parachaintypes.DisputeKey
	votes parachaintypes.CandidateVotes
}

// voteSelection selects dispute votes from PartitionedDisputes which should be sent to the runtime.
// Votes which are already onchain are filtered out. Result should be sorted by (SessionIndex, CandidateHash).
func voteSelection(
	overseerChan chan<- any,
	partitioned *partitionedDisputes,
	onchain map[parachaintypes.DisputeKey]parachaintypes.DisputeState,
	maxDisputesVotes int,
	voteSelectionBatchSize int,
) ([]voteSelectionResult, error) {
	// fetch in batches until there are enough votes
	disputes := partitioned.orderedPartitions()
	totalVotesLen := 0
	result := make(map[parachaintypes.DisputeKey]parachaintypes.CandidateVotes)
	requestVotesCounter := 0

	for len(disputes) > 0 {
		batchSize := min(voteSelectionBatchSize, len(disputes))
		batch := disputes[:batchSize]
		disputes = disputes[batchSize:]

		// Filter votes which are already onchain
		requestVotesCounter++
		candidatesVotes, err := requestVotes(overseerChan, batch)
		if err != nil {
			return nil, err
		}

		var selectedVotes []voteSelectionResult

		for _, candidateVote := range candidatesVotes {
			sessionIndex := candidateVote.SessionIndex
			candidateHash := candidateVote.CandidateHash
			votes := candidateVote.CandidateVotes

			key := parachaintypes.DisputeKey{
				SessionIndex:  sessionIndex,
				CandidateHash: candidateHash,
			}
			onchainState, ok := onchain[key]
			if !ok {
				// onchain knows nothing about this dispute - add all votes
				selectedVotes = append(selectedVotes,
					voteSelectionResult{
						key:   key,
						votes: votes,
					})
				continue
			}

			validatorIdxToRemove := make([]parachaintypes.ValidatorIndex, 0)
			votes.Valid.Ascend(
				parachaintypes.ValidatorIndex(0),
				func(validatorIdx parachaintypes.ValidatorIndex,
					vote parachaintypes.Vote[parachaintypes.ValidDisputeStatementKind],
				) bool {
					validDisputeStatement := &parachaintypes.DisputeStatement{}
					err := validDisputeStatement.SetValue(parachaintypes.ValidDisputeStatement{
						Kind: vote.Kind,
					})
					if err != nil {
						panic(
							fmt.Sprintf(
								"%T is an invalid variant of %T",
								vote.Kind,
								validDisputeStatement,
							),
						)
					}

					isVoteWorth := isVoteWorthToKeep(
						validatorIdx,
						*validDisputeStatement,
						onchainState,
					)
					if !isVoteWorth {
						validatorIdxToRemove = append(validatorIdxToRemove, validatorIdx)
					}
					return true
				},
			)

			for _, vi := range validatorIdxToRemove {
				votes.Valid.Delete(vi)
			}

			validatorIdxToRemove = make([]parachaintypes.ValidatorIndex, 0)
			votes.Invalid.Ascend(
				parachaintypes.ValidatorIndex(0),
				func(validatorIdx parachaintypes.ValidatorIndex,
					vote parachaintypes.Vote[parachaintypes.InvalidDisputeStatementKind],
				) bool {
					invalidDisputeStatement := &parachaintypes.DisputeStatement{}
					err := invalidDisputeStatement.SetValue(parachaintypes.InvalidDisputeStatement{
						Kind: vote.Kind,
					})
					if err != nil {
						panic(
							fmt.Sprintf(
								"%T is an valid variant of %T",
								vote.Kind,
								invalidDisputeStatement,
							),
						)
					}

					if !isVoteWorthToKeep(validatorIdx, *invalidDisputeStatement, onchainState) {
						validatorIdxToRemove = append(validatorIdxToRemove, validatorIdx)
					}
					return true
				},
			)

			for _, vi := range validatorIdxToRemove {
				votes.Invalid.Delete(vi)
			}

			selectedVotes = append(selectedVotes,
				voteSelectionResult{
					key: parachaintypes.DisputeKey{
						SessionIndex:  sessionIndex,
						CandidateHash: candidateHash,
					},
					votes: votes,
				})
		}

		// Check if votes are within the limit
		for _, vote := range selectedVotes {
			sessionIndex := vote.key.SessionIndex
			candidateHash := vote.key.CandidateHash
			selectedVotes := vote.votes

			votesLen := selectedVotes.Valid.Len() + selectedVotes.Invalid.Len()
			if votesLen+totalVotesLen > maxDisputesVotes {
				// we are done - no more votes can be added. Importantly, we don't add any votes for
				// a dispute here if we can't fit them all. This gives us an important invariant,
				// that backing votes for disputes make it into the provisioned vote set.

				return sortVoteSelectionResults(result), nil
			}

			key := parachaintypes.DisputeKey{
				SessionIndex:  sessionIndex,
				CandidateHash: candidateHash,
			}
			result[key] = selectedVotes
			totalVotesLen += votesLen
		}
	}

	return sortVoteSelectionResults(result), nil
}

// sortVoteSelectionResults sorts the vote selection results based on SessionIndex and CandidateHash.
func sortVoteSelectionResults(
	votes map[parachaintypes.DisputeKey]parachaintypes.CandidateVotes,
) []voteSelectionResult {
	var sortedResults []voteSelectionResult
	for key, vote := range votes {
		sortedResults = append(sortedResults, voteSelectionResult{key: key, votes: vote})
	}

	slices.SortFunc(sortedResults, func(i, j voteSelectionResult) int {
		switch {
		case i.key.SessionIndex < j.key.SessionIndex:
			return -1
		case i.key.SessionIndex > j.key.SessionIndex:
			return +1
		default:
			return bytes.Compare(i.key.CandidateHash.Value[:], j.key.CandidateHash.Value[:])
		}
	})

	return sortedResults
}

// partitionedDisputes contains disputes by partitions.
type partitionedDisputes struct {
	// Concluded and inactive disputes which are completely unknown for the Runtime.
	// Hopefully this should never happen. Will be sent to the Runtime with FIRST priority.
	inactiveUnknownOnchain []parachaintypes.DisputeKey

	// Disputes which are INACTIVE locally but they are unconcluded for the Runtime.
	// A dispute can have enough local vote to conclude and at the same time the
	// Runtime knows nothing about them at treats it as unconcluded. This discrepancy
	// should be treated with high priority.
	// Will be sent to the Runtime with SECOND priority.
	inactiveUnconcludedOnchain []parachaintypes.DisputeKey

	// Active disputes completely unknown onchain.
	// Will be sent to the Runtime with THIRD priority.
	activeUnknownOnchain []parachaintypes.DisputeKey

	// Active disputes unconcluded onchain.
	// Will be sent to the Runtime with FOURTH priority.
	activeUnconcludedOnchain []parachaintypes.DisputeKey

	// Active disputes concluded onchain. New votes are not that important for this partition.
	// Will be sent to the Runtime with FIFTH priority.
	activeConcludedOnchain []parachaintypes.DisputeKey

	// Inactive disputes which has concluded onchain. These are not interesting and
	// won't be sent to the Runtime.
	// Will be DROPPED
	inactiveConcludedOnchain []parachaintypes.DisputeKey
}

// orderedPartitions returns an array of partitions in the order they should be processed.
func (pd partitionedDisputes) orderedPartitions() []parachaintypes.DisputeKey {
	seqToIterate := [][]parachaintypes.DisputeKey{
		pd.inactiveUnknownOnchain,
		pd.inactiveUnconcludedOnchain,
		pd.activeUnknownOnchain,
		pd.activeUnconcludedOnchain,
		pd.activeConcludedOnchain,
		// pd.InactiveConcludedOnchain is dropped on purpose
	}

	var out []parachaintypes.DisputeKey
	for _, seq := range seqToIterate {
		out = append(out, seq...)
	}
	return out
}

func concludedOnchain(onchainState *parachaintypes.DisputeState) bool {
	n := onchainState.ValidatorsFor.Len()
	supermajority := n - (primitives.SaturatingSub(n, 1) / 3)

	return onchainState.ValidatorsFor.CountOnes() >= supermajority ||
		onchainState.ValidatorsAgainst.CountOnes() >= supermajority
}

func partitionRecentDisputes(
	recent []disputemessages.RecentDispute,
	onchain map[parachaintypes.DisputeKey]parachaintypes.DisputeState,
) *partitionedDisputes {
	partitioned := &partitionedDisputes{}

	// Drop any duplicates
	uniqueRecent := make(map[parachaintypes.DisputeKey]parachaintypes.DisputeStatus)
	for _, r := range recent {
		uniqueRecent[parachaintypes.DisputeKey{
			SessionIndex:  r.SessionIndex,
			CandidateHash: r.CandidateHash,
		}] = r.DisputeStatus
	}

	// Split recent disputes in ACTIVE and INACTIVE
	timeNow := uint64(time.Now().Unix())
	active := make(map[parachaintypes.DisputeKey]struct{})
	inactive := make(map[parachaintypes.DisputeKey]struct{})
	for k, v := range uniqueRecent {
		if !parachaintypes.DisputeIsInactive(&v, timeNow) {
			active[k] = struct{}{}
		} else {
			inactive[k] = struct{}{}
		}
	}

	// Split ACTIVE in three groups...
	for key := range active {
		if d, ok := onchain[key]; ok {
			if concludedOnchain(&d) {
				partitioned.activeConcludedOnchain = append(partitioned.activeConcludedOnchain, key)
			} else {
				partitioned.activeUnconcludedOnchain = append(partitioned.activeUnconcludedOnchain, key)
			}
		} else {
			partitioned.activeUnknownOnchain = append(partitioned.activeUnknownOnchain, key)
		}
	}

	// ... and INACTIVE in three more
	for key := range inactive {
		if onchainState, ok := onchain[key]; ok {
			if concludedOnchain(&onchainState) {
				partitioned.inactiveConcludedOnchain = append(
					partitioned.inactiveConcludedOnchain,
					key,
				)
			} else {
				partitioned.inactiveUnconcludedOnchain = append(partitioned.inactiveUnconcludedOnchain, key)
			}
		} else {
			partitioned.inactiveUnknownOnchain = append(partitioned.inactiveUnknownOnchain, key)
		}
	}

	return partitioned
}

// isVoteWorthToKeep determines if a vote is worth to be kept, based on the onchain disputes.
func isVoteWorthToKeep(
	validatorIndex parachaintypes.ValidatorIndex,
	disputeStatement parachaintypes.DisputeStatement,
	onchainState parachaintypes.DisputeState,
) bool {
	var offchainVote bool
	var validKind *parachaintypes.ValidDisputeStatementKind

	statement, err := disputeStatement.Value()
	if err != nil {
		panic(fmt.Sprintf("getting value from disputeStatement: %s", err))
	}

	switch inner := statement.(type) {
	case parachaintypes.ValidDisputeStatement:
		offchainVote = true
		validKind = &inner.Kind
	case parachaintypes.InvalidDisputeStatement:
		offchainVote = false
		validKind = nil
	}

	if validKind != nil {
		stmtKind, err := validKind.Value()
		if err != nil {
			panic("unexpected empty inner in ValidDisputeStatementKind")
		}

		// We want to keep all backing votes. This maximises the number of backers
		// punished when misbehaving.
		switch stmtKind.(type) {
		case parachaintypes.Valid, parachaintypes.SecondedCandidateHash:
			return true
		}
	}

	inValidatorsFor, err := onchainState.ValidatorsFor.Get(uint(validatorIndex))
	if err != nil {
		logger.Warnf("validator index out of bounds: %d", validatorIndex)
		inValidatorsFor = false
	}

	inValidatorsAgainst, err := onchainState.ValidatorsAgainst.Get(uint(validatorIndex))
	if err != nil {
		logger.Warnf("validator index out of bounds: %d", validatorIndex)
		inValidatorsAgainst = false
	}

	if inValidatorsFor && inValidatorsAgainst {
		// The validator has double voted and runtime knows about this. Ignore this vote.
		return false
	}

	if offchainVote && inValidatorsAgainst || !offchainVote && inValidatorsFor {
		// offchain vote differs from the onchain vote
		// we need this vote to punish the offending validator
		return true
	}

	// The vote is valid. Return true if it is not seen onchain.
	return !inValidatorsFor && !inValidatorsAgainst
}

// getOnchainDisputes gets the on-chain disputes at a given block number and returns them as a map
// for efficient searching. It takes a relay parent hash and returns a map of session index and
// candidate hash tuples to dispute states.
func getOnchainDisputes(
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

// requestVotes requests the relevant dispute statements for a set of disputes identified
// by CandidateHash and SessionIndex.
func requestVotes(overseerChan chan<- any, disputesToQuery []parachaintypes.DisputeKey) (
	[]disputemessages.CandidateVotesResponse, error,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	responseCh := make(chan []disputemessages.CandidateVotesResponse)
	query := disputemessages.QueryCandidateVotes{
		Query:    disputesToQuery,
		Response: responseCh,
	}

	overseerChan <- query

	select {
	case v := <-responseCh:
		return v, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("while querying candidate votes: %w", ctx.Err())
	}
}

func requestDisputes(overseerChan chan<- any) ([]disputemessages.RecentDispute, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	responseCh := make(chan []disputemessages.RecentDispute)
	msg := disputemessages.GetRecentDisputes{
		Response: responseCh,
	}

	overseerChan <- msg

	select {
	case recentDisputes := <-responseCh:
		return recentDisputes, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("while gathering recent disputes: %w", ctx.Err())
	}
}

// newMultiDisputeStatementSet converts vote selection results into a MultiDisputeStatementSet.
func newMultiDisputeStatementSet(
	voteResults []voteSelectionResult,
) parachaintypes.MultiDisputeStatementSet {
	disputeStmts := make(parachaintypes.MultiDisputeStatementSet, 0)

	for _, res := range voteResults {
		sessionIndex := res.key.SessionIndex
		candidateHash := res.key.CandidateHash
		votes := res.votes

		statements := make([]parachaintypes.DisputeStatementEntry, 0)

		votes.Valid.Ascend(
			parachaintypes.ValidatorIndex(0),
			func(validatorIdx parachaintypes.ValidatorIndex,
				vote parachaintypes.Vote[parachaintypes.ValidDisputeStatementKind],
			) bool {
				validStmt := new(parachaintypes.DisputeStatement)
				err := validStmt.SetValue(parachaintypes.ValidDisputeStatement{
					Kind: vote.Kind,
				})
				if err != nil {
					panic(fmt.Sprintf("unexpected set value fail: %s", err.Error()))
				}

				statements = append(statements, parachaintypes.DisputeStatementEntry{
					Index:     validatorIdx,
					Signature: vote.Signature,
					Statement: *validStmt,
				})
				return true
			},
		)

		votes.Invalid.Ascend(
			parachaintypes.ValidatorIndex(0),
			func(validatorIdx parachaintypes.ValidatorIndex,
				vote parachaintypes.Vote[parachaintypes.InvalidDisputeStatementKind],
			) bool {
				invalidStmt := new(parachaintypes.DisputeStatement)
				err := invalidStmt.SetValue(parachaintypes.InvalidDisputeStatement{
					Kind: vote.Kind,
				})
				if err != nil {
					panic(fmt.Sprintf("unexpected set value fail: %s", err.Error()))
				}

				statements = append(statements, parachaintypes.DisputeStatementEntry{
					Index:     validatorIdx,
					Signature: vote.Signature,
					Statement: *invalidStmt,
				})
				return true
			},
		)

		disputeStmts = append(disputeStmts, parachaintypes.DisputeStatementSet{
			Session:       sessionIndex,
			CandidateHash: candidateHash,
			Statements:    statements,
		})
	}

	return disputeStmts
}

// containsOnchain checks if the onchain disputes map has a dispute with the given session index and candidate hash.
func containsOnchain(
	onchain map[parachaintypes.DisputeKey]parachaintypes.DisputeState,
	session parachaintypes.SessionIndex,
	candidateHash parachaintypes.CandidateHash,
) bool {
	key := parachaintypes.DisputeKey{SessionIndex: session, CandidateHash: candidateHash}
	_, exists := onchain[key]
	return exists
}
