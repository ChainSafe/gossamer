package sync

import (
	"slices"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

type StateSyncStrategy struct {
	// Strategy dependencies and config
	peers            *peerViewSet
	badBlocks        []string
	warpSyncReqMaker network.RequestMaker
	syncReqMaker     network.RequestMaker
	warpSyncProvider WarpSyncProofProvider
	blockState       BlockState

	// State sync state
	startedAt   time.Time
	targetBlock common.Hash
	lastKeys    [][]byte
	skipProof   bool
}

// OnBlockAnnounce on every new block announce received.
// we are going to only update the peerset reputation and peers target block.
func (s *StateSyncStrategy) OnBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) (
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

	if slices.Contains(s.badBlocks, blockAnnounceHeaderHash.String()) {
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
		s.peers.update(from, blockAnnounceHeaderHash, uint32(msg.Number)) //nolint:gosec
	}

	return &Change{
		who: from,
		rep: peerset.ReputationChange{
			Value:  peerset.GossipSuccessValue,
			Reason: peerset.GossipSuccessReason,
		},
	}, nil
}

func (s *StateSyncStrategy) OnBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	s.peers.update(from, msg.BestBlockHash, msg.BestBlockNumber)
	return nil
}

func (s *StateSyncStrategy) Process(results []*SyncTaskResult) (
	done bool, repChanges []Change, peersToBlock []peer.ID, err error) {

	repChanges = make([]Change, 0)
	peersToBlock = make([]peer.ID, 0)

	for _, result := range results {
		switch response := result.response.(type) {
		case *messages.StateResponse:
			logger.Infof("Importing state data from %s with %s keys",
				result.who, len(response.Entries))

			if len(response.Entries) == 0 && len(response.Proof) == 0 {
				logger.Infof("Bad state response")
				peersToBlock = append(peersToBlock, result.who)
				repChanges = append(repChanges, Change{
					who: result.who,
					rep: peerset.ReputationChange{
						Value:  peerset.BadStateValue,
						Reason: peerset.BadStateReason,
					},
				})
				continue
			}

			if !s.skipProof && len(response.Proof) == 0 {
				logger.Infof("Missing proof")
				peersToBlock = append(peersToBlock, result.who)
				repChanges = append(repChanges, Change{
					who: result.who,
					rep: peerset.ReputationChange{
						Value:  peerset.BadStateValue,
						Reason: peerset.BadStateReason,
					},
				})
				continue
			}

			if !s.skipProof {
				logger.Infof("Importing state from %d trie nodes", len(response.Proof))
				proofSize := len(response.Proof)
				var compactProof trie.CompactProof
				err := scale.Unmarshal(response.Proof, &compactProof)
				if err != nil {
					logger.Infof("Error decoding proof: %w", err)
					repChanges = append(repChanges, Change{
						who: result.who,
						rep: peerset.ReputationChange{
							Value:  peerset.BadStateValue,
							Reason: peerset.BadStateReason,
						},
					})
					continue
				}

				
			}

		default:
			logger.Warnf("unexpected response type %T", response)
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

	return s.IsSynced(), repChanges, peersToBlock, nil
}

func (s *StateSyncStrategy) importState(response messages.StateResponse) error {
	panic("not implemented")
}

// NextActions returns the next actions to be taken by the sync service
func (s *StateSyncStrategy) NextActions() ([]*SyncTask, error) {
	s.startedAt = time.Now()

	task := &SyncTask{
		request:      messages.NewStateRequest(s.targetBlock, s.lastKeys, s.noProof),
		response:     &messages.WarpSyncProof{},
		requestMaker: s.warpSyncReqMaker,
	}

	return []*SyncTask{task}, nil
}

func (w *StateSyncStrategy) ShowMetrics() {
	panic("not implemented")
}

func (w *StateSyncStrategy) Result() any {
	panic("not implemented")
}

func (s *StateSyncStrategy) IsSynced() bool {
	panic("not implemented")
}

var _ Strategy = (*StateSyncStrategy)(nil)
