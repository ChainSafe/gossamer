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
