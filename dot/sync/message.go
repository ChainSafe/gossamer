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
	"errors"
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
func (s *Service) CreateBlockResponse(blockRequest *network.BlockRequestMessage) (*network.BlockResponseMessage, error) {
	var (
		startHash, endHash     common.Hash
		startHeader, endHeader *types.Header
		err                    error
		respSize               uint32
	)

	if blockRequest.Max != nil {
		respSize = *blockRequest.Max
		if respSize > maxResponseSize {
			respSize = maxResponseSize
		}
	} else {
		respSize = maxResponseSize
	}

	switch startBlock := blockRequest.StartingBlock.Value().(type) {
	case uint64:
		if startBlock == 0 {
			startBlock = 1
		}

		block, err := s.blockState.GetBlockByNumber(big.NewInt(0).SetUint64(startBlock)) //nolint
		if err != nil {
			return nil, fmt.Errorf("failed to get start block %d for request: %w", startBlock, err)
		}

		startHeader = &block.Header
		startHash = block.Header.Hash()
	case common.Hash:
		startHash = startBlock
		startHeader, err = s.blockState.GetHeader(startHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get start block %s for request: %w", startHash, err)
		}
	default:
		return nil, ErrInvalidBlockRequest
	}

	if blockRequest.EndBlockHash != nil {
		endHash = *blockRequest.EndBlockHash
		endHeader, err = s.blockState.GetHeader(endHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get end block %s for request: %w", endHash, err)
		}
	} else {
		endNumber := big.NewInt(0).Add(startHeader.Number, big.NewInt(int64(respSize-1)))
		bestBlockNumber, err := s.blockState.BestBlockNumber()
		if err != nil {
			return nil, fmt.Errorf("failed to get best block %d for request: %w", bestBlockNumber, err)
		}

		if endNumber.Cmp(bestBlockNumber) == 1 {
			endNumber = bestBlockNumber
		}

		endBlock, err := s.blockState.GetBlockByNumber(endNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get end block %d for request: %w", endNumber, err)
		}
		endHeader = &endBlock.Header
		endHash = endHeader.Hash()
	}

	logger.Debug("handling BlockRequestMessage", "start", startHeader.Number, "end", endHeader.Number, "startHash", startHash, "endHash", endHash)

	responseData := []*types.BlockData{}

	switch blockRequest.Direction {
	case network.Ascending:
		for i := startHeader.Number.Int64(); i <= endHeader.Number.Int64(); i++ {
			blockData, err := s.getBlockData(big.NewInt(i), blockRequest.RequestedData)
			if err != nil {
				return nil, err
			}
			responseData = append(responseData, blockData)
		}
	case network.Descending:
		for i := endHeader.Number.Int64(); i >= startHeader.Number.Int64(); i-- {
			blockData, err := s.getBlockData(big.NewInt(i), blockRequest.RequestedData)
			if err != nil {
				return nil, err
			}
			responseData = append(responseData, blockData)
		}
	default:
		return nil, errors.New("invalid BlockRequest direction")
	}

	logger.Debug("sending BlockResponseMessage", "start", startHeader.Number, "end", endHeader.Number)
	return &network.BlockResponseMessage{
		BlockData: responseData,
	}, nil
}

func (s *Service) getBlockData(num *big.Int, requestedData byte) (*types.BlockData, error) {
	hash, err := s.blockState.GetHashByNumber(num)
	if err != nil {
		return nil, err
	}

	blockData := &types.BlockData{
		Hash: hash,
	}

	if requestedData == 0 {
		return blockData, nil
	}

	if (requestedData & network.RequestedDataHeader) == 1 {
		blockData.Header, err = s.blockState.GetHeader(hash)
		if err != nil {
			logger.Debug("failed to get header for block", "number", num, "hash", hash, "error", err)
		}
	}

	if (requestedData&network.RequestedDataBody)>>1 == 1 {
		blockData.Body, err = s.blockState.GetBlockBody(hash)
		if err != nil {
			logger.Debug("failed to get body for block", "number", num, "hash", hash, "error", err)
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
