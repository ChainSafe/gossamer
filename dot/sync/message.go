// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

// CreateBlockResponse creates a block response message from a block request message
func (s *Service) CreateBlockResponse(from peer.ID, req *network.BlockRequestMessage) (*network.BlockResponseMessage, error) {
	logger.Debugf("sync request from %s: %s", from, req.String())

	switch req.Direction {
	case network.Ascending:
		return s.handleAscendingRequest(req)
	case network.Descending:
		return s.handleDescendingRequest(req)
	default:
		return nil, errInvalidRequestDirection
	}
}

func (s *Service) handleAscendingRequest(req *network.BlockRequestMessage) (*network.BlockResponseMessage, error) {
	var (
		max         uint = network.MaxBlocksInResponse
		startHash   *common.Hash
		startNumber uint
	)

	// determine maximum response size
	if req.Max != nil && *req.Max < network.MaxBlocksInResponse {
		max = uint(*req.Max)
	}

	bestBlockNumber, err := s.blockState.BestBlockNumber()
	if err != nil {
		return nil, fmt.Errorf("getting best block for request: %w", err)
	}

	switch startBlock := req.StartingBlock.Value().(type) {
	case uint32:
		if startBlock == 0 {
			startBlock = 1
		}

		// if request start is higher than our best block, return error
		if bestBlockNumber < uint(startBlock) {
			return nil, errRequestStartTooHigh
		}

		startNumber = uint(startBlock)
	case common.Hash:
		startHash = &startBlock

		// make sure we actually have the starting block
		header, err := s.blockState.GetHeader(*startHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get start block %s for request: %w", startHash, err)
		}

		startNumber = header.Number
	default:
		return nil, ErrInvalidBlockRequest
	}

	endNumber := startNumber + max - 1
	if endNumber > bestBlockNumber {
		endNumber = bestBlockNumber
	}

	var endHash *common.Hash
	if startHash != nil {
		eh, err := s.checkOrGetDescendantHash(*startHash, nil, endNumber)
		if err != nil {
			return nil, err
		}

		endHash = &eh
	}

	if startHash == nil {
		logger.Debugf("ascending request, "+
			"start block number: %d, "+
			"end block number: %d",
			startNumber, endNumber)

		return s.handleAscendingByNumber(startNumber, endNumber, req.RequestedData)
	}

	logger.Debugf("ascending request, "+
		"start block hash: %s, "+
		"end block hash: %s",
		*startHash, *endHash)

	return s.handleChainByHash(*startHash, *endHash, max, req.RequestedData, req.Direction)
}

func (s *Service) handleDescendingRequest(req *network.BlockRequestMessage) (*network.BlockResponseMessage, error) {
	var (
		startHash   *common.Hash
		startNumber uint
		max         uint = network.MaxBlocksInResponse
	)

	// determine maximum response size
	if req.Max != nil && *req.Max < network.MaxBlocksInResponse {
		max = uint(*req.Max)
	}

	switch startBlock := req.StartingBlock.Value().(type) {
	case uint32:
		bestBlockNumber, err := s.blockState.BestBlockNumber()
		if err != nil {
			return nil, fmt.Errorf("failed to get best block %d for request: %w", bestBlockNumber, err)
		}

		// if request start is higher than our best block, only return blocks from our best block and below
		if bestBlockNumber < uint(startBlock) {
			startNumber = bestBlockNumber
		} else {
			startNumber = uint(startBlock)
		}
	case common.Hash:
		startHash = &startBlock

		// make sure we actually have the starting block
		header, err := s.blockState.GetHeader(*startHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get start block %s for request: %w", startHash, err)
		}

		startNumber = header.Number
	default:
		return nil, ErrInvalidBlockRequest
	}

	endNumber := uint(1)
	if startNumber > max+1 {
		endNumber = startNumber - max + 1
	}

	var endHash *common.Hash
	if startHash != nil {
		// need to get blocks by subchain if start hash is provided, get end hash
		endHeader, err := s.blockState.GetHeaderByNumber(endNumber)
		if err != nil {
			return nil, fmt.Errorf("getting end block %d for request: %w", endNumber, err)
		}

		hash := endHeader.Hash()
		endHash = &hash
	}

	if startHash == nil || endHash == nil {
		logger.Debugf("descending request, "+
			"start block number: %s, "+
			"end block number: %s",
			startNumber, endNumber)
		return s.handleDescendingByNumber(startNumber, endNumber, req.RequestedData)
	}

	logger.Debugf("descending request, "+
		"start block hash: %s, "+
		"end block hash: %s",
		*startHash, *endHash)
	return s.handleChainByHash(*endHash, *startHash, max, req.RequestedData, req.Direction)
}

// checkOrGetDescendantHash checks if the provided `descendant` is
// on the same chain as the `ancestor`, if it's provided, otherwise
// it sets `descendant` to a block with number=`descendantNumber` that is a descendant of the ancestor.
// If used with an Ascending request, ancestor is the start block and descendant is the end block
// If used with an Descending request, ancestor is the end block and descendant is the start block
func (s *Service) checkOrGetDescendantHash(ancestor common.Hash,
	descendant *common.Hash, descendantNumber uint) (common.Hash, error) {
	// if `descendant` was provided, check that it's a descendant of `ancestor`
	if descendant != nil {
		header, err := s.blockState.GetHeader(ancestor)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to get descendant %s: %w", *descendant, err)
		}

		// if descendant number is lower than ancestor number, this is an error
		if header.Number > descendantNumber {
			return common.Hash{},
				fmt.Errorf("invalid request, descendant number %d is lower than ancestor %d",
					descendantNumber, header.Number)
		}

		// check if provided start hash is descendant of provided descendant hash
		is, err := s.blockState.IsDescendantOf(ancestor, *descendant)
		if err != nil {
			return common.Hash{}, err
		}

		if !is {
			return common.Hash{}, errStartAndEndMismatch
		}

		return *descendant, nil
	}

	// otherwise, get block on canonical chain by descendantNumber
	hash, err := s.blockState.GetHashByNumber(descendantNumber)
	if err != nil {
		return common.Hash{}, err
	}

	// check if it's a descendant of the provided ancestor hash
	is, err := s.blockState.IsDescendantOf(ancestor, hash)
	if err != nil {
		return common.Hash{}, err
	}

	if !is {
		// if it's not a descendant, search for a block that has number=descendantNumber that is
		hashes, err := s.blockState.GetAllBlocksAtNumber(descendantNumber)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to get blocks at number %d: %w", descendantNumber, err)
		}

		for _, hash := range hashes {
			is, err := s.blockState.IsDescendantOf(ancestor, hash)
			if err != nil || !is {
				continue
			}

			// this sets the descendant hash to whatever the first block we find with descendantNumber
			// is, however there might be multiple blocks that fit this criteria
			h := common.Hash{}
			copy(h[:], hash[:])
			descendant = &h
			break
		}

		if descendant == nil {
			return common.Hash{}, fmt.Errorf("%w with number %d", errFailedToGetDescendant, descendantNumber)
		}
	} else {
		// if it is, set descendant hash to our block w/ descendantNumber
		descendant = &hash
	}

	logger.Tracef("determined descendant %s with number %d and ancestor %s",
		*descendant, descendantNumber, ancestor)
	return *descendant, nil
}

func (s *Service) handleAscendingByNumber(start, end uint,
	requestedData byte) (*network.BlockResponseMessage, error) {
	var err error
	data := make([]*types.BlockData, (end-start)+1)

	for i := uint(0); start+i <= end; i++ {
		blockNumber := start + i
		data[i], err = s.getBlockDataByNumber(blockNumber, requestedData)
		if err != nil {
			return nil, err
		}
	}

	return &network.BlockResponseMessage{
		BlockData: data,
	}, nil
}

func (s *Service) handleDescendingByNumber(start, end uint,
	requestedData byte) (*network.BlockResponseMessage, error) {
	var err error
	data := make([]*types.BlockData, (start-end)+1)

	for i := uint(0); start-i >= end; i++ {
		blockNumber := start - i
		data[i], err = s.getBlockDataByNumber(blockNumber, requestedData)
		if err != nil {
			return nil, err
		}
	}

	return &network.BlockResponseMessage{
		BlockData: data,
	}, nil
}

func (s *Service) handleChainByHash(ancestor, descendant common.Hash,
	max uint, requestedData byte, direction network.SyncDirection) (
	*network.BlockResponseMessage, error) {
	subchain, err := s.blockState.Range(ancestor, descendant)
	if err != nil {
		return nil, fmt.Errorf("retrieving range: %w", err)
	}

	// If the direction is descending, prune from the start.
	// if the direction is ascending it should prune from the end.
	if uint(len(subchain)) > max {
		if direction == network.Ascending {
			subchain = subchain[:max]
		} else {
			subchain = subchain[uint(len(subchain))-max:]
		}
	}

	data := make([]*types.BlockData, len(subchain))

	for i, hash := range subchain {
		data[i], err = s.getBlockData(hash, requestedData)
		if err != nil {
			return nil, err
		}
	}

	// reverse BlockData, if descending request
	if direction == network.Descending {
		reverseBlockData(data)
	}

	return &network.BlockResponseMessage{
		BlockData: data,
	}, nil
}

func (s *Service) getBlockDataByNumber(num uint, requestedData byte) (*types.BlockData, error) {
	hash, err := s.blockState.GetHashByNumber(num)
	if err != nil {
		return nil, err
	}

	return s.getBlockData(hash, requestedData)
}

func (s *Service) getBlockData(hash common.Hash, requestedData byte) (*types.BlockData, error) {
	var err error
	blockData := &types.BlockData{
		Hash: hash,
	}

	if requestedData == 0 {
		return blockData, nil
	}

	if (requestedData & network.RequestedDataHeader) == 1 {
		blockData.Header, err = s.blockState.GetHeader(hash)
		if err != nil {
			logger.Debugf("failed to get header for block with hash %s: %s", hash, err)
		}
	}

	if (requestedData&network.RequestedDataBody)>>1 == 1 {
		blockData.Body, err = s.blockState.GetBlockBody(hash)
		if err != nil {
			logger.Debugf("failed to get body for block with hash %s: %s", hash, err)
		}
	}

	if (requestedData&network.RequestedDataReceipt)>>2 == 1 {
		retData, err := s.blockState.GetReceipt(hash)
		if err == nil && retData != nil {
			blockData.Receipt = &retData
		}
	}

	if (requestedData&network.RequestedDataMessageQueue)>>3 == 1 {
		retData, err := s.blockState.GetMessageQueue(hash)
		if err == nil && retData != nil {
			blockData.MessageQueue = &retData
		}
	}

	if (requestedData&network.RequestedDataJustification)>>4 == 1 {
		retData, err := s.blockState.GetJustification(hash)
		if err == nil && retData != nil {
			blockData.Justification = &retData
		}
	}

	return blockData, nil
}
