// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package util

import (
	"context"
	"fmt"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// / Given a new chain-head hash, this determines the hashes of all new blocks we should track
// / metadata for, given this head.
// /
// / This is guaranteed to be a subset of the (inclusive) ancestry of `head` determined as all
// / blocks above the lower bound or above the highest known block, whichever is higher.
// / This is formatted in descending order by block height.
// /
// / An implication of this is that if `head` itself is known or not above the lower bound,
// / then the returned list will be empty.
// /
// / This may be somewhat expensive when first recovering from major sync.
// pub async fn determine_new_blocks<E, Sender>(
//
//	sender: &mut Sender,
//	is_known: impl Fn(&Hash) -> Result<bool, E>,
//	head: Hash,
//	header: &Header,
//	lower_bound_number: BlockNumber,
//
// ) -> Result<Vec<(Hash, Header)>, E>
// where
//
//	Sender: SubsystemSender<ChainApiMessage>,
//
//	{
//		const ANCESTRY_STEP: usize = 4;
//
//		let min_block_needed = lower_bound_number + 1;
//
//		// Early exit if the block is in the DB or too early.
//		{
//			let already_known = is_known(&head)?;
//
//			let before_relevant = header.number < min_block_needed;
//
//			if already_known || before_relevant {
//				return Ok(Vec::new())
//			}
//		}
//
//		let mut ancestry = vec![(head, header.clone())];
//
//		// Early exit if the parent hash is in the DB or no further blocks
//		// are needed.
//		if is_known(&header.parent_hash)? || header.number == min_block_needed {
//			return Ok(ancestry)
//		}
//
//		'outer: loop {
//			let (last_hash, last_header) = ancestry
//				.last()
//				.expect("ancestry has length 1 at initialization and is only added to; qed");
//
//			assert!(
//				last_header.number > min_block_needed,
//				"Loop invariant: the last block in ancestry is checked to be \
//				above the minimum before the loop, and at the end of each iteration; \
//				qed"
//			);
//
//			let (tx, rx) = oneshot::channel();
//
//			// This is always non-zero as determined by the loop invariant
//			// above.
//			let ancestry_step =
//				std::cmp::min(ANCESTRY_STEP, (last_header.number - min_block_needed) as usize);
//
//			let batch_hashes = if ancestry_step == 1 {
//				vec![last_header.parent_hash]
//			} else {
//				sender
//					.send_message(
//							ChainApiMessage::Ancestors {
//							hash: *last_hash,
//							k: ancestry_step,
//							response_channel: tx,
//						}
//						.into(),
//					)
//					.await;
//
//				// Continue past these errors.
//				match rx.await {
//					Err(_) | Ok(Err(_)) => break 'outer,
//					Ok(Ok(ancestors)) => ancestors,
//				}
//			};
//
//			let batch_headers = {
//				let (batch_senders, batch_receivers) = (0..batch_hashes.len())
//					.map(|_| oneshot::channel())
//					.unzip::<_, _, Vec<_>, Vec<_>>();
//
//				for (hash, batched_sender) in batch_hashes.iter().cloned().zip(batch_senders) {
//					sender
//						.send_message(ChainApiMessage::BlockHeader(hash, batched_sender).into())
//						.await;
//				}
//
//				let mut requests = futures::stream::FuturesOrdered::new();
//				batch_receivers
//					.into_iter()
//					.map(|rx| async move {
//						match rx.await {
//							Err(_) | Ok(Err(_)) => None,
//							Ok(Ok(h)) => h,
//						}
//					})
//					.for_each(|x| requests.push_back(x));
//
//				let batch_headers: Vec<_> =
//					requests.flat_map(|x: Option<Header>| stream::iter(x)).collect().await;
//
//				// Any failed header fetch of the batch will yield a `None` result that will
//				// be skipped. Any failure at this stage means we'll just ignore those blocks
//				// as the chain DB has failed us.
//				if batch_headers.len() != batch_hashes.len() {
//					break 'outer
//				}
//				batch_headers
//			};
//
//			for (hash, header) in batch_hashes.into_iter().zip(batch_headers) {
//				let is_known = is_known(&hash)?;
//
//				let is_relevant = header.number >= min_block_needed;
//				let is_terminating = header.number == min_block_needed;
//
//				if is_known || !is_relevant {
//					break 'outer
//				}
//
//				ancestry.push((hash, header));
//
//				if is_terminating {
//					break 'outer
//				}
//			}
//		}
//
//		Ok(ancestry)
//	}

type HashHeader struct {
	Hash   common.Hash
	header types.Header
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

func DetermineNewBlocks(subsystemToOverseer chan<- any, isKnown func(hash common.Hash) bool, head common.Hash,
	header types.Header,
	lowerBoundNumber parachaintypes.BlockNumber) ([]HashHeader, error) {
	fmt.Printf("determineNewBlocks\n")

	minBlockNeeded := uint(lowerBoundNumber + 1)

	// Early exit if the block is in the DB or too early.
	alreadyKnown := isKnown(head)

	beforeRelevant := header.Number < minBlockNeeded
	fmt.Printf("beforeRelevant: %v\n", beforeRelevant)
	if alreadyKnown || beforeRelevant {
		return make([]HashHeader, 0), nil
	}

	ancestry := make([]HashHeader, 0)
	headerClone, err := header.DeepCopy()
	if err != nil {
		return nil, fmt.Errorf("failed to deep copy header: %w", err)
	}

	ancestry = append(ancestry, HashHeader{Hash: head, header: *headerClone})

	// Early exit if the parent hash is in the DB or no further blocks are needed.
	if isKnown(header.ParentHash) || header.Number == minBlockNeeded {
		return ancestry, nil
	}

	lastHeader := ancestry[len(ancestry)-1].header
	// This is always non-zero as determined by the loop invariant above.
	ancestryStep := min(4, (lastHeader.Number - minBlockNeeded))

	fmt.Printf("ancestryStep: %v\n", ancestryStep)

	ancestors, err := GetBlockAncestors(subsystemToOverseer, head, uint32(ancestryStep))
	if err != nil {
		return nil, fmt.Errorf("getting block ancestors: %w", err)
	}
	fmt.Printf("ancestors: %v\n", ancestors)

	// outer loop, build ancestry
	//for {
	// call ChainApiMessage::Ancestors to get batch hashes

	// build batch headers from batch hashes

	// loop batch_hashes, build ancestry
	//}

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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case response := <-responseChan:
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

const timeout = 10 * time.Second

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

func RequestCandidateEvents(hash common.Hash) []parachaintypes.CandidateEvent {
	return nil
}
