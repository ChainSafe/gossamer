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
	}
}

func (ts *TestState) handleResumeSyncWithEvents(t *testing.T,
	session *parachaintypes.SessionIndex,
	initialEvents []parachaintypes.CandidateEvent,
) []disputetypes.UncheckedDisputeMessage {
	ctrl := gomock.NewController(t)
	sender := NewMockSender(ctrl)

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
		err := sender.SendMessage(MuxedMessage{
			Signal: &overseer.Signal{
				ActiveLeaves: &overseer.ActiveLeavesUpdate{Activated: &activatedLeaf},
			},
		})
		require.NoError(t, err)

		var events []parachaintypes.CandidateEvent
		if i == 1 {
			events = initialEvents
		}

		newMessages := ts.handleSyncQueries(t, leaf, session, events)
		messages = append(messages, newMessages...)
	}

	return messages
}

func TestDisputesCoordinator(t *testing.T) {
	t.Run("too_many_unconfirmed_statements_are_considered_spam", func(t *testing.T) {

	})
}
