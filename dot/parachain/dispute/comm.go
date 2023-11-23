package dispute

import (
	"context"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"time"
)

const timeout = 10 * time.Second

func getBlockNumber(overseerChannel chan<- any, receipt parachainTypes.CandidateReceipt) (uint32, error) {
	respCh := make(chan any, 1)
	relayParent, err := receipt.Hash()
	if err != nil {
		return 0, fmt.Errorf("get hash: %w", err)
	}

	message := overseer.ChainAPIMessage[overseer.BlockNumberRequest]{
		Message:         overseer.BlockNumberRequest{Hash: relayParent},
		ResponseChannel: respCh,
	}
	result, err := call(overseerChannel, message, message.ResponseChannel)
	if err != nil {
		return 0, fmt.Errorf("send message: %w", err)
	}

	blockNumber, ok := result.(uint32)
	if !ok {
		return 0, fmt.Errorf("unexpected response type: %T", result)
	}
	return blockNumber, nil
}

func sendMessage(channel chan<- any, message any) error {
	// Send with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case channel <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func call(channel chan<- any, message any, responseChan chan any) (any, error) {
	// Send with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case channel <- message:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case response := <-responseChan:
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func sendResult(overseerChannel chan<- any, request ParticipationRequest, outcome types.ParticipationOutcomeType) {
	participationOutcome, err := types.NewCustomParticipationOutcomeVDT(outcome)
	if err != nil {
		logger.Errorf(
			"failed to create participation outcome: %s, error: %s",
			outcome,
			err,
		)
		return
	}

	statement := ParticipationStatement{
		Session:          request.session,
		CandidateHash:    request.candidateHash,
		CandidateReceipt: request.candidateReceipt,
		Outcome:          participationOutcome,
	}
	if err := sendMessage(overseerChannel, statement); err != nil {
		logger.Errorf(
			"failed to send participation statement: %s, error: %s",
			statement,
			err,
		)
	}
}
