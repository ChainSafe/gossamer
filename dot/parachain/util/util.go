// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package util

import (
	"fmt"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
)

// SigningKeyAndIndex finds the first key we can sign with from the given set of validators,
// if any, and returns it along with the validator index.
func SigningKeyAndIndex(
	validators []parachaintypes.ValidatorID,
	ks keystore.Keystore,
) (*parachaintypes.ValidatorID, parachaintypes.ValidatorIndex) {
	for i, validator := range validators {
		publicKey, _ := sr25519.NewPublicKey(validator[:])
		keypair := ks.GetKeypair(publicKey)

		if keypair != nil {
			return &validator, parachaintypes.ValidatorIndex(i)
		}
	}
	return nil, 0
}

type HashHeader struct {
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

type Ancestors struct {
	Hash common.Hash
	K    uint32
}

type BlockHeader struct {
	Hash common.Hash
}

// DetermineNewBlocks determines the hashes of all new blocks we should track metadata for, given this head.
// Given a new chain-head hash, this determines the hashes of all new blocks we should track
// metadata for, given this head.
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
	lowerBoundNumber parachaintypes.BlockNumber) ([]HashHeader, error) {

	minBlockNeeded := uint(lowerBoundNumber + 1)

	// Early exit if the block is in the DB or too early.
	alreadyKnown := isKnown(head)

	beforeRelevant := header.Number < minBlockNeeded
	if alreadyKnown || beforeRelevant {
		return make([]HashHeader, 0), nil
	}

	ancestry := make([]HashHeader, 0)
	headerClone, err := header.DeepCopy()
	if err != nil {
		return nil, fmt.Errorf("failed to deep copy header: %w", err)
	}

	ancestry = append(ancestry, HashHeader{Hash: head, Header: *headerClone})

	// Early exit if the parent hash is in the DB or no further blocks are needed.
	if isKnown(header.ParentHash) || header.Number == minBlockNeeded {
		return ancestry, nil
	}

	lastHeader := ancestry[len(ancestry)-1].Header
	// This is always non-zero as determined by the loop invariant above.
	ancestryStep := min(4, (lastHeader.Number - minBlockNeeded))

	ancestors, err := GetBlockAncestors(subsystemToOverseer, head, uint32(ancestryStep))
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
			Hash: head,
			K:    numAncestors,
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
	if err := SendMessage(channel, message); err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

const timeout = 10 * time.Second

// SendMessage sends the given message to the given channel with a timeout
func SendMessage(channel chan<- any, message any) error {
	// Send with timeout

	select {
	case channel <- message:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout sending message")
	}
}
