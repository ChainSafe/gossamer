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

package network

import (
	"math/big"
	mrand "math/rand"
	"time"

	"github.com/ChainSafe/gossamer/lib/common/optional"

	log "github.com/ChainSafe/log15"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/exp/rand"
)

// blockSync submodule
type blockSync struct {
	host              *host
	blockState        BlockState
	requestedBlockIDs map[uint64]bool // track requested block id messages
}

// newBlockSync creates a new blockSync instance from the host
func newBlockSync(host *host, blockState BlockState) *blockSync {
	return &blockSync{
		host:              host,
		blockState:        blockState,
		requestedBlockIDs: make(map[uint64]bool),
	}
}

// addRequestedBlockID adds a requested block id to non-persistent state
func (bs *blockSync) addRequestedBlockID(blockID uint64) {
	bs.requestedBlockIDs[blockID] = true
	log.Trace("[network] Block added to blockSync", "block", blockID)
}

// hasRequestedBlockID returns true if the block id has been requested
func (bs *blockSync) hasRequestedBlockID(blockID uint64) bool {
	hasBeenRequested := bs.requestedBlockIDs[blockID]
	log.Trace("[network] Check block request in blockSync", "block", blockID, "requested", hasBeenRequested)
	return hasBeenRequested
}

// removeRequestedBlockID removes a requested block id from non-persistent state
func (bs *blockSync) removeRequestedBlockID(blockID uint64) {
	delete(bs.requestedBlockIDs, blockID)
	log.Trace("[network] Block removed from blockSync", "block", blockID)
}

// handleStatusMesssage sends a block request message if peer best block
// number is greater than host best block number
func (bs *blockSync) handleStatusMesssage(peer peer.ID, statusMessage *StatusMessage) {

	// get latest block header from block state
	latestHeader, err := bs.blockState.BestBlockHeader()
	if err != nil {
		log.Error("[network] Failed to get best block header from block state", "error", err)
		return
	}

	bestBlockNum := big.NewInt(int64(statusMessage.BestBlockNumber))

	// check if peer block number is greater than host block number
	if latestHeader.Number.Cmp(bestBlockNum) == -1 {

		// generate random ID
		s1 := rand.NewSource(uint64(time.Now().UnixNano()))
		seed := rand.New(s1).Uint64()
		randomID := mrand.New(mrand.NewSource(int64(seed))).Uint64()

		// store requested block ids in blockSync submodule (non-persistent state)
		bs.addRequestedBlockID(randomID)

		currentHash := latestHeader.Hash()

		blockRequestMessage := &BlockRequestMessage{
			ID:            randomID, // random
			RequestedData: 3,        // block body
			StartingBlock: append([]byte{0}, currentHash[:]...),
			EndBlockHash:  optional.NewHash(true, latestHeader.Hash()),
			Direction:     1,
			Max:           optional.NewUint32(false, 0),
		}

		// send block request message
		err := bs.host.send(peer, blockRequestMessage)
		if err != nil {
			log.Error("[network] Failed to send block request message to peer", "error", err)
		}
	}
}
