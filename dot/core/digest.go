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
	"fmt"

	"github.com/ChainSafe/gossamer/dot/core/types"
	//"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"

	log "github.com/ChainSafe/log15"
)

// finalizeBabeSession finalizes the BABE session by ensuring the first block
// was set and first block and epoch number are reset for the next epoch
func (s *Syncer) finalizeBabeEpoch() error {

	// check if first block was set for current epoch
	if s.firstBlock == nil {

		// TODO: NextEpochDescriptor is included in first block of an epoch #662
		// return fmt.Errorf("first block not set for current epoch")

		log.Error("[core] first block not set for current epoch") // TODO: remove
	}

	// get epoch number for best block
	bestHash := s.blockState.BestBlockHash()
	currentEpoch, err := s.blockFromCurrentEpoch(bestHash)
	if err != nil {
		return fmt.Errorf("failed to check best block from current epoch: %s", err)
	}

	// verify best block is from current epoch
	if !currentEpoch {
		return fmt.Errorf("best block is not from current epoch")
	}

	// get best epoch number from best header
	bestEpoch, err := s.getBlockEpoch(bestHash)
	if err != nil {
		return fmt.Errorf("failed to get epoch number for best block: %s", err)
	}

	// verify current epoch number matches best epoch number
	if s.currentEpoch() != bestEpoch {
		return fmt.Errorf("block epoch does not match current epoch")
	}

	// set next epoch number
	//s.epochNumber = bestEpoch + 1
	s.verificationManager.IncrementEpoch()

	// reset first block number
	s.firstBlock = nil

	return nil
}

// handleBlockDigest checks if the provided header is the block header for
// the first block of the current epoch, finds and decodes the consensus digest
// item from the block digest, and then sets the epoch data for the next epoch
func (s *Syncer) checkForConsensusDigest(header *types.Header) (err error) {

	// check if block header digest items exist
	if header.Digest == nil || len(header.Digest) == 0 {
		return fmt.Errorf("header digest is not set")
	}

	// declare digest item
	var item types.DigestItem

	// decode each digest item and check its type
	for _, digest := range header.Digest {
		item, err = types.DecodeDigestItem(digest)
		if err != nil {
			return err
		}

		// check if digest item is consensus digest type
		if item.Type() == types.ConsensusDigestType {

			// cast decoded consensus digest item to conensus digest type
			consensusDigest, ok := item.(*types.ConsensusDigest)
			if !ok {
				return errors.New("failed to cast DigestItem to ConsensusDigest")
			}

			err = s.handleConsensusDigest(header, consensusDigest)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// handleConsensusDigest checks if the first block has been set for the current
// epoch, if first block has not been set or the block header has a lower block
// number, sets epoch data for the next epoch using data in the ConsensusDigest
func (s *Syncer) handleConsensusDigest(header *types.Header, digest *types.ConsensusDigest) error {

	// // check if block header is from current epoch
	// currentEpoch, err := s.blockFromCurrentEpoch(header.Hash())
	// if err != nil {
	// 	return fmt.Errorf("failed to check if block header is from current epoch: %s", err)
	// } else if !currentEpoch {
	// 	return fmt.Errorf("block header is not from current epoch")
	// }

	inCurrentEpoch, err := s.blockFromCurrentEpoch(header.Hash())
	if err != nil {
		return err
	}

	// check if first block has been set for current epoch
	if inCurrentEpoch && s.firstBlock != nil {

		// check if block header has lower block number than current first block
		if header.Number.Cmp(s.firstBlock.Number) >= 0 {

			// error if block header does not have lower block number
			return fmt.Errorf("first block already set for current epoch")
		}

		// either BABE produced two first blocks or we received invalid first block from connected peer
		log.Warn("[core] received first block header with lower block number than current first block")
	}

	// TODO: set this after creating next session
	// err = s.setNextEpochDescriptor(digest.Data)
	// if err != nil {
	// 	return fmt.Errorf("failed to set next epoch descriptor: %s", err)
	// }

	if inCurrentEpoch {
		// set first block in current epoch
		s.firstBlock = header
	}

	return nil
}

// getBlockEpoch gets the epoch number using the provided block hash
func (s *Syncer) getBlockEpoch(hash common.Hash) (epoch uint64, err error) {

	// get slot number to determine epoch number
	slot, err := s.blockState.GetSlotForBlock(hash)
	if err != nil {
		return epoch, fmt.Errorf("failed to get slot from block hash: %s", err)
	}

	if slot != 0 {
		// epoch number = (slot - genesis slot) / epoch length
		epoch = (slot - 1) / 6 // TODO: use epoch length from babe or core config
	}

	return epoch, nil
}

// blockFromCurrentEpoch verifies the provided block hash is from current epoch
func (s *Syncer) blockFromCurrentEpoch(hash common.Hash) (bool, error) {

	// get epoch number of block header
	epoch, err := s.getBlockEpoch(hash)
	if err != nil {
		return false, fmt.Errorf("[core] failed to get epoch from block header: %s", err)
	}

	// check if block epoch number matches current epoch number
	if epoch != s.currentEpoch() {
		return false, nil
	}

	return true, nil
}
