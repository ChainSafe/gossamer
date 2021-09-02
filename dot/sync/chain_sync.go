package sync

import (
	"math/big"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
)

type BlockResponseMessage = network.BlockResponseMessage

type chainSync struct {
	blockState BlockState

	bestSeenNumber *big.Int
	bestSeenHash   common.Hash

	// queue of work created by setting peer heads
	workQueue chan *peerState

	// workers are put here when they are completed so we can handle their result
	resultQueue chan *worker

	// tracks the latest state we know of from our peers,
	// ie. their best block hash and number
	peerState map[peer.ID]*peerState

	// current workers that are attempting to obtain blocks
	nextWorker uint64
	workers    map[uint64]*worker
}

type peerState struct {
	who    peer.ID
	hash   common.Hash
	number *big.Int
}

type worker struct {
	startHash    common.Hash
	startNumber  *big.Int
	targetHash   common.Hash
	targetNumber *big.Int

	duration time.Duration
	resp     *BlockResponseMessage
	err      error
}

func newChainSync(bs BlockState) *chainSync {
	return &chainSync{
		blockState: bs,
	}
}

func (cs *chainSync) setPeerHead(p peer.ID, hash common.Hash, number *big.Int) {
	cs.peerState[p] = &peerState{
		hash:   hash,
		number: number,
	}

	if number.Cmp(cs.bestSeenNumber) == 1 {
		cs.bestSeenNumber = number
		cs.bestSeenHash = hash
	}

	cs.workQueue <- cs.peerState[p]
}

func (cs *chainSync) start() {
	go cs.sync()
}

func (cs *chainSync) sync() {
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case ps := <-cs.workQueue:
			// if a peer reports a greater head than us, or a chain which
			// appears to be a fork, begin syncing
			err := cs.handleWork(ps)
			if err != nil {
				logger.Error("failed to handle chain sync work", "error", err)
			}
		case _ = <-cs.resultQueue:
			// handle results from workers
			// if success, validate the response
			// otherwise, potentially retry the worker
		case <-ticker.C:
			// bootstrap complete, switch state to idle
		}

	}
}

func (cs *chainSync) handleWork(ps *peerState) error {
	// if the peer reports a lower or equal best block number than us,
	// check if they are on a fork or not
	head, err := cs.blockState.BestBlockNumber()
	if err != nil {
		return err
	}

	if ps.number.Cmp(head) <= 0 {
		// check if our block hash for that number is the same, if so, do nothing
		hash, err := cs.blockState.GetHashByNumber(ps.number)
		if err != nil {
			return err
		}

		if hash.Equal(ps.hash) {
			return nil
		}

		// check if their best block is on an invalid chain, if it is,
		// potentially downscore them
		// for now, we can remove them from the syncing peers set
		fin, err := cs.blockState.GetHighestFinalisedHeader()
		if err != nil {
			return err
		}

		// their block hash doesn't match ours for that number (ie. they are on a different
		// chain), and also the highest finalised block is higher than that number.
		// thus the peer is on an invalid chain
		if fin.Number.Cmp(ps.number) >= 0 {
			// TODO: downscore this peer, or temporarily don't sync from them?
			delete(cs.peerState, ps.who)
		}

		return nil
	}

	return nil
}
