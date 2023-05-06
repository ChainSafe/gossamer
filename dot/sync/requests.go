package sync

import (
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
)

func singleBlockRequest(blockHash common.Hash, requestedData byte) *network.BlockRequestMessage {
	one := uint32(1)
	return &network.BlockRequestMessage{
		RequestedData: requestedData,
		StartingBlock: *variadic.MustNewUint32OrHash(blockHash),
		Direction:     network.Descending,
		Max:           &one,
	}
}

func descendingBlockRequest(blockHash common.Hash, amount uint32, requestedData byte) *network.BlockRequestMessage {
	return &network.BlockRequestMessage{
		RequestedData: requestedData,
		StartingBlock: *variadic.MustNewUint32OrHash(blockHash),
		Direction:     network.Descending,
		Max:           &amount,
	}
}

func ascedingBlockRequests(startNumber, targetNumber uint, requestedData byte) []*network.BlockRequestMessage {
	if startNumber > targetNumber {
		return []*network.BlockRequestMessage{}
	}

	diff := targetNumber - startNumber

	// start and end block are the same, just request 1 block
	if diff == 0 {
		one := uint32(1)
		return []*network.BlockRequestMessage{
			{
				RequestedData: requestedData,
				StartingBlock: *variadic.MustNewUint32OrHash(uint32(startNumber)),
				Direction:     network.Ascending,
				Max:           &one,
			},
		}
	}

	numRequests := diff / maxResponseSize
	// we should check if the diff is in the maxResponseSize bounds
	// otherwise we should increase the numRequests by one, take this
	// example, we want to sync from 0 to 259, the diff is 259
	// then the num of requests is 2 (uint(259)/uint(128)) however two requests will
	// retrieve only 256 blocks (each request can retrive a max of 128 blocks), so we should
	// create one more request to retrive those missing blocks, 3 in this example.
	missingBlocks := diff % maxResponseSize
	if missingBlocks != 0 {
		numRequests++
	}

	reqs := make([]*network.BlockRequestMessage, numRequests)
	// check if we want to specify a size
	for i := uint(0); i < numRequests; i++ {
		max := uint32(maxResponseSize)

		lastIteration := numRequests - 1
		if i == lastIteration && missingBlocks != 0 {
			max = uint32(missingBlocks)
		}

		start := variadic.MustNewUint32OrHash(startNumber)

		reqs[i] = &network.BlockRequestMessage{
			RequestedData: requestedData,
			StartingBlock: *start,
			Direction:     network.Ascending,
			Max:           &max,
		}
		startNumber += uint(max)
	}

	return reqs
}

func totalOfBlocksRequested(requests []*network.BlockRequestMessage) (total uint32) {
	for _, request := range requests {
		if request.Max != nil {
			total += *request.Max
		}
	}

	return total
}
