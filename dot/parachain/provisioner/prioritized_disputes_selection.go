package provisioner

import (
	"context"
	"fmt"
	"iter"
	"time"

	disputemessages "github.com/ChainSafe/gossamer/dot/parachain/disputes-coordinator/messages"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/primitives"
)

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance parachain.RuntimeInstance, err error)
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

// Iter returns an iterator over the PartitionedDisputes.
func (pd partitionedDisputes) Iter() iter.Seq2[parachaintypes.SessionIndex, parachaintypes.CandidateHash] {
	return func(yield func(parachaintypes.SessionIndex, parachaintypes.CandidateHash) bool) {
		seqToIterate := [][]parachaintypes.DisputeKey{
			pd.inactiveUnknownOnchain,
			pd.inactiveUnconcludedOnchain,
			pd.activeUnknownOnchain,
			pd.activeUnconcludedOnchain,
			pd.activeConcludedOnchain,
			// pd.InactiveConcludedOnchain is dropped on purpose
		}

		for _, seq := range seqToIterate {
			for _, d := range seq {
				if !yield(d.SessionIndex, d.CandidateHash) {
					return
				}
			}
		}
	}
}

func concludedOnchain(onchainState *parachaintypes.DisputeState) bool {
	n := onchainState.ValidatorsFor.Len()
	supermajority := n - (primitives.SaturatingSub(n, 1) / 3)

	return onchainState.ValidatorsFor.CountOnes() >= supermajority ||
		onchainState.ValidatorsAgainst.CountOnes() >= supermajority
}

func partitionRecentDisputes(
	recent []disputemessages.RecentDisputesResponse,
	onchain map[parachaintypes.DisputeKey]parachaintypes.DisputeState,
) partitionedDisputes {
	partitioned := partitionedDisputes{}

	// Drop any duplicates
	uniqueRecent := make(map[parachaintypes.DisputeKey]parachaintypes.DisputeStatus)
	for _, r := range recent {
		uniqueRecent[parachaintypes.DisputeKey{SessionIndex: r.SessionIndex, CandidateHash: r.CandidateHash}] = r.DisputeStatus
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
				partitioned.inactiveConcludedOnchain = append(partitioned.inactiveConcludedOnchain, key)
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
		panic("unexpected empty inner in DisputeStatement")
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

		// We want to keep all backing votes. This maximizes the number of backers
		// punished when misbehaving.
		switch stmtKind.(type) {
		case parachaintypes.BackingValid, parachaintypes.BackingSeconded:
			return true
		}
	}

	inValidatorsFor := onchainState.ValidatorsFor.Get(int(validatorIndex))
	inValidatorsAgainst := onchainState.ValidatorsAgainst.Get(int(validatorIndex))

	if inValidatorsFor && inValidatorsAgainst {
		// The validator has double voted and runtime knows about this. Ignore this vote.
		return false
	}

	if (offchainVote && inValidatorsAgainst) || (!offchainVote && inValidatorsFor) {
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
	[]disputemessages.CandidateVotesResponse, error) {
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

func requestDisputes(overseerChan chan<- any) ([]disputemessages.RecentDisputesResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	responseCh := make(chan []disputemessages.RecentDisputesResponse)
	msg := disputemessages.RecentDisputes{
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
