// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package chainapi

import (
	"fmt"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type hashHeader struct {
	Hash   common.Hash
	Header types.Header
}
type ChainAPIMessage[message any] struct {
	Message         message
	ResponseChannel chan any
}

type AncestorsResponse struct {
	Ancestors []common.Hash
	Error     error
}

// Ancestors is a message to get the ancestors of a block.
type Ancestors struct {
	Hash              common.Hash
	numberOfAncestors uint32
}

type BlockHeader common.Hash

// DetermineNewBlocks determines the hashes of all new blocks we should track metadata for, given this head.
//
// This is guaranteed to be a subset of the (inclusive) ancestry of `head` determined as all
// blocks above the lower bound or above the highest known block, whichever is higher.
// This is formatted in descending order by block height.
//
// An implication of this is that if `head` itself is known or not above the lower bound,
// then the returned list will be empty.
//
// This may be somewhat expensive when first recovering from major sync.
func DetermineNewBlocks(subsystemToOverseer chan<- any, isKnown func(hash common.Hash) bool, head common.Hash,
	header types.Header,
	lowerBoundNumber parachaintypes.BlockNumber) ([]hashHeader, error) {
	const maxNumberOfAncestors = 4
	minBlockNeeded := uint(lowerBoundNumber + 1)

	// Early exit if the block is in the DB or too early.
	alreadyKnown := isKnown(head)
	beforeRelevant := header.Number < minBlockNeeded
	if alreadyKnown || beforeRelevant {
		return nil, nil
	}

	ancestry := make([]hashHeader, 0)
	headerClone, err := header.DeepCopy()
	if err != nil {
		return nil, fmt.Errorf("failed to deep copy header: %w", err)
	}

	ancestry = append(ancestry, hashHeader{Hash: head, Header: *headerClone})

	// Early exit if the parent hash is in the DB or no further blocks are needed.
	if isKnown(header.ParentHash) || header.Number == minBlockNeeded {
		return ancestry, nil
	}

	lastHeader := ancestry[len(ancestry)-1].Header
	// This is always non-zero as determined by the loop invariant above.
	numberOfAncestors := min(maxNumberOfAncestors, (lastHeader.Number - minBlockNeeded))

	ancestors, err := GetBlockAncestors(subsystemToOverseer, head, uint32(numberOfAncestors))
	if err != nil {
		return nil, fmt.Errorf("getting block ancestors: %w", err)
	}
	fmt.Printf("ancestors: %v\n", ancestors)
	// TODO: finish this, see issue #3933

	return ancestry, nil
}

// GetBlockAncestors sends a message to the overseer to get the ancestors of a block.
func GetBlockAncestors(
	overseerChannel chan<- any,
	head common.Hash,
	numAncestors uint32,
) ([]common.Hash, error) {
	respChan := make(chan any, 1)
	message := ChainAPIMessage[Ancestors]{
		Message: Ancestors{
			Hash:              head,
			numberOfAncestors: numAncestors,
		},
		ResponseChannel: respChan,
	}
	res, err := Call(overseerChannel, message, message.ResponseChannel)
	if err != nil {
		return nil, fmt.Errorf("sending message to get block ancestors: %w", err)
	}

	response, ok := res.(AncestorsResponse)
	if !ok {
		return nil, fmt.Errorf("getting block ancestors: got unexpected response type %T", res)
	}
	if response.Error != nil {
		return nil, fmt.Errorf("getting block ancestors: %w", response.Error)
	}

	return response.Ancestors, nil
}

// Call sends the given message to the given channel and waits for a response with a timeout
func Call(channel chan<- any, message any, responseChan chan any) (any, error) {
	channel <- message
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(timeout):
		return nil, parachaintypes.ErrSubsystemRequestTimeout
	}
}

const timeout = 10 * time.Second

func GetNumberOfValidators() uint {
	// TODO: implement this, currently it's just a stub that should be replaced, see issue #3932
	return 10
}
