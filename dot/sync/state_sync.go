package sync

import (
	"slices"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/libp2p/go-libp2p/core/peer"
)

type StateSyncStrategy struct {
	// Strategy dependencies and config
	peers      *peerViewSet
	badBlocks  []string
	reqMaker   network.RequestMaker
	blockState BlockState

	// State sync state
	startedAt   time.Time
	targetBlock common.Hash
	lastKeys    [][]byte
	completed   bool
	state       trie.Trie
}

// TODO: handle merkle proofs

type StateSyncStrategyConfig struct {
	Telemetry   Telemetry
	BadBlocks   []string
	BlockState  BlockState
	Peers       *peerViewSet
	ReqMaker    network.RequestMaker
	TargetBlock common.Hash
}

func NewStateSyncStrategy(
	cfg *StateSyncStrategyConfig,
) *StateSyncStrategy {
	return &StateSyncStrategy{
		peers:       cfg.Peers,
		badBlocks:   cfg.BadBlocks,
		blockState:  cfg.BlockState,
		targetBlock: cfg.TargetBlock,
		reqMaker:    cfg.ReqMaker,
		state:       inmemory.NewEmptyTrie(),
	}

	// TODO: set right state version
	// state.SetVersion(version)
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

			if len(response.Entries) == 0 {
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

			if len(s.lastKeys) == 2 && len(response.Entries[0].StateEntries) == 0 {
				s.lastKeys = s.lastKeys[:len(s.lastKeys)-1]
			} else {
				s.lastKeys = [][]byte{}
			}

			for _, stateEntry := range response.Entries {
				if !stateEntry.Complete {
					lastItemInResponse := stateEntry.StateEntries[len(stateEntry.StateEntries)-1]
					s.lastKeys = append(s.lastKeys, lastItemInResponse.Key)
					s.completed = false
				} else {
					s.completed = true
				}
			}

			return s.completed, repChanges, peersToBlock, s.importState(*response)
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

// importState imports the retreived state into our block state
func (s *StateSyncStrategy) importState(response messages.StateResponse) error {
	for _, stateEntry := range response.Entries {
		for _, kv := range stateEntry.StateEntries {
			if err := s.state.Put(kv.Key, kv.Value); err != nil {
				return err
			}
		}
	}

	return nil
	// TODO: if we retrieved all the state flush in-memory trie to our block state
}

// NextActions returns the next actions to be taken by the sync service
func (s *StateSyncStrategy) NextActions() ([]*SyncTask, error) {
	s.startedAt = time.Now()

	task := &SyncTask{
		request:      messages.NewStateRequest(s.targetBlock, s.lastKeys, true),
		response:     &messages.WarpSyncProof{},
		requestMaker: s.reqMaker,
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
	return s.completed
}

var _ Strategy = (*StateSyncStrategy)(nil)
