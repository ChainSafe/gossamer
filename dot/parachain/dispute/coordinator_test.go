package dispute

import (
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	disputetypes "github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"testing"
)

type TestState struct {
	validators        []keystore.KeyPair
	validatorPublic   []parachaintypes.ValidatorID
	validatorGroups   [][]parachaintypes.ValidatorIndex
	masterKeystore    keystore.Keystore
	subsystemKeystore keystore.Keystore
	headers           map[common.Hash]types.Header
	blockNumToHeader  map[uint32]common.Hash
	lastBlock         common.Hash
	knownSession      *parachaintypes.SessionIndex
	db                database.Database
	overseer          chan any
	runtime           *MockRuntimeInstance
}

func newTestState(t *testing.T) *TestState {
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	pair1, err := sr25519.NewKeypairFromMnenomic("//Polka", "")
	require.NoError(t, err)
	pair2, err := sr25519.NewKeypairFromMnenomic("//Dot", "")
	require.NoError(t, err)
	pair3, err := sr25519.NewKeypairFromMnenomic("//Kusama", "")
	require.NoError(t, err)

	validators := []keystore.KeyPair{
		kr.KeyAlice,
		kr.KeyBob,
		kr.KeyCharlie,
		kr.KeyDave,
		kr.KeyEve,
		kr.KeyFerdie,
		kr.KeyGeorge,
		// Two more keys needed so disputes are not confirmed already with only 3 statements.
		pair1,
		pair2,
		pair3,
	}

	var validatorPublic []parachaintypes.ValidatorID
	for _, v := range validators {
		validatorPublic = append(validatorPublic, parachaintypes.ValidatorID(v.Public().Encode()))
	}

	validatorGroups := [][]parachaintypes.ValidatorIndex{
		{0, 1},
		{2, 3},
		{4, 5, 6},
	}

	masterKeyStore := keystore.NewBasicKeystore("master", crypto.Sr25519Type)
	for _, v := range validators {
		err = masterKeyStore.Insert(v)
		require.NoError(t, err)
	}

	subsystemKeyStore := keystore.NewBasicKeystore("subsystem", crypto.Sr25519Type)
	err = subsystemKeyStore.Insert(kr.KeyAlice)
	require.NoError(t, err)

	db, err := database.NewPebble("test", true)
	require.NoError(t, err)

	genesisHeader := types.Header{
		ParentHash:     common.Hash{},
		Number:         0,
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Digest:         types.NewDigest(),
	}
	lastBlock := genesisHeader.Hash()

	headers := make(map[common.Hash]types.Header)
	blockNumToHeader := make(map[uint32]common.Hash)
	headers[lastBlock] = genesisHeader
	blockNumToHeader[0] = lastBlock

	return &TestState{
		validators:        validators,
		validatorPublic:   validatorPublic,
		validatorGroups:   validatorGroups,
		masterKeystore:    masterKeyStore,
		subsystemKeystore: subsystemKeyStore,
		headers:           headers,
		blockNumToHeader:  blockNumToHeader,
		lastBlock:         lastBlock,
		knownSession:      nil,
		db:                db,
		overseer:          make(chan any, 1),
	}
}

func (ts *TestState) sessionInfo() *parachaintypes.SessionInfo {
	var (
		discoveryKeys  []parachaintypes.AuthorityDiscoveryID
		assignmentKeys []parachaintypes.AssignmentID

		validatorIndices []parachaintypes.ValidatorIndex
	)

	for i, v := range ts.validatorPublic {
		discoveryKeys = append(discoveryKeys, parachaintypes.AuthorityDiscoveryID(v))
		assignmentKeys = append(assignmentKeys, parachaintypes.AssignmentID(v))

		validatorIndices = append(validatorIndices, parachaintypes.ValidatorIndex(i))
	}

	return &parachaintypes.SessionInfo{
		ActiveValidatorIndices:  validatorIndices,
		RandomSeed:              [32]byte{0},
		DisputePeriod:           6,
		Validators:              ts.validatorPublic,
		DiscoveryKeys:           discoveryKeys,
		AssignmentKeys:          assignmentKeys,
		ValidatorGroups:         ts.validatorGroups,
		NCores:                  uint32(len(ts.validatorGroups)),
		ZerothDelayTrancheWidth: 0,
		RelayVRFModuloSamples:   1,
		NDelayTranches:          100,
		NoShowSlots:             1,
		NeededApprovals:         10,
	}
}

func (ts *TestState) activateLeafAtSession(t *testing.T,
	session parachaintypes.SessionIndex,
	blockNumber uint,
	candidateEvents []parachaintypes.CandidateEvent,
) {
	require.True(t, blockNumber > 0)

	blockHeader := types.Header{
		ParentHash:     ts.lastBlock,
		Number:         blockNumber,
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Digest:         types.NewDigest(),
	}
	blockHash := blockHeader.Hash()

	ts.headers[blockHash] = blockHeader
	ts.blockNumToHeader[uint32(blockHeader.Number)] = blockHash
	ts.lastBlock = blockHash

	t.Log("activating block")

	activatedLeaf := overseer.ActivatedLeaf{
		Hash:   blockHash,
		Number: uint32(blockNumber),
		Status: overseer.LeafStatusFresh,
	}
	err := sendMessage(ts.overseer, overseer.Signal[overseer.ActiveLeavesUpdate]{
		Data: overseer.ActiveLeavesUpdate{Activated: &activatedLeaf},
	})
	require.NoError(t, err)

	ts.mockSyncQueries(t, blockHash, session, candidateEvents)
}

func (ts *TestState) mockSyncQueries(t *testing.T,
	blockHash common.Hash,
	session parachaintypes.SessionIndex,
	candidateEvents []parachaintypes.CandidateEvent,
) []disputetypes.UncheckedDisputeMessage {
	var (
		gotSessionInformation bool
		//	gotScrapingInformation bool
	)

	var sentDisputes []disputetypes.UncheckedDisputeMessage

	ts.runtime.EXPECT().ParachainHostSessionIndexForChild(gomock.Any()).DoAndReturn(func(arg0 common.Hash) (parachaintypes.SessionIndex, error) {
		require.False(t, gotSessionInformation, "session info already retrieved")

		gotSessionInformation = true
		require.Equal(t, blockHash, arg0)

		if ts.knownSession == nil {
			ts.knownSession = &session
		}

		return session, nil
	})

	ts.runtime.EXPECT().ParachainHostSessionInfo(gomock.Any(), gomock.Any()).
		DoAndReturn(func(arg0 parachaintypes.SessionIndex, arg1 common.Hash) (*parachaintypes.SessionInfo, error) {
			require.True(t, arg0 < session)
			require.Equal(t, blockHash, arg1)

			return ts.sessionInfo(), nil
		})

	ts.runtime.EXPECT().ParachainHostCandidateEvents(gomock.Any()).DoAndReturn(func(arg0 common.Hash) ([]parachaintypes.CandidateEvent, error) {
		return candidateEvents, nil
	})

	ts.runtime.EXPECT().ParachainHostOnChainVotes(gomock.Any()).DoAndReturn(func(arg0 common.Hash) (*parachaintypes.ScrapedOnChainVotes, error) {
		return &parachaintypes.ScrapedOnChainVotes{
			Session: session,
		}, nil
	})

	//ts.sender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg any) error {
	//	switch message := msg.(type) {
	//	case overseer.Signal[overseer.Block]:
	//		require.False(t, gotScrapingInformation, "scraping info already retrieved")
	//		gotScrapingInformation = true
	//		msg.ResponseChannel <- 0
	//
	//	case msg.Communication.BlockNumber != nil:
	//		msg.ResponseChannel <- uint32(ts.headers[blockHash].Number)
	//
	//	case msg.DistributionMessage != nil:
	//		sentDisputes = append(sentDisputes, *msg.DistributionMessage)
	//
	//	}
	//})

	return sentDisputes
}

func (ts *TestState) mockResumeSyncWithEvents(t *testing.T,
	session *parachaintypes.SessionIndex,
	initialEvents []parachaintypes.CandidateEvent,
) []disputetypes.UncheckedDisputeMessage {
	leaves := make([]common.Hash, len(ts.headers))
	for leaf := range ts.headers {
		leaves = append(leaves, leaf)
	}

	var messages []disputetypes.UncheckedDisputeMessage
	for i, leaf := range leaves {
		activatedLeaf := overseer.ActivatedLeaf{
			Hash:   leaf,
			Number: uint32(i),
			Status: overseer.LeafStatusFresh,
		}
		err := sendMessage(ts.overseer, overseer.Signal[overseer.ActiveLeavesUpdate]{
			Data: overseer.ActiveLeavesUpdate{Activated: &activatedLeaf},
		})
		require.NoError(t, err)

		var events []parachaintypes.CandidateEvent
		if i == 1 {
			events = initialEvents
		}

		newMessages := ts.mockSyncQueries(t, leaf, *session, events)
		messages = append(messages, newMessages...)
	}

	return messages
}

func TestDisputesCoordinator(t *testing.T) {
	t.Run("too_many_unconfirmed_statements_are_considered_spam", func(t *testing.T) {

	})
}
