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
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	log "github.com/ChainSafe/log15"
)

// // setNextEpochDescriptor sets the epoch data for the next epoch from the data
// // included in the ConsensusDigest of the first block of each epoch
// func (s *Service) setNextEpochDescriptor(data []byte) error {

// 	// initialize epoch data interface for next epoch
// 	nextEpochData := new(babe.NextEpochDescriptor)

// 	// decode consensus digest data for next epoch
// 	err := nextEpochData.Decode(data)
// 	if err != nil {
// 		return err
// 	}

// <<<<<<< HEAD
// 	// set epoch data for next epoch
// 	return s.bs.SetEpochData(nextEpochData)
// =======
// 	// verify best block is from current epoch
// 	if !currentEpoch {
// 		return fmt.Errorf("best block is not from current epoch")
// 	}

// 	// get best epoch number from best header
// 	bestEpoch, err := s.getBlockEpoch(bestHash)
// 	if err != nil {
// 		return fmt.Errorf("failed to get epoch number for best block: %s", err)
// 	}

// 	// verify current epoch number matches best epoch number
// 	if s.epochNumber != bestEpoch {
// 		return fmt.Errorf("block epoch does not match current epoch")
// 	}

// 	// set next epoch number
// 	s.epochNumber = bestEpoch + 1

// 	// reset first block number
// 	s.firstBlock = nil

// 	return nil
// >>>>>>> 7c8c332e69b3fcc09ce1d1b3ec44e82847802a39
// }

// initializeBabeSession creates a new BABE session
func (s *Service) initializeBabeSession() (*babe.Session, error) {
	log.Debug(
		"[core] initializing BABE session...",
		"epoch", s.epochNumber,
	)

	// TODO: AuthorityData comes from NextEpochDescriptor within the ConsensusDigest
	// of the block Digest, which is included in the first block of each epoch
	authData := s.bs.AuthorityData()
	if len(authData) == 0 {
		return nil, fmt.Errorf("authority data not set")
	}

	newBlocks := make(chan types.Block)
	s.blkRec = newBlocks

	epochDone := make(chan struct{})
	s.epochDone = epochDone

	babeKill := make(chan struct{})
	s.babeKill = babeKill

	keys := s.keys.Sr25519Keypairs()

	// get best slot to determine next start slot
	bestHash := s.blockState.BestBlockHash()
	bestSlot, err := s.blockState.GetSlotForBlock(bestHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get slot for latest block: %s", err)
	}

	// BABE session configuration
	bsConfig := &babe.SessionConfig{
		Keypair:          keys[0].(*sr25519.Keypair),
		Runtime:          s.rt,
		NewBlocks:        newBlocks, // becomes block send channel in BABE session
		BlockState:       s.blockState,
		StorageState:     s.storageState,
		TransactionQueue: s.transactionQueue,
		AuthData:         authData,
		Done:             epochDone,
		Kill:             babeKill,
		StartSlot:        bestSlot + 1,
		SyncLock:         s.syncLock,
	}

	// create new BABE session
	bs, err := babe.NewSession(bsConfig)
	if err != nil {
		log.Error("[core] failed to initialize BABE session", "error", err)
		return nil, err
	}

	log.Debug(
		"[core] BABE session initialized",
		"epoch", s.epochNumber,
	)

	return bs, nil
}
