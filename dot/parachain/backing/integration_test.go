package backing_test

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	candidatevalidation "github.com/ChainSafe/gossamer/dot/parachain/candidate-validation"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	"github.com/ChainSafe/gossamer/dot/parachain/overseer"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/erasure"
	"github.com/ChainSafe/gossamer/lib/keystore"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

// register the backing subsystem, run backing subsystem, start overseer
func initBackingAndOverseerMock(t *testing.T, ctx context.Context, cancel context.CancelFunc,
) (*backing.CandidateBacking, *overseer.MockableOverseer) {
	t.Helper()

	overseerMock := overseer.NewMockableOverseer(t, ctx, cancel)

	backing := backing.New(overseerMock.SubsystemsToOverseer)
	backing.OverseerToSubSystem = overseerMock.RegisterSubsystem(backing)
	backing.SubSystemToOverseer = overseerMock.GetSubsystemToOverseerChannel()

	backing.Keystore = keystore.NewBasicKeystore("test", crypto.Sr25519Type)

	overseerMock.Start()

	return backing, overseerMock
}

func getDummyHash(t *testing.T, num byte) common.Hash {
	t.Helper()

	hash := common.Hash{}
	for i := 0; i < 32; i++ {
		hash[i] = num
	}
	return hash
}

func dummyPVD(t *testing.T) parachaintypes.PersistedValidationData {
	t.Helper()

	return parachaintypes.PersistedValidationData{
		ParentHead:             parachaintypes.HeadData{Data: []byte{7, 8, 9}},
		RelayParentNumber:      0,
		RelayParentStorageRoot: getDummyHash(t, 0),
		MaxPovSize:             1024,
	}
}

func makeErasureRoot(
	t *testing.T,
	numOfValidators uint,
	pov parachaintypes.PoV,
	pvd parachaintypes.PersistedValidationData,
) common.Hash {
	t.Helper()

	availableData := availabilitystore.AvailableData{
		PoV:            pov,
		ValidationData: pvd,
	}

	dataBytes, err := scale.Marshal(availableData)
	require.NoError(t, err)

	chunks, err := erasure.ObtainChunks(numOfValidators, dataBytes)
	require.NoError(t, err)

	trie, err := erasure.ChunksToTrie(chunks)
	require.NoError(t, err)

	root, err := trie.Hash()
	require.NoError(t, err)

	return root
}

// newCommittedCandidate creates a new committed candidate receipt for testing purposes.
func newCommittedCandidate(
	t *testing.T,
	paraID uint32,
	headData parachaintypes.HeadData,
	povHash, relayParent, erasureRoot, pvdHash common.Hash,
	validationCode parachaintypes.ValidationCode,
) parachaintypes.CommittedCandidateReceipt {
	t.Helper()

	var collatorID parachaintypes.CollatorID
	var collatorSignature parachaintypes.CollatorSignature

	headDataHash, err := headData.Hash()
	require.NoError(t, err)

	validationCodeHash, err := common.Blake2bHash(validationCode)
	require.NoError(t, err)

	return parachaintypes.CommittedCandidateReceipt{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      paraID,
			RelayParent:                 relayParent,
			Collator:                    collatorID,
			PersistedValidationDataHash: pvdHash,
			PovHash:                     povHash,
			ErasureRoot:                 erasureRoot,
			Signature:                   collatorSignature,
			ParaHead:                    headDataHash,
			ValidationCodeHash:          parachaintypes.ValidationCodeHash(validationCodeHash),
		},
		Commitments: parachaintypes.CandidateCommitments{
			UpwardMessages:            []parachaintypes.UpwardMessage{},
			HorizontalMessages:        []parachaintypes.OutboundHrmpMessage{},
			NewValidationCode:         nil,
			HeadData:                  headData,
			ProcessedDownwardMessages: 0,
			HrmpWatermark:             0,
		},
	}
}

// parachainValidators returns a list of parachain validator IDs for testing purposes.
func parachainValidators(t *testing.T, ks keystore.Keystore) []parachaintypes.ValidatorID {
	t.Helper()

	validatorIds := make([]parachaintypes.ValidatorID, 0, 6)

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	keyPairs := []keystore.KeyPair{
		keyring.Alice(),
		keyring.Bob(),
		keyring.Charlie(),
		keyring.Dave(),
		keyring.Eve(),
		keyring.Ferdie(),
	}

	for _, kp := range keyPairs {
		var validatorID parachaintypes.ValidatorID

		ks.Insert(kp)
		bytes := kp.Public().Encode()
		copy(validatorID[:], bytes)
		validatorIds = append(validatorIds, validatorID)
	}

	return validatorIds
}

// validatorGroups returns validator groups for testing purposes.
func validatorGroups(t *testing.T) *parachaintypes.ValidatorGroups {
	t.Helper()

	validatorGroups := parachaintypes.ValidatorGroups{
		Validators: [][]parachaintypes.ValidatorIndex{
			{2, 0, 3, 5},
			{1},
		},
		GroupRotationInfo: parachaintypes.GroupRotationInfo{
			SessionStartBlock:      0,
			GroupRotationFrequency: 100,
			Now:                    1,
		},
	}

	return &validatorGroups
}

// availabilityCores returns a list of availability cores for testing purposes.
func availabilityCores(t *testing.T) []parachaintypes.CoreState {
	t.Helper()

	cores := parachaintypes.NewAvailabilityCores()

	core1 := parachaintypes.CoreState{}
	core1.SetValue(parachaintypes.ScheduledCore{ParaID: 1})

	core2 := parachaintypes.CoreState{}
	core2.SetValue(parachaintypes.ScheduledCore{ParaID: 2})

	cores = append(cores, core1, core2)
	return cores
}

func TestSecondsValidCandidate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	candidateBacking, overseer := initBackingAndOverseerMock(t, ctx, cancel)
	defer overseer.Stop()

	wg := new(sync.WaitGroup)
	paraValidators := parachainValidators(t, candidateBacking.Keystore)
	numOfValidators := uint(len(paraValidators))
	relayParent := getDummyHash(t, 5)
	paraID := uint32(1)

	ctrl := gomock.NewController(t)
	mockBlockState := backing.NewMockBlockState(ctrl)
	mockRuntime := backing.NewMockInstance(ctrl)
	mockImplicitView := backing.NewMockImplicitView(ctrl)

	candidateBacking.BlockState = mockBlockState
	candidateBacking.ImplicitView = mockImplicitView

	// mock BlockState methods
	mockBlockState.EXPECT().GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).
		Return(mockRuntime, nil).Times(4)

	// mock Runtime Instance methods
	mockRuntime.EXPECT().ParachainHostAsyncBackingParams().
		Return(nil, wazero_runtime.ErrExportFunctionNotFound)
	mockRuntime.EXPECT().ParachainHostSessionIndexForChild().
		Return(parachaintypes.SessionIndex(1), nil).Times(3)
	mockRuntime.EXPECT().ParachainHostValidators().
		Return(paraValidators, nil)
	mockRuntime.EXPECT().ParachainHostValidatorGroups().
		Return(validatorGroups(t), nil)
	mockRuntime.EXPECT().ParachainHostAvailabilityCores().
		Return(availabilityCores(t), nil)
	mockRuntime.EXPECT().ParachainHostMinimumBackingVotes().
		Return(backing.LEGACY_MIN_BACKING_VOTES, nil)
	mockRuntime.EXPECT().
		ParachainHostSessionExecutorParams(gomock.AssignableToTypeOf(parachaintypes.SessionIndex(0))).
		Return(nil, wazero_runtime.ErrExportFunctionNotFound).Times(2)

	//mock ImplicitView
	mockImplicitView.EXPECT().AllAllowedRelayParents().
		Return([]common.Hash{})

	pov1 := parachaintypes.PoV{BlockData: []byte{42, 43, 44}}
	pvd1 := dummyPVD(t)
	validationCode1 := parachaintypes.ValidationCode{1, 2, 3}

	pov1Hash, err := pov1.Hash()
	require.NoError(t, err)

	pvd1Hash, err := pvd1.Hash()
	require.NoError(t, err)

	candidate1 := newCommittedCandidate(
		t,
		paraID,
		parachaintypes.HeadData{},
		pov1Hash,
		relayParent,
		makeErasureRoot(t, numOfValidators, pov1, pvd1),
		pvd1Hash,
		validationCode1,
	)

	// to make entry in perRelayParent map
	overseer.ReceiveMessage(parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{Hash: relayParent, Number: 1},
	})

	// mocked for invalid second message
	mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.AssignableToTypeOf(common.Hash{})).
		Return(&validationCode1, nil)

	wg.Add(1)
	// mock the actions of overseer messages
	go func(subToOverseer chan any, t *testing.T) {
		defer wg.Done()

		// message we receive from candidate backing subsystem to overseer
		var msg any

		// handle ValidateFromExhaustive message
		select {
		case <-time.After(2 * time.Minute):
			t.Error("timed out waiting for candidatevalidation.ValidateFromExhaustive message")
			overseer.Stop()
			return
		case msg = <-subToOverseer:
			validateFromExhaustive, ok := msg.(candidatevalidation.ValidateFromExhaustive)
			if !ok {
				overseer.Stop()
				t.Error("Should be true")
				return
			}

			badReturn := candidatevalidation.BadReturn
			validateFromExhaustive.Ch <- parachaintypes.OverseerFuncRes[candidatevalidation.ValidationResult]{
				Data: candidatevalidation.ValidationResult{
					InvalidResult: &badReturn,
				},
			}
		}

		// handle collator protocol Invalid message
		select {
		case <-time.After(2 * time.Minute):
			t.Error("timed out waiting for collatorprotocolmessages.Invalid message")
			overseer.Stop()
		case msg = <-subToOverseer:
			// reported to collator protocol about invalid candidate
			_, ok := msg.(collatorprotocolmessages.Invalid)
			if !ok {
				overseer.Stop()
				t.Error("Should be true")
			}
		}
	}(overseer.SubsystemsToOverseer, t)

	// receive second message from overseer to candidate backing subsystem
	overseer.ReceiveMessage(
		backing.SecondMessage{
			RelayParent:             relayParent,
			CandidateReceipt:        candidate1.ToPlain(),
			PersistedValidationData: pvd1,
			PoV:                     pov1,
		})
	wg.Wait()

	pov2 := parachaintypes.PoV{BlockData: []byte{45, 46, 47}}

	pov2Hash, err := pov2.Hash()
	require.NoError(t, err)

	pvd2 := dummyPVD(t)
	pvd2.ParentHead.Data = []byte{14, 15, 16}
	pvd2.MaxPovSize = pvd2.MaxPovSize / 2

	pvd2Hash, err := pvd2.Hash()
	require.NoError(t, err)

	validationCode2 := parachaintypes.ValidationCode{4, 5, 6}

	candidate2 := newCommittedCandidate(
		t,
		paraID,
		parachaintypes.HeadData{Data: []byte{4, 5, 6}},
		pov2Hash,
		relayParent,
		makeErasureRoot(t, numOfValidators, pov2, pvd2),
		pvd2Hash,
		validationCode2,
	)

	// mocked for valid second message
	mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.AssignableToTypeOf(common.Hash{})).
		Return(&validationCode2, nil)

	wg.Add(1)
	// mock the actions of overseer messages
	go func(subToOverseer chan any, t *testing.T) {
		defer wg.Done()

		var msg any

		// handle ValidateFromExhaustive message
		select {
		case <-time.After(2 * time.Minute):
			t.Error("timed out waiting for ValidateFromExhaustive message")
			overseer.Stop()
			return
		case msg = <-subToOverseer:
			validateFromExhaustive, ok := msg.(candidatevalidation.ValidateFromExhaustive)
			if !ok {
				overseer.Stop()
				t.Error("Should be true")
				return
			}

			validateFromExhaustive.Ch <- parachaintypes.OverseerFuncRes[candidatevalidation.ValidationResult]{
				Data: candidatevalidation.ValidationResult{
					ValidResult: &candidatevalidation.ValidValidationResult{
						CandidateCommitments: parachaintypes.CandidateCommitments{
							UpwardMessages:            []parachaintypes.UpwardMessage{},
							HorizontalMessages:        []parachaintypes.OutboundHrmpMessage{},
							NewValidationCode:         nil,
							HeadData:                  candidate2.Commitments.HeadData,
							ProcessedDownwardMessages: 0,
							HrmpWatermark:             0,
						},
						PersistedValidationData: pvd2,
					},
				},
			}
		}

		// handle availabilitystore.StoreAvailableData message
		select {
		case <-time.After(2 * time.Minute):
			t.Error("timed out waiting for availabilitystore.StoreAvailableData message")
			overseer.Stop()
			return
		case msg = <-subToOverseer:
			store, ok := msg.(availabilitystore.StoreAvailableData)
			if !ok {
				overseer.Stop()
				t.Error("Should be true")
				return
			}
			store.Sender <- nil
		}

		// handle parachaintypes.StatementDistributionMessageShare message
		select {
		case <-time.After(2 * time.Minute):
			t.Error("timed out waiting for parachaintypes.StatementDistributionMessageShare message")
			overseer.Stop()
			return
		case msg = <-subToOverseer:
			// we have seconded a candidate and shared the statement to peers
			share, ok := msg.(parachaintypes.StatementDistributionMessageShare)
			if !ok {
				overseer.Stop()
				t.Error("Should be true")
				return
			}

			statement, err := share.SignedFullStatementWithPVD.SignedFullStatement.Payload.Value()
			require.NoError(t, err)

			if !(requireEqual(t, statement, parachaintypes.Seconded(candidate2)) &&
				requireEqual(t, *share.SignedFullStatementWithPVD.PersistedValidationData, pvd2) &&
				requireEqual(t, share.RelayParent, relayParent)) {
				overseer.Stop()
			}
		}

		// handle collatorprotocolmessages.Seconded message
		select {
		case <-time.After(2 * time.Minute):
			t.Error("timed out waiting for collatorprotocolmessages.Seconded message")
			overseer.Stop()
			return
		case msg = <-subToOverseer:
			// informed collator protocol that we have seconded the candidate
			_, ok := msg.(collatorprotocolmessages.Seconded)
			if !ok {
				overseer.Stop()
				t.Error("Should be true")
			}
		}
	}(overseer.SubsystemsToOverseer, t)

	// receive second message from overseer to candidate backing subsystem
	overseer.ReceiveMessage(
		backing.SecondMessage{
			RelayParent:             relayParent,
			CandidateReceipt:        candidate2.ToPlain(),
			PersistedValidationData: pvd2,
			PoV:                     pov2,
		})
	wg.Wait()
}

// customised to return the bool, So that we can stop the overseer in case of value mismatch
func requireEqual(t *testing.T, expected any, actual any) (isEqual bool) {
	t.Helper()

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("\nNot Equal: \nexpected: %+v\nactual: %+v\n", expected, actual)
		return false
	}
	return true
}
