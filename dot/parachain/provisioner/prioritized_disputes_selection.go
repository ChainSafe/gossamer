package provisioner

import (
	"context"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/disputes-coordinator/messages"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance parachain.RuntimeInstance, err error)
}

// IsVoteWorthToKeep determines if a vote is worth to be kept, based on the onchain disputes.
func IsVoteWorthToKeep(
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

// GetOnchainDisputes gets the on-chain disputes at a given block number and returns them as a map
// for efficient searching. It takes a relay parent hash and returns a map of session index and
// candidate hash tuples to dispute states.
func GetOnchainDisputes(
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

// RequestVotes requests the relevant dispute statements for a set of disputes identified
// by CandidateHash and SessionIndex.
func RequestVotes(overseerChan chan<- any, disputesToQuery []parachaintypes.DisputeKey) (
	[]messages.CandidateVotesResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	responseCh := make(chan []messages.CandidateVotesResponse)
	query := messages.QueryCandidateVotes{
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

// RequestDisputes requests disputes identified by CandidateHash and SessionIndex.
func RequestDisputes(overseerChan chan<- any) ([]messages.RecentDisputesResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	responseCh := make(chan []messages.RecentDisputesResponse)
	msg := messages.RecentDisputes{
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
