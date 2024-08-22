// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing_test

import (
	"errors"
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

// Ensure overseer stops before test completion
func stopOverseerAndWaitForCompletion(overseer *overseer.MockableOverseer) {
	overseer.Stop()
	time.Sleep(100 * time.Millisecond) // Give some time for any ongoing processes to finish
}

// register the backing subsystem, run backing subsystem, start overseer
func initBackingAndOverseerMock(t *testing.T) (*backing.CandidateBacking, *overseer.MockableOverseer) {
	t.Helper()

	overseerMock := overseer.NewMockableOverseer(t)

	backing := backing.New(overseerMock.SubsystemsToOverseer)
	overseerMock.RegisterSubsystem(backing)
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
	paraID uint32, //nolint:unparam
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

// signingContext returns a signing context for testing purposes.
func signingContext(t *testing.T) parachaintypes.SigningContext {
	t.Helper()

	return parachaintypes.SigningContext{
		SessionIndex: 1,
		ParentHash:   getDummyHash(t, 5),
	}
}

// this is a helper function to create an expected action for the ValidateFromExhaustive message
// that will return a valid result
func validResponseForValidateFromExhaustive(
	headData parachaintypes.HeadData,
	pvd parachaintypes.PersistedValidationData,
) func(msg any) bool {
	return func(msg any) bool {
		msgValidate, ok := msg.(candidatevalidation.ValidateFromExhaustive)
		if !ok {
			return false
		}

		msgValidate.Ch <- parachaintypes.OverseerFuncRes[candidatevalidation.ValidationResult]{
			Data: candidatevalidation.ValidationResult{
				ValidResult: &candidatevalidation.ValidValidationResult{
					CandidateCommitments: parachaintypes.CandidateCommitments{
						HeadData:                  headData,
						UpwardMessages:            []parachaintypes.UpwardMessage{},
						HorizontalMessages:        []parachaintypes.OutboundHrmpMessage{},
						NewValidationCode:         nil,
						ProcessedDownwardMessages: 0,
						HrmpWatermark:             0,
					},
					PersistedValidationData: pvd,
				},
			},
		}
		return true
	}
}

// this is a expected action for the StoreAvailableData message that will return a nil error
func storeAvailableData(msg any) bool {
	store, ok := msg.(availabilitystore.StoreAvailableData)
	if !ok {
		return false
	}

	store.Sender <- nil
	return true
}

// we can second a valid candidate when the previous candidate has been found invalid
func TestSecondsValidCandidate(t *testing.T) {
	candidateBacking, overseer := initBackingAndOverseerMock(t)
	defer stopOverseerAndWaitForCompletion(overseer)

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

	validate := func(msg any) bool {
		validateFromExhaustive, ok := msg.(candidatevalidation.ValidateFromExhaustive)
		if !ok {
			return false
		}

		badReturn := candidatevalidation.BadReturn
		validateFromExhaustive.Ch <- parachaintypes.OverseerFuncRes[candidatevalidation.ValidationResult]{
			Data: candidatevalidation.ValidationResult{
				InvalidResult: &badReturn,
			},
		}
		return true
	}

	reportInvalid := func(msg any) bool {
		// reported to collator protocol about invalid candidate
		_, ok := msg.(collatorprotocolmessages.Invalid)
		return ok
	}

	// set expected actions for overseer messages we send from the subsystem.
	overseer.ExpectActions(validate, reportInvalid)

	// receive second message from overseer to candidate backing subsystem
	overseer.ReceiveMessage(
		backing.SecondMessage{
			RelayParent:             relayParent,
			CandidateReceipt:        candidate1.ToPlain(),
			PersistedValidationData: pvd1,
			PoV:                     pov1,
		})

	time.Sleep(1 * time.Second)

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

	validate2 := validResponseForValidateFromExhaustive(candidate2.Commitments.HeadData, pvd2)

	distribute := func(msg any) bool {
		// we have seconded a candidate and shared the statement to peers
		share, ok := msg.(parachaintypes.StatementDistributionMessageShare)
		if !ok {
			return false
		}

		statement, err := share.SignedFullStatementWithPVD.SignedFullStatement.Payload.Value()
		require.NoError(t, err)

		require.Equal(t, statement, parachaintypes.Seconded(candidate2))
		require.Equal(t, *share.SignedFullStatementWithPVD.PersistedValidationData, pvd2)
		require.Equal(t, share.RelayParent, relayParent)

		return true
	}

	informSeconded := func(msg any) bool {
		// informed collator protocol that we have seconded the candidate
		_, ok := msg.(collatorprotocolmessages.Seconded)
		return ok
	}

	// set expected actions for overseer messages we send from the subsystem.
	overseer.ExpectActions(validate2, storeAvailableData, distribute, informSeconded)

	// receive second message from overseer to candidate backing subsystem
	overseer.ReceiveMessage(
		backing.SecondMessage{
			RelayParent:             relayParent,
			CandidateReceipt:        candidate2.ToPlain(),
			PersistedValidationData: pvd2,
			PoV:                     pov2,
		})

	time.Sleep(1 * time.Second)
}

// candidate reaches quorum.
// in legacy backing, we need 2 approvals to reach quorum.
func TestCandidateReachesQuorum(t *testing.T) {
	candidateBacking, overseer := initBackingAndOverseerMock(t)
	defer stopOverseerAndWaitForCompletion(overseer)

	paraValidators := parachainValidators(t, candidateBacking.Keystore)
	numOfValidators := uint(len(paraValidators))
	relayParent := getDummyHash(t, 5)
	paraID := uint32(1)

	pov := parachaintypes.PoV{BlockData: []byte{1, 2, 3}}
	povHash, err := pov.Hash()
	require.NoError(t, err)

	pvd := dummyPVD(t)
	validationCode := parachaintypes.ValidationCode{1, 2, 3}

	signingContext := signingContext(t)

	ctrl := gomock.NewController(t)
	mockBlockState := backing.NewMockBlockState(ctrl)
	mockRuntime := backing.NewMockInstance(ctrl)
	mockImplicitView := backing.NewMockImplicitView(ctrl)

	candidateBacking.BlockState = mockBlockState
	candidateBacking.ImplicitView = mockImplicitView

	// mock BlockState methods
	mockBlockState.EXPECT().GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).
		Return(mockRuntime, nil).Times(3)

	// mock Runtime Instance methods
	mockRuntime.EXPECT().ParachainHostAsyncBackingParams().
		Return(nil, wazero_runtime.ErrExportFunctionNotFound)
	mockRuntime.EXPECT().ParachainHostSessionIndexForChild().
		Return(parachaintypes.SessionIndex(1), nil).Times(2)
	mockRuntime.EXPECT().ParachainHostValidators().
		Return(paraValidators, nil)
	mockRuntime.EXPECT().ParachainHostValidatorGroups().
		Return(validatorGroups(t), nil)
	mockRuntime.EXPECT().ParachainHostAvailabilityCores().
		Return(availabilityCores(t), nil)
	mockRuntime.EXPECT().ParachainHostMinimumBackingVotes().
		Return(backing.LEGACY_MIN_BACKING_VOTES, nil)
	mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.AssignableToTypeOf(common.Hash{})).
		Return(&validationCode, nil)
	mockRuntime.EXPECT().
		ParachainHostSessionExecutorParams(gomock.AssignableToTypeOf(parachaintypes.SessionIndex(0))).
		Return(nil, wazero_runtime.ErrExportFunctionNotFound).Times(1)

	//mock ImplicitView
	mockImplicitView.EXPECT().AllAllowedRelayParents().
		Return([]common.Hash{})

	// to make entry in perRelayParent map
	overseer.ReceiveMessage(parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{Hash: relayParent, Number: 1},
	})

	time.Sleep(1 * time.Second)

	headData := parachaintypes.HeadData{Data: []byte{4, 5, 6}}

	candidate := newCommittedCandidate(
		t,
		paraID,
		headData,
		povHash,
		relayParent,
		makeErasureRoot(t, numOfValidators, pov, pvd),
		common.Hash{},
		validationCode,
	)

	candidateHash, err := parachaintypes.GetCandidateHash(candidate)
	require.NoError(t, err)

	statementSeconded := parachaintypes.NewStatementVDT()
	err = statementSeconded.SetValue(parachaintypes.Seconded(candidate))
	require.NoError(t, err)

	statementSecondedSign, err := statementSeconded.Sign(candidateBacking.Keystore, signingContext, paraValidators[2])
	require.NoError(t, err)

	signedStatementSeconded := parachaintypes.SignedFullStatementWithPVD{
		SignedFullStatement: parachaintypes.SignedFullStatement{
			Payload:        statementSeconded,
			ValidatorIndex: 2,
			Signature:      *statementSecondedSign,
		},
		PersistedValidationData: &pvd,
	}

	fetchPov := func(msg any) bool {
		fetch, ok := msg.(parachaintypes.AvailabilityDistributionMessageFetchPoV)
		if !ok {
			return false
		}

		fetch.PovCh <- parachaintypes.OverseerFuncRes[parachaintypes.PoV]{Data: pov}
		return true
	}

	validate := validResponseForValidateFromExhaustive(headData, pvd)

	distribute := func(msg any) bool {
		_, ok := msg.(parachaintypes.StatementDistributionMessageShare)
		return ok
	}

	provisionerMessageProvisionableData := func(msg any) bool {
		_, ok := msg.(parachaintypes.ProvisionerMessageProvisionableData)
		return ok
	}

	// set expected actions for overseer messages we send from the subsystem.
	overseer.ExpectActions(fetchPov, validate, storeAvailableData, distribute, provisionerMessageProvisionableData)

	// receive statement message from overseer to candidate backing subsystem containing seconded statement
	overseer.ReceiveMessage(backing.StatementMessage{
		RelayParent:         relayParent,
		SignedFullStatement: signedStatementSeconded,
	})

	time.Sleep(1 * time.Second)

	getBackable := backing.GetBackableCandidatesMessage{
		Candidates: []*backing.CandidateHashAndRelayParent{
			{
				CandidateHash:        candidateHash,
				CandidateRelayParent: relayParent,
			},
		},
		ResCh: make(chan []*parachaintypes.BackedCandidate),
	}

	// receive get backable candidates message from overseer to candidate backing subsystem
	overseer.ReceiveMessage(getBackable)
	backableCandidates := <-getBackable.ResCh

	// we need minimum 2 approvals to consider candidate as backable(in legacy backing).
	// we have received seconded statement, that means we have 1st approval.
	// as it is a seconded statement, we validate the candidate and if we find it valid, that is the 2nd approval.
	require.Len(t, backableCandidates, 1)
	require.Len(t, backableCandidates[0].ValidityVotes, 2)

	time.Sleep(1 * time.Second)

	statementValid := parachaintypes.NewStatementVDT()
	err = statementValid.SetValue(parachaintypes.Valid(candidateHash))
	require.NoError(t, err)

	statementValidSign, err := statementValid.Sign(candidateBacking.Keystore, signingContext, paraValidators[5])
	require.NoError(t, err)

	signedStatementValid := parachaintypes.SignedFullStatementWithPVD{
		SignedFullStatement: parachaintypes.SignedFullStatement{
			Payload:        statementValid,
			ValidatorIndex: 5,
			Signature:      *statementValidSign,
		},
	}

	// receive statement message from overseer to candidate backing subsystem containing valid statement
	overseer.ReceiveMessage(backing.StatementMessage{
		RelayParent:         relayParent,
		SignedFullStatement: signedStatementValid,
	})

	time.Sleep(1 * time.Second)

	getBackable = backing.GetBackableCandidatesMessage{
		Candidates: []*backing.CandidateHashAndRelayParent{
			{
				CandidateHash:        candidateHash,
				CandidateRelayParent: relayParent,
			},
		},
		ResCh: make(chan []*parachaintypes.BackedCandidate),
	}

	overseer.ReceiveMessage(getBackable)
	backableCandidates = <-getBackable.ResCh

	// we already have 2 approvals,
	// and we have received valid statement, that is the 3rd approval.
	// as it is a valid statement, we do not validate the candidate, just store into the statement table.
	require.Len(t, backableCandidates, 1)
	require.Len(t, backableCandidates[0].ValidityVotes, 3)
}

// if the validation of the candidate has failed this does not stop the work of this subsystem
// and so it is not fatal to the node.
func TestValidationFailDoesNotStopSubsystem(t *testing.T) {
	candidateBacking, overseer := initBackingAndOverseerMock(t)
	defer stopOverseerAndWaitForCompletion(overseer)

	paraValidators := parachainValidators(t, candidateBacking.Keystore)
	numOfValidators := uint(len(paraValidators))
	relayParent := getDummyHash(t, 5)
	paraID := uint32(1)

	pov := parachaintypes.PoV{BlockData: []byte{1, 2, 3}}
	povHash, err := pov.Hash()
	require.NoError(t, err)

	pvd := dummyPVD(t)
	validationCode := parachaintypes.ValidationCode{1, 2, 3}

	signingContext := signingContext(t)

	ctrl := gomock.NewController(t)
	mockBlockState := backing.NewMockBlockState(ctrl)
	mockRuntime := backing.NewMockInstance(ctrl)
	mockImplicitView := backing.NewMockImplicitView(ctrl)

	candidateBacking.BlockState = mockBlockState
	candidateBacking.ImplicitView = mockImplicitView

	// mock BlockState methods
	mockBlockState.EXPECT().GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).
		Return(mockRuntime, nil).Times(3)

	// mock Runtime Instance methods
	mockRuntime.EXPECT().ParachainHostAsyncBackingParams().
		Return(nil, wazero_runtime.ErrExportFunctionNotFound)
	mockRuntime.EXPECT().ParachainHostSessionIndexForChild().
		Return(parachaintypes.SessionIndex(1), nil).Times(2)
	mockRuntime.EXPECT().ParachainHostValidators().
		Return(paraValidators, nil)
	mockRuntime.EXPECT().ParachainHostValidatorGroups().
		Return(validatorGroups(t), nil)
	mockRuntime.EXPECT().ParachainHostAvailabilityCores().
		Return(availabilityCores(t), nil)
	mockRuntime.EXPECT().ParachainHostMinimumBackingVotes().
		Return(backing.LEGACY_MIN_BACKING_VOTES, nil)
	mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.AssignableToTypeOf(common.Hash{})).
		Return(&validationCode, nil)
	mockRuntime.EXPECT().
		ParachainHostSessionExecutorParams(gomock.AssignableToTypeOf(parachaintypes.SessionIndex(0))).
		Return(nil, wazero_runtime.ErrExportFunctionNotFound).Times(1)

	//mock ImplicitView
	mockImplicitView.EXPECT().AllAllowedRelayParents().
		Return([]common.Hash{})

	// to make entry in perRelayParent map
	overseer.ReceiveMessage(parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{Hash: relayParent, Number: 1},
	})

	time.Sleep(1 * time.Second)

	headData := parachaintypes.HeadData{Data: []byte{4, 5, 6}}

	candidate := newCommittedCandidate(
		t,
		paraID,
		headData,
		povHash,
		relayParent,
		makeErasureRoot(t, numOfValidators, pov, pvd),
		common.Hash{},
		validationCode,
	)

	statementSeconded := parachaintypes.NewStatementVDT()
	err = statementSeconded.SetValue(parachaintypes.Seconded(candidate))
	require.NoError(t, err)

	statementSecondedSign, err := statementSeconded.Sign(candidateBacking.Keystore, signingContext, paraValidators[2])
	require.NoError(t, err)

	signedStatementSeconded := parachaintypes.SignedFullStatementWithPVD{
		SignedFullStatement: parachaintypes.SignedFullStatement{
			Payload:        statementSeconded,
			ValidatorIndex: 2,
			Signature:      *statementSecondedSign,
		},
		PersistedValidationData: &pvd,
	}

	fetchPov := func(msg any) bool {
		fetch, ok := msg.(parachaintypes.AvailabilityDistributionMessageFetchPoV)
		if !ok {
			return false
		}

		fetch.PovCh <- parachaintypes.OverseerFuncRes[parachaintypes.PoV]{Data: pov}
		return true
	}

	validate := func(msg any) bool {
		msgValidate, ok := msg.(candidatevalidation.ValidateFromExhaustive)
		if !ok {
			return false
		}

		msgValidate.Ch <- parachaintypes.OverseerFuncRes[candidatevalidation.ValidationResult]{
			Err: errors.New("some internal error"),
		}
		return true
	}

	overseer.ExpectActions(fetchPov, validate)

	// receive statement message from overseer to candidate backing subsystem containing seconded statement
	overseer.ReceiveMessage(backing.StatementMessage{
		RelayParent:         relayParent,
		SignedFullStatement: signedStatementSeconded,
	})

	time.Sleep(1 * time.Second)

	candidateHash, err := parachaintypes.GetCandidateHash(candidate)
	require.NoError(t, err)

	getBackable := backing.GetBackableCandidatesMessage{
		Candidates: []*backing.CandidateHashAndRelayParent{
			{
				CandidateHash:        candidateHash,
				CandidateRelayParent: relayParent,
			},
		},
		ResCh: make(chan []*parachaintypes.BackedCandidate),
	}

	// to make sure the candidate backing subsystem has not stopped working,
	// we receive get backable candidates message from overseer to candidate backing subsystem
	overseer.ReceiveMessage(getBackable)
	backableCandidates := <-getBackable.ResCh

	require.Len(t, backableCandidates, 0)
}

// It's impossible to second multiple candidates per relay parent without prospective parachains.
func TestCanNotSecondMultipleCandidatesPerRelayParent(t *testing.T) {
	candidateBacking, overseer := initBackingAndOverseerMock(t)
	defer stopOverseerAndWaitForCompletion(overseer)

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

	// to make entry in perRelayParent map
	overseer.ReceiveMessage(parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{Hash: relayParent, Number: 1},
	})

	time.Sleep(1 * time.Second)

	headData := parachaintypes.HeadData{Data: []byte{4, 5, 6}}

	pov := parachaintypes.PoV{BlockData: []byte{1, 2, 3}}
	povHash, err := pov.Hash()
	require.NoError(t, err)

	pvd := dummyPVD(t)
	pvdHash, err := pvd.Hash()
	require.NoError(t, err)

	validationCode1 := parachaintypes.ValidationCode{1, 2, 3}

	candidate1 := newCommittedCandidate(t,
		paraID,
		headData,
		povHash,
		relayParent,
		makeErasureRoot(t, numOfValidators, pov, pvd),
		pvdHash,
		validationCode1,
	)

	validate := validResponseForValidateFromExhaustive(headData, pvd)

	distribute := func(msg any) bool {
		// we have seconded a candidate and shared the statement to peers
		share, ok := msg.(parachaintypes.StatementDistributionMessageShare)
		if !ok {
			return false
		}

		statement, err := share.SignedFullStatementWithPVD.SignedFullStatement.Payload.Value()
		require.NoError(t, err)

		require.Equal(t, statement, parachaintypes.Seconded(candidate1))
		require.Equal(t, *share.SignedFullStatementWithPVD.PersistedValidationData, pvd)
		require.Equal(t, share.RelayParent, relayParent)

		return true
	}

	informSeconded := func(msg any) bool {
		// informed collator protocol that we have seconded the candidate
		_, ok := msg.(collatorprotocolmessages.Seconded)
		return ok
	}

	overseer.ExpectActions(validate, storeAvailableData, distribute, informSeconded)

	// mocked for candidate1
	mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.AssignableToTypeOf(common.Hash{})).
		Return(&validationCode1, nil)

	overseer.ReceiveMessage(backing.SecondMessage{
		RelayParent:             relayParent,
		CandidateReceipt:        candidate1.ToPlain(),
		PersistedValidationData: pvd,
		PoV:                     pov,
	})

	time.Sleep(1 * time.Second)

	validationCode2 := parachaintypes.ValidationCode{4, 5, 6}

	candidate2 := newCommittedCandidate(t,
		paraID,
		headData,
		povHash,
		relayParent,
		makeErasureRoot(t, numOfValidators, pov, pvd),
		pvdHash,
		validationCode2,
	)

	// Validate the candidate, but the candidate is rejected because the leaf is already occupied.
	// should not expect `StatementDistributionMessageShare` and `collator protocol messages.Seconded` overseer messages.
	overseer.ExpectActions(validate, storeAvailableData)

	// mocked for candidate2
	mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.AssignableToTypeOf(common.Hash{})).
		Return(&validationCode2, nil)

	// Try to second candidate with the same relay parent again.
	overseer.ReceiveMessage(backing.SecondMessage{
		RelayParent:             relayParent,
		CandidateReceipt:        candidate2.ToPlain(),
		PersistedValidationData: pvd,
		PoV:                     pov,
	})

	time.Sleep(1 * time.Second)
}
