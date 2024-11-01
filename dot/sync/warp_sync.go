// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"slices"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/libp2p/go-libp2p/core/peer"
)

type WarpSyncPhase uint

const (
	WarpProof = iota
	TargetBlock
	Completed
)

type WarpSyncProofProvider interface {
	CurrentAuthorities() (primitives.AuthorityList, error)
	Verify(encodedProof []byte, setId primitives.SetID, authorities primitives.AuthorityList) (
		*network.WarpSyncVerificationResult, error)
}

type WarpSyncStrategy struct {
	// Strategy dependencies and config
	peers            *peerViewSet
	badBlocks        []string
	reqMaker         network.RequestMaker
	warpSyncProvider WarpSyncProofProvider
	blockState       BlockState

	// Warp sync state
	startedAt       time.Time
	phase           WarpSyncPhase
	syncedFragments int
	setId           primitives.SetID
	authorities     primitives.AuthorityList
	lastBlock       *types.Header
	result          types.BlockData
}

type WarpSyncConfig struct {
	Telemetry        Telemetry
	BadBlocks        []string
	RequestMaker     network.RequestMaker
	WarpSyncProvider WarpSyncProofProvider
	BlockState       BlockState
	Peers            *peerViewSet
}

// NewWarpSyncStrategy returns a new warp sync strategy
func NewWarpSyncStrategy(cfg *WarpSyncConfig) *WarpSyncStrategy {
	authorities, err := cfg.WarpSyncProvider.CurrentAuthorities()
	if err != nil {
		panic("failed to get current authorities")
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

// OnBlockAnnounce on every new block announce received
// Synce it is a warp sync strategy, we are going to only update the peerset reputation
// And peers target block
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
		w.peers.update(from, blockAnnounceHeaderHash, uint32(msg.Number)) //nolint:gosec
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
			response:     &messages.WarpSyncProof{},
			requestMaker: w.reqMaker,
		}
	case TargetBlock:
		req := messages.NewBlockRequest(
			*messages.NewFromBlock(lastBlock.Hash()),
			1,
			messages.RequestedDataHeader+
				messages.RequestedDataBody+
				messages.RequestedDataJustification,
			messages.Ascending,
		)
		task = SyncTask{
			request:      req,
			response:     &messages.BlockResponseMessage{},
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
		var warpProofResult *network.WarpSyncVerificationResult

		repChanges, bans, warpProofResult = w.validateWarpSyncResults(results)

		if warpProofResult != nil {
			if !warpProofResult.Completed {
				logger.Infof("[WARP SYNC] partial warp sync")

				// Partial warp proof
				w.setId = warpProofResult.SetId
				w.authorities = warpProofResult.AuthorityList
				w.lastBlock = &warpProofResult.Header
			} else {
				logger.Infof("[WARP SYNC] complete warp sync")

				w.phase = TargetBlock
				w.lastBlock = &warpProofResult.Header
			}
		}

	case TargetBlock:
		var validRes []RequestResponseData

		// Reuse same validator than in fullsync
		repChanges, bans, validRes = validateResults(results, w.badBlocks)

		// TODO: check if this can cause an issue
		w.result = *validRes[0].responseData[0]
		w.phase = Completed
	}

	return w.IsSynced(), repChanges, bans, nil
}

func (w *WarpSyncStrategy) validateWarpSyncResults(results []*SyncTaskResult) (
	repChanges []Change, peersToBlock []peer.ID, result *network.WarpSyncVerificationResult) {

	repChanges = make([]Change, 0)
	peersToBlock = make([]peer.ID, 0)
	bestProof := &messages.WarpSyncProof{}
	var bestResult *network.WarpSyncVerificationResult

	for _, result := range results {
		if !result.completed {
			repChanges = append(repChanges, Change{
				who: result.who,
				rep: peerset.ReputationChange{
					Value:  peerset.UnexpectedResponseValue,
					Reason: peerset.UnexpectedResponseReason,
				}})
			peersToBlock = append(peersToBlock, result.who)
			continue
		}

		switch response := result.response.(type) {
		case *messages.WarpSyncProof:
			if !result.completed {
				continue
			}

			// If invalid warp sync proof, then we should block the peer and update its reputation
			encodedProof, err := response.Encode()
			if err != nil {
				// This should never happen since the proof is already decoded without issues
				panic("fail to encode warp proof")
			}

			// Best proof will be the finished proof or the proof with more fragments
			res, err := w.warpSyncProvider.Verify(encodedProof, w.setId, w.authorities)

			if err != nil {
				logger.Errorf("[WARP SYNC] bad warp proof: %s", err)

				repChanges = append(repChanges, Change{
					who: result.who,
					rep: peerset.ReputationChange{
						Value:  peerset.BadWarpProofValue,
						Reason: peerset.BadWarpProofReason,
					}})
				peersToBlock = append(peersToBlock, result.who)

				return repChanges, peersToBlock, nil
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

	return repChanges, peersToBlock, bestResult
}

func (w *WarpSyncStrategy) ShowMetrics() {
	totalSyncSeconds := time.Since(w.startedAt).Seconds()

	fps := float64(w.syncedFragments) / totalSyncSeconds
	logger.Infof("⏩ Warping, downloading finality proofs, fragments %d, best %x "+
		"took: %.2f seconds, fps: %.2f fragments/second",
		w.syncedFragments, w.lastBlock.Number, totalSyncSeconds, fps)
}

func (w *WarpSyncStrategy) IsSynced() bool {
	return w.phase == Completed
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
