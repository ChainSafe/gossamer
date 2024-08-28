// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"bytes"
	"errors"
	"fmt"
	"slices"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

const maxNumberOfSameRequestPerPeer uint = 2

var (
	ErrInvalidBlockRequest     = errors.New("invalid block request")
	errMaxNumberOfSameRequest  = errors.New("max number of same request reached")
	errInvalidRequestDirection = errors.New("invalid request direction")
	errRequestStartTooHigh     = errors.New("request start number is higher than our best block")
	errStartAndEndNotOnChain   = errors.New("request start and end hash are not on the same chain")
	errFailedToGetDescendant   = errors.New("failed to find descendant block")
)

// CreateBlockResponse creates a block response message from a block request message
func (s *SyncService) CreateBlockResponse(from peer.ID, req *messages.BlockRequestMessage) (
	*messages.BlockResponseMessage, error) {
	logger.Debugf("sync request from %s: %s", from, req.String())

	if !req.StartingBlock.IsUint32() && !req.StartingBlock.IsHash() {
		return nil, ErrInvalidBlockRequest
	}

	encodedRequest, err := req.Encode()
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	encodedKey := bytes.Join([][]byte{[]byte(from.String()), encodedRequest}, nil)
	requestHash, err := common.Blake2bHash(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("hashing encoded block request sync message: %w", err)
	}

	numOfRequests := s.seenBlockSyncRequests.Get(requestHash)

	if numOfRequests > maxNumberOfSameRequestPerPeer {
		s.network.ReportPeer(peerset.ReputationChange{
			Value:  peerset.SameBlockSyncRequest,
			Reason: peerset.SameBlockSyncRequestReason,
		}, from)

		logger.Debugf("max number of same request reached by: %s", from.String())
		return nil, fmt.Errorf("%w: %s", errMaxNumberOfSameRequest, from.String())
	}

	s.seenBlockSyncRequests.Put(requestHash, numOfRequests+1)

	switch req.Direction {
	case messages.Ascending:
		return s.handleAscendingRequest(req)
	case messages.Descending:
		return s.handleDescendingRequest(req)
	default:
		return nil, fmt.Errorf("%w: %v", errInvalidRequestDirection, req.Direction)
	}
}

func (s *SyncService) handleAscendingRequest(req *messages.BlockRequestMessage) (*messages.BlockResponseMessage, error) {
	var (
		max         uint = messages.MaxBlocksInResponse
		startHash   *common.Hash
		startNumber uint
	)

	// determine maximum response size
	if req.Max != nil && *req.Max < messages.MaxBlocksInResponse {
		max = uint(*req.Max)
	}

	bestBlockNumber, err := s.blockState.BestBlockNumber()
	if err != nil {
		return nil, fmt.Errorf("getting best block for request: %w", err)
	}

	if req.StartingBlock.IsHash() {
		startingBlockHash := req.StartingBlock.Hash()
		startHash = &startingBlockHash

		// make sure we actually have the starting block
		header, err := s.blockState.GetHeader(startingBlockHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get start block %s for request: %w", startHash, err)
		}

		startNumber = header.Number
	} else if req.StartingBlock.IsUint32() {
		startBlock := req.StartingBlock.Uint32()
		if startBlock == 0 {
			startBlock = 1
		}

		// if request start is higher than our best block, return error
		if bestBlockNumber < uint(startBlock) {
			return nil, errRequestStartTooHigh
		}

		startNumber = uint(startBlock)
	} else {
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
		logger.Debugf("handling block request: direction %s, "+
			"start block number: %d, "+
			"end block number: %d",
			req.Direction, startNumber, endNumber)

		return s.handleAscendingByNumber(startNumber, endNumber, req.RequestedData)
	}

	logger.Debugf("handling block request: direction %s, "+
		"start block hash: %s, "+
		"end block hash: %s",
		req.Direction, *startHash, *endHash)

	return s.handleChainByHash(*startHash, *endHash, max, req.RequestedData, req.Direction)
}

func (s *SyncService) handleDescendingRequest(req *messages.BlockRequestMessage) (*messages.BlockResponseMessage, error) {
	var (
		startHash   *common.Hash
		startNumber uint
		max         uint = messages.MaxBlocksInResponse
	)

	// determine maximum response size
	if req.Max != nil && *req.Max < messages.MaxBlocksInResponse {
		max = uint(*req.Max)
	}

	if req.StartingBlock.IsHash() {
		startingBlockHash := req.StartingBlock.Hash()
		startHash = &startingBlockHash

		// make sure we actually have the starting block
		header, err := s.blockState.GetHeader(*startHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get start block %s for request: %w", startHash, err)
		}

		startNumber = header.Number
	} else if req.StartingBlock.IsUint32() {
		startBlock := req.StartingBlock.Uint32()
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
	} else {
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
		logger.Debugf("handling BlockRequestMessage with direction %s "+
			"from start block with number %d to end block with number %d",
			req.Direction, startNumber, endNumber)
		return s.handleDescendingByNumber(startNumber, endNumber, req.RequestedData)
	}

	logger.Debugf("handling block request message with direction %s "+
		"from start block with hash %s to end block with hash %s",
		req.Direction, *startHash, *endHash)
	return s.handleChainByHash(*endHash, *startHash, max, req.RequestedData, req.Direction)
}

// checkOrGetDescendantHash checks if the provided `descendant` is
// on the same chain as the `ancestor`, if it's provided, otherwise
// it sets `descendant` to a block with number=`descendantNumber` that is a descendant of the ancestor.
// If used with an Ascending request, ancestor is the start block and descendant is the end block
// If used with an Descending request, ancestor is the end block and descendant is the start block
func (s *SyncService) checkOrGetDescendantHash(ancestor common.Hash,
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
			return common.Hash{}, errStartAndEndNotOnChain
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

func (s *SyncService) handleAscendingByNumber(start, end uint,
	requestedData byte) (*messages.BlockResponseMessage, error) {
	var err error
	data := make([]*types.BlockData, (end-start)+1)

	for i := uint(0); start+i <= end; i++ {
		blockNumber := start + i
		data[i], err = s.getBlockDataByNumber(blockNumber, requestedData)
		if err != nil {
			return nil, err
		}
	}

	return &messages.BlockResponseMessage{
		BlockData: data,
	}, nil
}

func (s *SyncService) handleDescendingByNumber(start, end uint,
	requestedData byte) (*messages.BlockResponseMessage, error) {
	var err error
	data := make([]*types.BlockData, (start-end)+1)

	for i := uint(0); start-i >= end; i++ {
		blockNumber := start - i
		data[i], err = s.getBlockDataByNumber(blockNumber, requestedData)
		if err != nil {
			return nil, err
		}
	}

	return &messages.BlockResponseMessage{
		BlockData: data,
	}, nil
}

func (s *SyncService) handleChainByHash(ancestor, descendant common.Hash,
	max uint, requestedData byte, direction messages.SyncDirection) (
	*messages.BlockResponseMessage, error) {
	subchain, err := s.blockState.Range(ancestor, descendant)
	if err != nil {
		return nil, fmt.Errorf("retrieving range: %w", err)
	}

	// If the direction is descending, prune from the start.
	// if the direction is ascending it should prune from the end.
	if uint(len(subchain)) > max {
		if direction == messages.Ascending {
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
	if direction == messages.Descending {
		slices.Reverse(data)
	}

	return &messages.BlockResponseMessage{
		BlockData: data,
	}, nil
}

func (s *SyncService) getBlockDataByNumber(num uint, requestedData byte) (*types.BlockData, error) {
	hash, err := s.blockState.GetHashByNumber(num)
	if err != nil {
		return nil, err
	}

	return s.getBlockData(hash, requestedData)
}

func (s *SyncService) getBlockData(hash common.Hash, requestedData byte) (*types.BlockData, error) {
	var err error
	blockData := &types.BlockData{
		Hash: hash,
	}

	if requestedData == 0 {
		return blockData, nil
	}

	if (requestedData & messages.RequestedDataHeader) == 1 {
		blockData.Header, err = s.blockState.GetHeader(hash)
		if err != nil {
			logger.Debugf("failed to get header for block with hash %s: %s", hash, err)
		}
	}

	if (requestedData&messages.RequestedDataBody)>>1 == 1 {
		blockData.Body, err = s.blockState.GetBlockBody(hash)
		if err != nil {
			logger.Debugf("failed to get body for block with hash %s: %s", hash, err)
		}
	}

	if (requestedData&messages.RequestedDataReceipt)>>2 == 1 {
		retData, err := s.blockState.GetReceipt(hash)
		if err == nil && retData != nil {
			blockData.Receipt = &retData
		}
	}

	if (requestedData&messages.RequestedDataMessageQueue)>>3 == 1 {
		retData, err := s.blockState.GetMessageQueue(hash)
		if err == nil && retData != nil {
			blockData.MessageQueue = &retData
		}
	}

	if (requestedData&messages.RequestedDataJustification)>>4 == 1 {
		retData, err := s.blockState.GetJustification(hash)
		if err == nil && retData != nil {
			blockData.Justification = &retData
		}
	}

	return blockData, nil
}
