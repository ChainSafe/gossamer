// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"fmt"
	"slices"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/lib/grandpa/warpsync"
	"github.com/libp2p/go-libp2p/core/peer"
)

type WarpSyncPhase uint

const (
	WarpProof = iota
	WarpSyncCompleted
)

type WarpSyncProofProvider interface {
	CurrentAuthorities() (primitives.AuthorityList, error)
	Verify(encodedProof []byte, setId primitives.SetID, authorities primitives.AuthorityList) (
		*warpsync.WarpSyncVerificationResult, error)
}

type WarpSyncStrategy struct {
	// Strategy dependencies and config
	peers            *peerViewSet
	badBlocks        []string
	reqMaker         network.RequestMaker
	warpSyncProvider WarpSyncProofProvider
	blockState       state.BlockState

	// Warp sync state
	startedAt       time.Time
	phase           WarpSyncPhase
	syncedFragments int
	setId           primitives.SetID
	authorities     primitives.AuthorityList
	lastBlock       *types.Header
	result          warpsync.WarpSyncVerificationResult
}

type WarpSyncConfig struct {
	Telemetry        Telemetry
	BadBlocks        []string
	RequestMaker     network.RequestMaker
	WarpSyncProvider WarpSyncProofProvider
	BlockState       state.BlockState
	Peers            *peerViewSet
}

// NewWarpSyncStrategy returns a new warp sync strategy
func NewWarpSyncStrategy(cfg *WarpSyncConfig) *WarpSyncStrategy {
	authorities, err := cfg.WarpSyncProvider.CurrentAuthorities()
	if err != nil {
		panic(fmt.Sprintf("failed to get current authorities %s", err))
	}

	return &WarpSyncStrategy{
		warpSyncProvider: cfg.WarpSyncProvider,
		blockState:       cfg.BlockState,
		badBlocks:        cfg.BadBlocks,
		reqMaker:         cfg.RequestMaker,
		peers:            cfg.Peers,
		setId:            0,
		authorities:      authorities,
	}
}

// OnBlockAnnounce on every new block announce received.
// Since it is a warp sync strategy, we are going to only update the peerset reputation
// And peers target block.
func (w *WarpSyncStrategy) OnBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) (
	repChange *Change, err error) {

	blockAnnounceHeaderHash, err := msg.Hash()
	if err != nil {
		return nil, err
	}

	logger.Debugf("received block announce from %s: #%d (%s) best block: %v",
		from,
		msg.Number,
		blockAnnounceHeaderHash,
		msg.BestBlock,
	)

	if slices.Contains(w.badBlocks, blockAnnounceHeaderHash.String()) {
		logger.Debugf("bad block received from %s: #%d (%s) is a bad block",
			from, msg.Number, blockAnnounceHeaderHash)

		return &Change{
			who: from,
			rep: peerset.ReputationChange{
				Value:  peerset.BadBlockAnnouncementValue,
				Reason: peerset.BadBlockAnnouncementReason,
			},
		}, errBadBlockReceived
	}

	if msg.BestBlock {
		w.peers.update(from, blockAnnounceHeaderHash, uint32(msg.Number))
	}

	return &Change{
		who: from,
		rep: peerset.ReputationChange{
			Value:  peerset.GossipSuccessValue,
			Reason: peerset.GossipSuccessReason,
		},
	}, nil
}

func (w *WarpSyncStrategy) OnBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	w.peers.update(from, msg.BestBlockHash, msg.BestBlockNumber)
	return nil
}

// NextActions returns the next actions to be taken by the sync service
func (w *WarpSyncStrategy) NextActions() ([]*SyncTask, error) {
	w.startedAt = time.Now()

	lastBlock, err := w.lastBlockHeader()
	if err != nil {
		return nil, err
	}

	var task SyncTask
	switch w.phase {
	case WarpProof:
		task = SyncTask{
			request:      messages.NewWarpProofRequest(lastBlock.Hash()),
			response:     &warpsync.WarpSyncProof{},
			requestMaker: w.reqMaker,
		}
	}

	return []*SyncTask{&task}, nil
}

// Process processes the results of the sync tasks, getting the best warp sync response and
// Updating our block state
func (w *WarpSyncStrategy) Process(results []*SyncTaskResult) (
	done bool, repChanges []Change, bans []peer.ID, err error) {

	switch w.phase {
	case WarpProof:
		logger.Debug("processing warp sync proof results")

		var warpProofResult *warpsync.WarpSyncVerificationResult

		repChanges, bans, warpProofResult = w.validateWarpSyncResults(results)

		if warpProofResult != nil {
			w.lastBlock = &warpProofResult.Header

			if !warpProofResult.Completed {
				logger.Debug("partial warp sync proof received")

				w.setId = warpProofResult.SetId
				w.authorities = warpProofResult.AuthorityList
			} else {
				logger.Debugf("⏩ Warping, finish processing proofs, downloading target block #%d (%s)",
					w.lastBlock.Number, w.lastBlock.Hash().String())
				w.result = *warpProofResult
				w.phase = WarpSyncCompleted
			}
		}
	case WarpSyncCompleted:
		logger.Debug("Warp sync completed")
	}

	return w.IsSynced(), repChanges, bans, nil
}

func (w *WarpSyncStrategy) validateWarpSyncResults(results []*SyncTaskResult) (
	repChanges []Change, peersToBlock []peer.ID, result *warpsync.WarpSyncVerificationResult) {

	repChanges = make([]Change, 0)
	peersToBlock = make([]peer.ID, 0)
	bestProof := &warpsync.WarpSyncProof{}
	var bestResult *warpsync.WarpSyncVerificationResult

	for _, result := range results {
		switch response := result.response.(type) {
		case *warpsync.WarpSyncProof:
			if !result.completed {
				continue
			}

			// If invalid warp sync proof, then we should block the peer and update its reputation
			encodedProof, err := response.Encode()
			if err != nil {
				// This should never happen since the proof is already decoded without issues
				panic("fail to encode warp proof")
			}

			res, err := w.warpSyncProvider.Verify(encodedProof, w.setId, w.authorities)

			if err != nil {
				logger.Debugf("bad warp proof response: %s", err)

				repChanges = append(repChanges, Change{
					who: result.who,
					rep: peerset.ReputationChange{
						Value:  peerset.BadWarpProofValue,
						Reason: peerset.BadWarpProofReason,
					}})
				peersToBlock = append(peersToBlock, result.who)
				continue
			}

			if response.IsFinished || len(response.Proofs) > len(bestProof.Proofs) {
				bestProof = response
				bestResult = res
			}
		default:
			repChanges = append(repChanges, Change{
				who: result.who,
				rep: peerset.ReputationChange{
					Value:  peerset.UnexpectedResponseValue,
					Reason: peerset.UnexpectedResponseReason,
				}})
			peersToBlock = append(peersToBlock, result.who)
			continue
		}
	}

	w.syncedFragments += len(bestProof.Proofs)

	return repChanges, peersToBlock, bestResult
}

func (w *WarpSyncStrategy) ShowStatus() {
	switch w.phase {
	case WarpProof:
		totalSyncSeconds := time.Since(w.startedAt).Seconds()

		logger.Infof("⏩ Warping, downloading finality proofs, fragments %d, best #%d (%s) "+
			"took: %.2f seconds",
			w.syncedFragments, w.lastBlock.Number, w.lastBlock.Hash().Short(), totalSyncSeconds)
	case WarpSyncCompleted:
		logger.Infof("⏩ Warping, completed")
	}
}

func (w *WarpSyncStrategy) IsSynced() bool {
	return w.phase == WarpSyncCompleted
}

func (w *WarpSyncStrategy) Result() any {
	return w.result
}

func (w *WarpSyncStrategy) lastBlockHeader() (header *types.Header, err error) {
	if w.lastBlock == nil {
		w.lastBlock, err = w.blockState.GetHighestFinalisedHeader()
		if err != nil {
			return nil, err
		}
	}
	return w.lastBlock, nil
}

var _ Strategy = (*WarpSyncStrategy)(nil)
