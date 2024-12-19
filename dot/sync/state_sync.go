package sync

import (
	"fmt"
	"slices"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/libp2p/go-libp2p/core/peer"
)

type StateStorage interface {
	StoreTrie(ts *storage.TrieState, header *types.Header) error
}

// TODO: re use or create a similar struct than StateRequestProvider in retrieve_state.go
type StateSyncStrategy struct {
	// Strategy dependencies and config
	peers                *peerViewSet
	badBlocks            []string
	reqMaker             network.RequestMaker
	blockState           BlockState
	storage              StateStorage
	stateRequestProvider *StateRequestProvider

	// State sync state
	startedAt   time.Time
	targetBlock types.Header
	completed   bool
}

type StateSyncStrategyConfig struct {
	Telemetry    Telemetry
	BadBlocks    []string
	BlockState   BlockState
	Peers        *peerViewSet
	ReqMaker     network.RequestMaker
	TargetBlock  types.Header
	StateStorage StateStorage
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
		storage:     cfg.StateStorage,
		// TODO: set right state version
		stateRequestProvider: NewStateRequestProvider(cfg.TargetBlock.Hash(), trie.V1),
	}
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
		s.peers.update(from, blockAnnounceHeaderHash, uint32(msg.Number))
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
	// TODO: handle merkle proofs
	repChanges = make([]Change, 0)
	peersToBlock = make([]peer.ID, 0)

	for _, result := range results {
		switch response := result.response.(type) {
		case *messages.StateResponse:
			logger.Debugf("Retrieving state data from %s with %s keys",
				result.who, len(response.Entries))

			s.completed, err = s.stateRequestProvider.ProcessResponse(response)
			if err != nil {
				switch err {
				case errEmptyStateEntries:
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
			}

			if s.completed {
				return true, repChanges, peersToBlock, s.importState()
			}

			return false, repChanges, peersToBlock, nil

		default:
			logger.Warnf("unexpected response type %T for state request, banning peer: %s", result.who, response)
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

// importState imports the retreived state into our state storage
func (s *StateSyncStrategy) importState() error {
	// Store state in our state storage
	trieState, err := s.stateRequestProvider.BuildTrie()
	if err != nil {
		return err
	}

	if trieState.MustHash() != s.targetBlock.StateRoot {
		return fmt.Errorf("state root mismatch: got %s expected %s", trieState.MustHash(), s.targetBlock.StateRoot)
	}

	storageTrie := storage.NewTrieState(trieState)
	return s.storage.StoreTrie(storageTrie, &s.targetBlock)
}

// NextActions returns the next actions to be taken by the sync service
func (s *StateSyncStrategy) NextActions() ([]*SyncTask, error) {
	s.startedAt = time.Now()

	task := &SyncTask{
		request:      s.stateRequestProvider.BuildRequest(),
		response:     &messages.WarpSyncProof{},
		requestMaker: s.reqMaker,
	}

	return []*SyncTask{task}, nil
}

func (s *StateSyncStrategy) ShowMetrics() {
	cursor := int32(s.stateRequestProvider.GetLastKeys()[0][0])
	percentDone := cursor * 100 / 256

	logger.Infof("⚙️ State Sync, downloading state %d% ", percentDone)
}

func (w *StateSyncStrategy) Result() any {
	logger.Debug("unexpected call to Result() in StateSyncStrategy")
	return nil
}

func (s *StateSyncStrategy) IsSynced() bool {
	return s.completed
}

var _ Strategy = (*StateSyncStrategy)(nil)
