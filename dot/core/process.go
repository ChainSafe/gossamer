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

package core

import (
	"errors"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/transaction"
	log "github.com/ChainSafe/log15"
)

// BlockRequestMsgType 1

// ProcessBlockRequestMessage processes a block request message, returning a block response message
func (s *Service) ProcessBlockRequestMessage(msg network.Message) error {
	blockRequest := msg.(*network.BlockRequestMessage)

	var startHash common.Hash
	var endHash common.Hash

	switch c := blockRequest.StartingBlock.Value().(type) {
	case uint64:
		block, err := s.blockState.GetBlockByNumber(big.NewInt(0).SetUint64(c))
		if err != nil {
			log.Error("[core] cannot get starting block", "number", c)
			return err
		}

		startHash = block.Header.Hash()
	case common.Hash:
		startHash = c
	}

	if blockRequest.EndBlockHash.Exists() {
		endHash = blockRequest.EndBlockHash.Value()
	} else {
		endHash = s.blockState.BestBlockHash()
	}

	log.Trace("[core] got BlockRequestMessage", "startHash", startHash, "endHash", endHash)

	// get sub-chain of block hashes
	subchain := s.blockState.SubChain(startHash, endHash)

	if len(subchain) > maxResponseSize {
		subchain = subchain[:maxResponseSize]
	}

	responseData := []*types.BlockData{}

	for _, hash := range subchain {
		data, err := s.blockState.GetBlockData(hash)
		if err != nil {
			return err
		}

		blockData := new(types.BlockData)
		blockData.Hash = hash

		// TODO: checks for the existence of the following fields should be implemented once #596 is addressed.

		// header
		if blockRequest.RequestedData&1 == 1 {
			blockData.Header = data.Header
		} else {
			blockData.Header = optional.NewHeader(false, nil)
		}

		// body
		if (blockRequest.RequestedData&2)>>1 == 1 {
			blockData.Body = data.Body
		} else {
			blockData.Body = optional.NewBody(false, nil)
		}

		// receipt
		if (blockRequest.RequestedData&4)>>2 == 1 {
			blockData.Receipt = data.Receipt
		} else {
			blockData.Receipt = optional.NewBytes(false, nil)
		}

		// message queue
		if (blockRequest.RequestedData&8)>>3 == 1 {
			blockData.MessageQueue = data.MessageQueue
		} else {
			blockData.MessageQueue = optional.NewBytes(false, nil)
		}

		// justification
		if (blockRequest.RequestedData&16)>>4 == 1 {
			blockData.Justification = data.Justification
		} else {
			blockData.Justification = optional.NewBytes(false, nil)
		}

		responseData = append(responseData, blockData)
	}

	blockResponse := &network.BlockResponseMessage{
		ID:        blockRequest.ID,
		BlockData: responseData,
	}

	return s.safeMsgSend(blockResponse)
}

// BlockResponseMsgType 2

// ProcessBlockResponseMessage attempts to validate and add the block to the
// chain by calling `core_execute_block`. Valid blocks are stored in the block
// database to become part of the canonical chain.
func (s *Service) ProcessBlockResponseMessage(msg network.Message) error {
	log.Trace("[core] got BlockResponseMessage")

	blockData := msg.(*network.BlockResponseMessage).BlockData

	bestNum, err := s.blockState.BestBlockNumber()
	if err != nil {
		return err
	}

	for _, bd := range blockData {
		if bd.Header.Exists() {
			header, err := types.NewHeaderFromOptional(bd.Header)
			if err != nil {
				return err
			}

			// get block header; if exists, return
			existingHeader, err := s.blockState.GetHeader(bd.Hash)
			if err != nil && existingHeader == nil {
				err = s.blockState.SetHeader(header)
				if err != nil {
					return err
				}

				log.Info("[core] saved block header", "hash", header.Hash(), "number", header.Number)

				// TODO: handle consensus digest, if first in epoch
				// err = s.handleConsensusDigest(header)
				// if err != nil {
				// 	return err
				// }
			}
		}

		if bd.Header.Exists() && bd.Body.Exists {
			header, err := types.NewHeaderFromOptional(bd.Header)
			if err != nil {
				return err
			}

			body, err := types.NewBodyFromOptional(bd.Body)
			if err != nil {
				return err
			}

			block := &types.Block{
				Header: header,
				Body:   body,
			}

			// TODO: why doesn't execute block work with block we built?

			// blockWithoutDigests := block
			// blockWithoutDigests.Header.Digest = [][]byte{{}}

			// enc, err := block.Encode()
			// if err != nil {
			// 	return err
			// }

			// err = s.executeBlock(enc)
			// if err != nil {
			// 	log.Error("[core] failed to validate block", "err", err)
			// 	return err
			// }

			if header.Number.Cmp(bestNum) == 1 {
				err = s.blockState.AddBlock(block)
				if err != nil {
					log.Error("[core] Failed to add block to state", "error", err, "hash", header.Hash(), "parentHash", header.ParentHash)
					return err
				}

				log.Info("[core] imported block", "number", header.Number, "hash", header.Hash())

				err = s.checkForRuntimeChanges()
				if err != nil {
					return err
				}
			}
		}

		err := s.blockState.CompareAndSetBlockData(bd)
		if err != nil {
			return err
		}
	}

	return nil
}

// BlockAnnounceMsgType 3

// ProcessBlockAnnounceMessage creates a block request message from the block
// announce messages (block announce messages include the header but the full
// block is required to execute `core_execute_block`).
func (s *Service) ProcessBlockAnnounceMessage(msg network.Message) error {
	log.Trace("[core] got BlockAnnounceMessage")

	blockAnnounceMessage, ok := msg.(*network.BlockAnnounceMessage)
	if !ok {
		return errors.New("could not cast network.Message to BlockAnnounceMessage")
	}

	header, err := types.NewHeader(blockAnnounceMessage.ParentHash, blockAnnounceMessage.Number, blockAnnounceMessage.StateRoot, blockAnnounceMessage.ExtrinsicsRoot, blockAnnounceMessage.Digest)
	if err != nil {
		return err
	}

	_, err = s.blockState.GetHeader(header.Hash())
	if err != nil && err.Error() == "Key not found" {
		err = s.blockState.SetHeader(header)
		if err != nil {
			return err
		}

		log.Info("[core] saved block", "number", header.Number, "hash", header.Hash())
	} else {
		return err
	}

	bestNum, err := s.blockState.BestBlockNumber()
	if err != nil {
		log.Error("[core] BlockAnnounceMessage", "error", err)
		return err
	}

	messageBlockNumMinusOne := big.NewInt(0).Sub(blockAnnounceMessage.Number, big.NewInt(1))

	// check if we should send block request message
	if bestNum.Cmp(messageBlockNumMinusOne) == -1 {
		log.Debug("[core] sending new block to syncer", "number", blockAnnounceMessage.Number)
		s.syncChan <- blockAnnounceMessage.Number
	}

	return nil
}

// TransactionMsgType 4

// ProcessTransactionMessage validates each transaction in the message and
// adds valid transactions to the transaction queue of the BABE session
func (s *Service) ProcessTransactionMessage(msg network.Message) error {

	// get transactions from message extrinsics
	txs := msg.(*network.TransactionMessage).Extrinsics

	for _, tx := range txs {
		tx := tx // pin

		// validate each transaction
		val, err := s.ValidateTransaction(tx)
		if err != nil {
			log.Error("[core] failed to validate transaction", "err", err)
			return err // exit
		}

		// create new valid transaction
		vtx := transaction.NewValidTransaction(tx, val)

		if s.isAuthority {
			// push to the transaction queue of BABE session
			s.transactionQueue.Push(vtx)
		}
	}

	return nil
}
