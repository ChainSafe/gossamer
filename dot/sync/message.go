// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package sync

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	// maxResponseSize is maximum number of block data a BlockResponse message can contain
	maxResponseSize = 128
)

// CreateBlockResponse creates a block response message from a block request message
func (s *Service) CreateBlockResponse(req *network.BlockRequestMessage) (*network.BlockResponseMessage, error) {
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
		startHash              *common.Hash
		endHash                = req.EndBlockHash
		startNumber, endNumber uint64
		max                    uint32 = maxResponseSize
	)

	// determine maximum response size
	if req.Max != nil && *req.Max < maxResponseSize {
		max = *req.Max
	}

	switch startBlock := req.StartingBlock.Value().(type) {
	case uint64:
		if startBlock == 0 {
			startBlock = 1
		}

		bestBlockNumber, err := s.blockState.BestBlockNumber()
		if err != nil {
			return nil, fmt.Errorf("failed to get best block %d for request: %w", bestBlockNumber, err)
		}

		// if request start is higher than our best block, return error
		if bestBlockNumber.Uint64() < startBlock {
			return nil, errRequestStartTooHigh
		}

		startNumber = startBlock

		if endHash != nil {
			// TODO: end hash is provided but start hash isn't, so we need to determine a start block
			// that is an ancestor of the end block
			sh, err := s.blockState.GetHashByNumber(big.NewInt(int64(startNumber)))
			if err != nil {
				return nil, fmt.Errorf("failed to get start block %d for request: %w", startNumber, err)
			}

			is, err := s.blockState.IsDescendantOf(sh, *endHash)
			if err != nil {
				return nil, err
			}

			if !is {
				return nil, fmt.Errorf("%w: hash=%s", errFailedToGetEndHashAncestor, *endHash)
			}

			startHash = &sh
		}
	case common.Hash:
		startHash = &startBlock

		// make sure we actually have the starting block
		header, err := s.blockState.GetHeader(*startHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get start block %s for request: %w", startHash, err)
		}

		startNumber = header.Number.Uint64()
	default:
		return nil, ErrInvalidBlockRequest
	}

	if endHash == nil {
		endNumber = startNumber + uint64(max) - 1
		bestBlockNumber, err := s.blockState.BestBlockNumber()
		if err != nil {
			return nil, fmt.Errorf("failed to get best block %d for request: %w", bestBlockNumber, err)
		}

		if endNumber > bestBlockNumber.Uint64() {
			endNumber = bestBlockNumber.Uint64()
		}
	} else {
		header, err := s.blockState.GetHeader(*endHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get end block %s: %w", *endHash, err)
		}

		endNumber = header.Number.Uint64()
	}

	// start hash provided, need to determine end hash that is descendant of start hash
	if startHash != nil {
		eh, err := s.checkOrGetDescendantHash(*startHash, endHash, big.NewInt(int64(endNumber)))
		if err != nil {
			return nil, err
		}

		endHash = &eh
	}

	if startHash == nil || endHash == nil {
		logger.Debug("handling BlockRequestMessage",
			"start", startNumber,
			"end", endNumber,
			"direction", req.Direction,
		)
		return s.handleAscendingByNumber(startNumber, endNumber, req.RequestedData)
	}

	logger.Debug("handling BlockRequestMessage",
		"start", *startHash,
		"end", *endHash,
		"direction", req.Direction,
	)
	return s.handleChainByHash(*startHash, *endHash, max, req.RequestedData, req.Direction)
}

func (s *Service) handleDescendingRequest(req *network.BlockRequestMessage) (*network.BlockResponseMessage, error) {
	var (
		startHash              *common.Hash
		endHash                = req.EndBlockHash
		startNumber, endNumber uint64
		max                    uint32 = maxResponseSize
	)

	// determine maximum response size
	if req.Max != nil && *req.Max < maxResponseSize {
		max = *req.Max
	}

	switch startBlock := req.StartingBlock.Value().(type) {
	case uint64:
		bestBlockNumber, err := s.blockState.BestBlockNumber()
		if err != nil {
			return nil, fmt.Errorf("failed to get best block %d for request: %w", bestBlockNumber, err)
		}

		// if request start is higher than our best block, only return blocks from our best block and below
		if bestBlockNumber.Uint64() < startBlock {
			startNumber = bestBlockNumber.Uint64()
		} else {
			startNumber = startBlock
		}
	case common.Hash:
		startHash = &startBlock

		// make sure we actually have the starting block
		header, err := s.blockState.GetHeader(*startHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get start block %s for request: %w", startHash, err)
		}

		startNumber = header.Number.Uint64()
	default:
		return nil, ErrInvalidBlockRequest
	}

	// end hash provided, need to determine start hash that is descendant of end hash
	if endHash != nil {
		sh, err := s.checkOrGetDescendantHash(*endHash, startHash, big.NewInt(int64(startNumber)))
		startHash = &sh
		if err != nil {
			return nil, err
		}
	}

	// end hash is not provided, calculate end by number
	if endHash == nil {
		if startNumber <= uint64(max+1) {
			endNumber = 1
		} else {
			endNumber = startNumber - uint64(max) + 1
		}

		if startHash != nil {
			// need to get blocks by subchain if start hash is provided, get end hash
			endHeader, err := s.blockState.GetHeaderByNumber(big.NewInt(int64(endNumber)))
			if err != nil {
				return nil, fmt.Errorf("failed to get end block %d for request: %w", endNumber, err)
			}

			hash := endHeader.Hash()
			endHash = &hash
		}
	}

	if startHash == nil || endHash == nil {
		logger.Debug("handling BlockRequestMessage",
			"start", startNumber,
			"end", endNumber,
			"direction", req.Direction,
		)
		return s.handleDescendingByNumber(startNumber, endNumber, req.RequestedData)
	}

	logger.Debug("handling BlockRequestMessage",
		"start", *startHash,
		"end", *endHash,
		"direction", req.Direction,
	)
	return s.handleChainByHash(*endHash, *startHash, max, req.RequestedData, req.Direction)
}

// checkOrGetDescendantHash checks if the provided `descendant` is on the same chain as the `ancestor`, if it's provided,
// otherwise, it sets `descendant` to a block with number=`descendantNumber` that is a descendant of the ancestor
// if used with an Ascending request, ancestor is the start block and descendant is the end block
// if used with an Descending request, ancestor is the end block and descendant is the start block
func (s *Service) checkOrGetDescendantHash(ancestor common.Hash, descendant *common.Hash, descendantNumber *big.Int) (common.Hash, error) {
	if descendantNumber == nil {
		return common.Hash{}, errNilDescendantNumber
	}

	// if `descendant` was provided, check that it's a descendant of `ancestor`
	if descendant != nil {
		header, err := s.blockState.GetHeader(ancestor)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to get descendant %s: %w", *descendant, err)
		}

		// if descendant number is lower than ancestor number, this is an error
		if header.Number.Cmp(descendantNumber) > 0 {
			return common.Hash{}, fmt.Errorf("invalid request, descendant number %d is higher than ancestor %d", header.Number, descendantNumber)
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

	logger.Trace("determined descendant",
		"ancestor", ancestor,
		"descendant", *descendant,
		"number", descendantNumber,
	)
	return *descendant, nil
}

func (s *Service) handleAscendingByNumber(start, end uint64, requestedData byte) (*network.BlockResponseMessage, error) {
	var err error
	data := make([]*types.BlockData, (end-start)+1)

	for i := uint64(0); start+i <= end; i++ {
		blockNumber := start + i
		data[i], err = s.getBlockDataByNumber(big.NewInt(int64(blockNumber)), requestedData)
		if err != nil {
			return nil, err
		}
	}

	return &network.BlockResponseMessage{
		BlockData: data,
	}, nil
}

func (s *Service) handleDescendingByNumber(start, end uint64, requestedData byte) (*network.BlockResponseMessage, error) {
	var err error
	data := make([]*types.BlockData, (start-end)+1)

	for i := uint64(0); start-i >= end; i++ {
		blockNumber := start - i
		data[i], err = s.getBlockDataByNumber(big.NewInt(int64(blockNumber)), requestedData)
		if err != nil {
			return nil, err
		}
	}

	return &network.BlockResponseMessage{
		BlockData: data,
	}, nil
}

func (s *Service) handleChainByHash(ancestor, descendant common.Hash, max uint32, requestedData byte, direction network.SyncDirection) (*network.BlockResponseMessage, error) {
	subchain, err := s.blockState.SubChain(ancestor, descendant)
	if err != nil {
		return nil, err
	}

	if uint32(len(subchain)) > max {
		subchain = subchain[:max]
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

func (s *Service) getBlockDataByNumber(num *big.Int, requestedData byte) (*types.BlockData, error) {
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
			logger.Debug("failed to get header for block", "hash", hash, "error", err)
		}
	}

	if (requestedData&network.RequestedDataBody)>>1 == 1 {
		blockData.Body, err = s.blockState.GetBlockBody(hash)
		if err != nil {
			logger.Debug("failed to get body for block", "hash", hash, "error", err)
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
