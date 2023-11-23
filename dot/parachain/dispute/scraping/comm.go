package scraping

import (
	"context"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/lib/common"
	"time"
)

const timeout = 10 * time.Second

// getFinalisedBlockNumber sends a message to the overseer to get the finalised block number.
func getFinalisedBlockNumber(overseerChannel chan<- any) (uint32, error) {
	message := overseer.ChainAPIMessage[overseer.FinalizedBlockNumberRequest]{
		ResponseChannel: make(chan any, 1),
	}
	res, err := call(overseerChannel, message, message.ResponseChannel)
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

// getBlockAncestors sends a message to the overseer to get the ancestors of a block.
func getBlockAncestors(
	overseerChannel chan<- any,
	head common.Hash,
	numAncestors uint32,
) ([]common.Hash, error) {
	respChan := make(chan any, 1)
	message := overseer.ChainAPIMessage[overseer.AncestorsRequest]{
		Message: overseer.AncestorsRequest{
			Hash: head,
			K:    numAncestors,
		},
		ResponseChannel: respChan,
	}
	res, err := call(overseerChannel, message, message.ResponseChannel)
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
