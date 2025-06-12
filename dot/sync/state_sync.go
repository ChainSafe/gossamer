// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"fmt"
	"slices"
	"time"

	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/grandpa/warpsync"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/libp2p/go-libp2p/core/peer"
)

type StateSyncPhase uint

const (
	DownloadState = iota
	DownloadFirstBlock
	StateSyncCompleted
)

type StateSyncStrategy struct {
	// Strategy dependencies and config
	peers                *peerViewSet
	badBlocks            []string
	reqMaker             network.RequestMaker
	blockReqMaker        network.RequestMaker
	blockState           state.BlockState
	grandpaState         GrandpaState
	epochState           EpochState
	storage              StorageState
	stateRequestProvider *StateRequestProvider
	finalityGadget       FinalityGadget
	blockImporter        importer
	blockImportHandler   BlockImportHandler

	// State sync state
	phase          WarpSyncPhase
	startedAt      time.Time
	warpSyncResult warpsync.WarpSyncVerificationResult
	targetHeader   types.Header
	firstBlock     types.BlockData
}

type StateSyncStrategyConfig struct {
	Telemetry          Telemetry
	BadBlocks          []string
	BlockState         state.BlockState
	GrandpaState       GrandpaState
	EpochState         EpochState
	Peers              *peerViewSet
	ReqMaker           network.RequestMaker
	BlockReqMaker      network.RequestMaker
	WarpSyncResult     warpsync.WarpSyncVerificationResult
	StateStorage       StorageState
	FinalityGadget     FinalityGadget
	TransactionState   TransactionState
	BabeVerifier       BabeVerifier
	BlockImportHandler BlockImportHandler
}

func NewStateSyncStrategy(
	cfg *StateSyncStrategyConfig,
) *StateSyncStrategy {
	targetHeader := cfg.WarpSyncResult.Header

	return &StateSyncStrategy{
		peers:              cfg.Peers,
		badBlocks:          cfg.BadBlocks,
		blockImportHandler: cfg.BlockImportHandler,
		blockState:         cfg.BlockState,
		grandpaState:       cfg.GrandpaState,
		epochState:         cfg.EpochState,
		targetHeader:       targetHeader,
		warpSyncResult:     cfg.WarpSyncResult,
		reqMaker:           cfg.ReqMaker,
		blockReqMaker:      cfg.BlockReqMaker,
		storage:            cfg.StateStorage,
		finalityGadget:     cfg.FinalityGadget,
		// TODO: we can assume that v1 is right for every chain but we need to find a way to set the right state version
		stateRequestProvider: NewStateRequestProvider(targetHeader.Hash(), trie.V1),
		phase:                DownloadState,
		blockImporter: newBlockImporter(&BlockImporterConfig{
			BlockState:         cfg.BlockState,
			StorageState:       cfg.StateStorage,
			TransactionState:   cfg.TransactionState,
			BabeVerifier:       cfg.BabeVerifier,
			FinalityGadget:     cfg.FinalityGadget,
			BlockImportHandler: cfg.BlockImportHandler,
			Telemetry:          cfg.Telemetry,
		}),
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
	repChanges = make([]Change, 0)
	peersToBlock = make([]peer.ID, 0)

	switch s.phase {
	case DownloadState:
		for _, result := range results {
			switch response := result.response.(type) {
			case *messages.StateResponse:
				logger.Debugf("Retrieving state data from %s with %d keys",
					result.who, len(response.Entries))

				completed, err := s.stateRequestProvider.ProcessResponse(response)
				if err != nil {
					switch err {
					case ErrEmptyStateEntries:
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

				if completed {
					s.phase = DownloadFirstBlock
				}

				return false, repChanges, peersToBlock, nil
			default:
				logger.Warnf("unexpected response type %T for state request, banning peer: %s", response, result.who)
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
	case DownloadFirstBlock:
		var validRes []RequestResponseData

		// Reuse same validator than in fullsync
		repChanges, peersToBlock, validRes = validateResults(results, s.badBlocks)

		if len(validRes) > 0 && len(validRes[0].responseData) > 0 {
			s.firstBlock = *validRes[0].responseData[0]
			s.phase = StateSyncCompleted
		}
		return true, repChanges, peersToBlock, s.setBlockAsFullSyncStartingBlock()
	}

	return s.IsSynced(), repChanges, peersToBlock, nil
}

// NextActions returns the next actions to be taken by the sync service
func (s *StateSyncStrategy) NextActions() ([]*SyncTask, error) {
	s.startedAt = time.Now()

	task := &SyncTask{}

	switch s.phase {
	case DownloadState:
		task = &SyncTask{
			request:      s.stateRequestProvider.BuildRequest(),
			response:     &messages.StateResponse{},
			requestMaker: s.reqMaker,
		}
	case DownloadFirstBlock:
		task =
			&SyncTask{
				request: messages.NewBlockRequest(
					*messages.NewFromBlock(uint(1)),
					1,
					messages.RequestedDataHeader+
						messages.RequestedDataBody+
						messages.RequestedDataJustification,
					messages.Ascending),
				response:     &messages.BlockResponseMessage{},
				requestMaker: s.blockReqMaker,
			}
	}

	return []*SyncTask{task}, nil
}

func (s *StateSyncStrategy) ShowStatus() {
	switch s.phase {
	case DownloadState:
		if len(s.stateRequestProvider.GetLastKeys()) == 0 {
			return
		}

		lastKey := s.stateRequestProvider.GetLastKeys()[0]
		cursor := float32(lastKey[0])
		percentDone := cursor / 256 * 100

		shortHash := fmt.Sprintf("0x%x...%x", lastKey[:2], lastKey[len(lastKey)-2:])

		logger.Infof("⚙️ State Sync, downloading state %.2f%%, last key: (%s)", percentDone, shortHash)
	case DownloadFirstBlock:
		logger.Infof("⚙️ State Sync, retrieving block #1")
	case StateSyncCompleted:
		logger.Infof("⚙️ State Sync, completed!")
	}
}

func (w *StateSyncStrategy) Result() any {
	logger.Debug("unexpected call to Result() in StateSyncStrategy")
	return nil
}

func (s *StateSyncStrategy) IsSynced() bool {
	return s.phase == StateSyncCompleted
}

// TODO: this is quick and dirty, we must find a better way to do this, maybe executing block importer
// with some extra logic to skip validations
func (s *StateSyncStrategy) setBlockAsFullSyncStartingBlock() error {
	// Importing first block to set epochs
	slotNumber, err := s.firstBlock.Header.SlotNumber()
	if err != nil {
		return fmt.Errorf("getting slot number, err: %w", err)
	}

	err = s.blockState.SetFirstNonOriginSlotNumber(slotNumber)
	if err != nil {
		return fmt.Errorf("setting non origin slot number, err: %w", err)
	}

	// Get download trie state
	trieState, err := s.stateRequestProvider.BuildTrie()
	if err != nil {
		return fmt.Errorf("building retrieved state, err: %w", err)
	}

	// Check state is the expected
	if trieState.MustHash() != s.targetHeader.StateRoot {
		return fmt.Errorf("state root mismatch: got %s expected %s", trieState.MustHash(), s.targetHeader.StateRoot)
	}

	// Store downloaded trie state
	storageTrie := storage.NewInMemoryTrieState(trieState)
	err = s.storage.StoreTrie(storageTrie, &s.targetHeader)
	if err != nil {
		return fmt.Errorf("storing new state trie, err: %w", err)
	}

	// Set new runtime based on genesis runtime configuration
	genesisHeader, err := s.blockState.BestBlockHeader()
	if err != nil {
		return fmt.Errorf("getting genesis header, err: %w", err)
	}

	genesisRuntime, err := s.blockState.GetRuntime(genesisHeader.Hash())
	if err != nil {
		return fmt.Errorf("getting genesis runtime, err: %w", err)
	}

	codeHash, err := s.storage.LoadCodeHash(nil)
	if err != nil {
		return fmt.Errorf("getting genesis runtime code hash, err: %w", err)
	}

	rtCfg := wazero_runtime.Config{
		LogLvl:      genesisRuntime.LogLvl(),
		Storage:     storage.NewInMemoryTrieState(trieState),
		Keystore:    genesisRuntime.Keystore(),
		NodeStorage: genesisRuntime.NodeStorage(),
		Network:     genesisRuntime.NetworkService(),
		Role:        genesisRuntime.Role(),
		CodeHash:    codeHash,
	}

	instance, err := wazero_runtime.NewInstanceFromTrie(trieState, rtCfg)
	if err != nil {
		return fmt.Errorf("creating new runtime, err: %w", err)
	}

	// Initialize runtime and set it in the new blocktree
	blockTree := blocktree.NewBlockTreeFromRoot(&s.targetHeader)

	blockTree.StoreRuntime(s.targetHeader.Hash(), instance)
	s.blockState.SetBlockTree(blockTree)

	// Set block header in block state
	err = s.blockState.SetHeader(&s.targetHeader)
	if err != nil {
		return fmt.Errorf("setting new block header, err: %w", err)
	}

	// Update grandpa state with latest authorities
	err = s.grandpaState.SetAuthorities(uint64(s.warpSyncResult.SetId),
		grandpaVotersFromAuthorities(s.warpSyncResult.AuthorityList))
	if err != nil {
		return fmt.Errorf("setting new authorities set: %w", err)
	}

	// Configure babe epoch
	err = s.configureBabeEpoch(trieState)
	if err != nil {
		return fmt.Errorf("configuring babe epoch: %w", err)
	}

	// Finalize block
	justification := s.warpSyncResult.Justification
	err = s.blockState.SetFinalisedHash(s.targetHeader.Hash(),
		justification.Justification.Round, uint64(s.warpSyncResult.SetId), false)
	if err != nil {
		return fmt.Errorf("setting finalised hash: %w", err)
	}

	encodedJustification, err := scale.Marshal(justification)
	if err != nil {
		return fmt.Errorf("encoding justification %w", err)
	}

	err = s.blockState.SetJustification(s.targetHeader.Hash(), encodedJustification)
	if err != nil {
		return fmt.Errorf("setting justification for block %s: %w", s.targetHeader.Hash(), err)
	}

	// Handle header digests
	err = s.blockImportHandler.HandleDigests(&s.targetHeader)
	if err != nil {
		return fmt.Errorf("handling header digests %w", err)
	}

	return nil
}

func grandpaVotersFromAuthorities(authorities grandpa.AuthorityList) []types.GrandpaVoter {
	voters := make([]types.GrandpaVoter, len(authorities))

	for _, auth := range authorities {
		voters = append(voters, types.GrandpaVoter{
			Key: ed25519.PublicKey(auth.AuthorityID[:]),
			ID:  uint64(auth.AuthorityWeight),
		})
	}

	return voters
}

func (s *StateSyncStrategy) configureBabeEpoch(newState trie.Trie) error {
	epochDataRaw, err := babe.GetNextEpochDataRawFromState(newState)
	if err != nil {
		return fmt.Errorf("getting epoch data from state: %w", err)
	}

	epochIndex, err := babe.GetCurrentEpochIndexFromState(newState)
	if err != nil {
		return fmt.Errorf("getting epoch index from state: %w", err)
	}

	return s.epochState.SetEpochDataRaw(epochIndex+1, epochDataRaw)
}

var _ Strategy = (*StateSyncStrategy)(nil)
