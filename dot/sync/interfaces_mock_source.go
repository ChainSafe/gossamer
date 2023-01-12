// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

// readyBlocksProcessor processes ready blocks.
// it is implemented by *chainProcessor
type readyBlocksProcessor interface {
	processReadyBlocks()
	stop()
}

// disjointBlockSetInterface represents a set of incomplete blocks, or blocks
// with an unknown parent. it is implemented by *disjointBlockSet
type disjointBlockSetInterface interface {
	run(done <-chan struct{})
	addHashAndNumber(hash common.Hash, number uint) error
	addHeader(*types.Header) error
	addBlock(*types.Block) error
	addJustification(common.Hash, []byte) error
	removeBlock(common.Hash)
	removeLowerBlocks(num uint)
	getBlock(common.Hash) *pendingBlock
	getBlocks() []*pendingBlock
	getReadyDescendants(curr common.Hash, ready []*types.BlockData) []*types.BlockData
	size() int
}

// chainSyncer contains the methods used by the high-level service into the `chainSync` module
type chainSyncer interface {
	start()
	stop()

	// called upon receiving a BlockAnnounce
	setBlockAnnounce(from peer.ID, header *types.Header) error

	// called upon receiving a BlockAnnounceHandshake
	setPeerHead(p peer.ID, hash common.Hash, number uint) error

	// syncState returns the current syncing state
	syncState() chainSyncState

	// getHighestBlock returns the highest block or an error
	getHighestBlock() (highestBlock uint, err error)
}

// workHandler handles new potential work (ie. reported peer state, block announces), results from dispatched workers,
// and stored pending work (ie. pending blocks set)
// workHandler should be implemented by `bootstrapSync` and `tipSync`
type workHandler interface {
	// handleNewPeerState returns a new worker based on a peerState.
	// The worker may be nil in which case we do nothing.
	handleNewPeerState(*peerState) (*worker, error)

	// handleWorkerResult handles the result of a worker, which may be
	// nil or error. It optionally returns a new worker to be dispatched.
	handleWorkerResult(w *worker) (workerToRetry *worker, err error)

	// hasCurrentWorker is called before a worker is to be dispatched to
	// check whether it is a duplicate. this function returns whether there is
	// a worker that covers the scope of the proposed worker; if true,
	// ignore the proposed worker
	hasCurrentWorker(*worker, map[uint64]*worker) bool

	// handleTick handles a timer tick
	handleTick() ([]*worker, error)
}
