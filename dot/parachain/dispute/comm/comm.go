package comm

import (
	"context"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

const timeout = 10 * time.Second

var logger = log.NewFromGlobal(log.AddContext("parachain", "disputes"))

// SendMessage sends the given message to the given channel with a timeout
func SendMessage(channel chan<- any, message any) error {
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

// Call sends the given message to the given channel and waits for a response with a timeout
func Call(channel chan<- any, message any, responseChan chan any) (any, error) {
	if err := SendMessage(channel, message); err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case response := <-responseChan:
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetBlockNumber returns the block number of the given receipt
func GetBlockNumber(overseerChannel chan<- any, receipt parachainTypes.CandidateReceipt) (uint32, error) {
	respCh := make(chan any, 1)
	relayParent, err := receipt.Hash()
	if err != nil {
		return 0, fmt.Errorf("get hash: %w", err)
	}

	message := overseer.ChainAPIMessage[overseer.BlockNumber]{
		Message:         overseer.BlockNumber{Hash: relayParent},
		ResponseChannel: respCh,
	}
	result, err := Call(overseerChannel, message, message.ResponseChannel)
	if err != nil {
		return 0, fmt.Errorf("send message: %w", err)
	}

	blockNumber, ok := result.(uint32)
	if !ok {
		return 0, fmt.Errorf("unexpected response type: %T", result)
	}
	return blockNumber, nil
}

// GetFinalisedBlockNumber sends a message to the overseer to get the finalised block number.
func GetFinalisedBlockNumber(overseerChannel chan<- any) (uint32, error) {
	message := overseer.ChainAPIMessage[overseer.FinalizedBlockNumber]{
		ResponseChannel: make(chan any, 1),
	}
	res, err := Call(overseerChannel, message, message.ResponseChannel)
	if err != nil {
		return 0, fmt.Errorf("sending message to get finalised block number: %w", err)
	}

	response, ok := res.(overseer.BlockNumberResponse)
	if !ok {
		return 0, fmt.Errorf("getting finalised block number: got unexpected response type %T", res)
	}

	if response.Err != nil {
		return 0, fmt.Errorf("getting finalised block number: %w", response.Err)
	}

	return response.Number, nil
}

// GetBlockAncestors sends a message to the overseer to get the ancestors of a block.
func GetBlockAncestors(
	overseerChannel chan<- any,
	head common.Hash,
	numAncestors uint32,
) ([]common.Hash, error) {
	respChan := make(chan any, 1)
	message := overseer.ChainAPIMessage[overseer.Ancestors]{
		Message: overseer.Ancestors{
			Hash: head,
			K:    numAncestors,
		},
		ResponseChannel: respChan,
	}
	res, err := Call(overseerChannel, message, message.ResponseChannel)
	if err != nil {
		return nil, fmt.Errorf("sending message to get block ancestors: %w", err)
	}

	response, ok := res.(overseer.AncestorsResponse)
	if !ok {
		return nil, fmt.Errorf("getting block ancestors: got unexpected response type %T", res)
	}
	if response.Error != nil {
		return nil, fmt.Errorf("getting block ancestors: %w", response.Error)
	}

	return response.Ancestors, nil
}

// SendResult sends the given participation outcome to the channel
func SendResult(channel chan<- any, request types.ParticipationRequest, outcome types.ParticipationOutcomeType) {
	participationOutcome, err := types.NewCustomParticipationOutcomeVDT(outcome)
	if err != nil {
		logger.Errorf(
			"failed to create participation outcome: %v, error: %s",
			outcome,
			err,
		)
		return
	}

	message := types.Message[types.ParticipationStatement]{
		Data: types.ParticipationStatement{
			Session:          request.Session,
			CandidateHash:    request.CandidateHash,
			CandidateReceipt: request.CandidateReceipt,
			Outcome:          participationOutcome,
		},
		ResponseChannel: nil,
	}
	if err := SendMessage(channel, message); err != nil {
		logger.Errorf(
			"failed to send participation statement: %v, error: %s",
			message,
			err,
		)
	}
}
