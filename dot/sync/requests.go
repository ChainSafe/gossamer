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

func ascedingBlockRequests(startNumber uint, targetNumber uint, requestedData byte) ([]*network.BlockRequestMessage, error) {
	diff := int(targetNumber) - int(startNumber)
	if diff < 0 {
		return nil, errInvalidDirection
	}

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
		}, nil
	}

	numRequests := uint(diff) / maxResponseSize
	if diff%maxResponseSize != 0 {
		numRequests++
	}

	reqs := make([]*network.BlockRequestMessage, numRequests)

	// check if we want to specify a size
	const max = uint32(maxResponseSize)
	for i := uint(0); i < numRequests; i++ {
		max := max
		start := variadic.MustNewUint32OrHash(startNumber)

		reqs[i] = &network.BlockRequestMessage{
			RequestedData: requestedData,
			StartingBlock: *start,
			Direction:     network.Ascending,
			Max:           &max,
		}
		startNumber += uint(max)
	}

	return reqs, nil
}

func totalRequestedBlocks(requests []*network.BlockRequestMessage) uint32 {
	acc := uint32(0)

	for _, request := range requests {
		if request.Max != nil {
			acc += *request.Max
		}
	}

	return acc
}
