package dispute

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	disputetypes "github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/dgraph-io/badger/v4"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func getByzantineThreshold(n int) int {
	if n < 1 {
		return 0
	}
	return (n - 1) / 3
}

func getSuperMajorityThreshold(n int) int {
	return n - getByzantineThreshold(n)
}

func newDisputesCoordinator(backend *overlayBackend,
	overseer chan any,
	receiver chan any,
	runtime *MockRuntimeInstance,
	keystore keystore.Keystore,
) Coordinator {
	runtimeCache := newRuntimeInfo(runtime)
	return Coordinator{
		store:        backend,
		overseer:     overseer,
		receiver:     receiver,
		runtime:      runtimeCache,
		keystore:     keystore,
		maxSpamVotes: 1,
	}
}

type VoteType int

const (
	BackingVote VoteType = iota
	ExplicitVote
)

func (ts *TestState) generateOpposingVotesPair(t *testing.T,
	validVoterID, invalidVoterID parachaintypes.ValidatorIndex,
	candidateHash common.Hash,
	session parachaintypes.SessionIndex,
	validVoteType VoteType,
) (disputetypes.SignedDisputeStatement, disputetypes.SignedDisputeStatement) {
	var validVote, invalidVote disputetypes.SignedDisputeStatement
	switch validVoteType {
	case BackingVote:
		validVote = ts.issueBackingStatementWithIndex(t, validVoterID, candidateHash, session)
	case ExplicitVote:
		validVote = ts.issueExplicitStatementWithIndex(t, validVoterID, candidateHash, session, true)
	default:
		panic("invalid vote type")
	}

	invalidVote = ts.issueExplicitStatementWithIndex(t, invalidVoterID, candidateHash, session, false)
	return validVote, invalidVote
}

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
	db                *overlayBackend
	mockOverseer      chan any
	subsystemReceiver chan any
	runtime           *MockRuntimeInstance
	subsystem         Coordinator
}

func newTestState(t *testing.T) *TestState {
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	pair1, err := sr25519.NewKeypairFromSeed(
		common.MustHexToBytes("0x19069d692afc5f9d30ed3931a3f17616abc883243990a4b4a68739d6bb6c5963"),
	)
	require.NoError(t, err)
	pair2, err := sr25519.NewKeypairFromSeed(
		common.MustHexToBytes("0x234cf5b4beb779c4fe2bd48a6cca4386f3b556fe83740d9008b17da23c311485"),
	)
	require.NoError(t, err)
	pair3, err := sr25519.NewKeypairFromSeed(
		common.MustHexToBytes("0xc77b5e3744610c1bdeae82aa541f7bdbff4cb52c3c1dc3aadabd658a3d0793d0"),
	)
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

	subsystemKeyStore := keystore.NewBasicKeystore("overseer", crypto.Sr25519Type)
	err = subsystemKeyStore.Insert(kr.KeyAlice)
	require.NoError(t, err)

	db, err := badger.Open(badger.DefaultOptions(t.TempDir()))
	require.NoError(t, err)

	dbBackend := NewDBBackend(db)
	backend := newOverlayBackend(dbBackend)

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

	mockOverseer := make(chan any, 10)
	subsystemReceiver := make(chan any)

	controller := gomock.NewController(t)
	runtime := NewMockRuntimeInstance(controller)
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
		db:                backend,
		mockOverseer:      mockOverseer,
		subsystemReceiver: subsystemReceiver,
		runtime:           runtime,
		subsystem:         newDisputesCoordinator(backend, mockOverseer, subsystemReceiver, runtime, subsystemKeyStore),
	}
}

func (ts *TestState) run(t *testing.T) {
	t.Helper()
	digestItem := scale.MustNewVaryingDataType(types.PreRuntimeDigest{}, types.ConsensusDigest{}, types.SealDigest{})
	digest := scale.NewVaryingDataTypeSlice(digestItem)
	h1 := types.Header{
		ParentHash:     ts.lastBlock,
		Number:         1,
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Digest:         digest,
	}
	h1Hash := h1.Hash()
	ts.headers[h1Hash] = h1
	ts.blockNumToHeader[1] = h1Hash
	ts.lastBlock = h1Hash

	h2 := types.Header{
		ParentHash:     ts.lastBlock,
		Number:         2,
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Digest:         digest,
	}
	h2Hash := h2.Hash()
	ts.headers[h2Hash] = h2
	ts.blockNumToHeader[2] = h2Hash
	ts.lastBlock = h2Hash

	err := ts.subsystem.Run()
	require.NoError(t, err)
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

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ts.mockSyncQueries(t, blockHash, session)
	}()

	activatedLeaf := overseer.ActivatedLeaf{
		Hash:   blockHash,
		Number: uint32(blockNumber),
		Status: overseer.LeafStatusFresh,
	}
	err := sendMessage(ts.subsystemReceiver, overseer.Signal[overseer.ActiveLeavesUpdate]{
		Data: overseer.ActiveLeavesUpdate{Activated: &activatedLeaf},
	})
	require.NoError(t, err)
	wg.Wait()
}

func (ts *TestState) mockSyncQueries(t *testing.T,
	blockHash common.Hash,
	session parachaintypes.SessionIndex,
) []disputetypes.UncheckedDisputeMessage {
	var (
		gotSessionInformation  bool
		gotScrapingInformation bool
		notifySession          = make(chan bool)
	)

	var sentDisputes []disputetypes.UncheckedDisputeMessage
	ts.runtime.EXPECT().ParachainHostSessionIndexForChild(blockHash).DoAndReturn(func(arg0 common.Hash) (parachaintypes.SessionIndex, error) {
		require.False(t, gotSessionInformation, "session info already retrieved")
		gotSessionInformation = true
		firstExpectedSession := saturatingSub(uint32(session), Window-1)

		counter := uint32(0)
		if ts.knownSession == nil {
			counter = uint32(session) - firstExpectedSession + 1
		}
		if counter > 0 {
			ts.runtime.EXPECT().ParachainHostSessionInfo(blockHash, gomock.Any()).
				DoAndReturn(func(arg0 common.Hash, arg1 parachaintypes.SessionIndex) (*parachaintypes.SessionInfo, error) {
					return ts.sessionInfo(), nil
				}).Times(int(counter))
		}

		ts.knownSession = &session
		notifySession <- true
		return session, nil
	}).Times(1)

	for {
		select {
		case msg := <-ts.mockOverseer:
			switch message := msg.(type) {
			case overseer.ChainAPIMessage[overseer.FinalizedBlockNumber]:
				require.False(t, gotScrapingInformation, "scraping info already retrieved")
				gotScrapingInformation = true
				message.ResponseChannel <- overseer.BlockNumberResponse{
					Number: uint32(ts.headers[blockHash].Number),
					Err:    nil,
				}
			case overseer.ChainAPIMessage[overseer.BlockNumber]:
				header, ok := ts.headers[blockHash]
				require.True(t, ok)
				message.ResponseChannel <- uint32(header.Number)
			case overseer.ChainAPIMessage[overseer.Ancestors]:
				targetHeader, ok := ts.headers[blockHash]
				require.True(t, ok)
				var ancestors []common.Hash
				for i := uint32(0); i < saturatingSub(uint32(targetHeader.Number), message.Message.K); i++ {
					ancestor, ok := ts.blockNumToHeader[i]
					require.True(t, ok)
					ancestors = append(ancestors, ancestor)
				}
				message.ResponseChannel <- overseer.AncestorsResponse{
					Ancestors: ancestors,
					Error:     nil,
				}
			case disputetypes.DisputeMessageVDT:
				value, err := message.Value()
				require.NoError(t, err)
				disputeMessage, ok := value.(disputetypes.UncheckedDisputeMessage)
				require.True(t, ok)
				sentDisputes = append(sentDisputes, disputeMessage)
			default:
				err := fmt.Errorf("unexpected message type: %T", msg)
				require.NoError(t, err)
			}
		case <-notifySession:
			t.Logf("gotSessionInformation: %t, gotScrapingInformation: %t", gotSessionInformation, gotScrapingInformation)
			if gotScrapingInformation {
				ts.runtime.ctrl.Finish()
				break
			}
		}

		if gotSessionInformation && gotScrapingInformation {
			ts.runtime.ctrl.Finish()
			break
		}
	}

	return sentDisputes
}

func (ts *TestState) mockRuntimeCalls(t *testing.T,
	session parachaintypes.SessionIndex,
	initialEvents *scale.VaryingDataTypeSlice,
	activatedSessionEvents *scale.VaryingDataTypeSlice,
	resumeEvents *scale.VaryingDataTypeSlice,
	initialised *bool,
	restarted *bool,
) {
	ts.runtime.EXPECT().ParachainHostCandidateEvents(gomock.Any()).DoAndReturn(func(arg0 common.Hash) (*scale.VaryingDataTypeSlice, error) {
		leaves := make([]common.Hash, 0, len(ts.headers))
		for leaf := range ts.headers {
			leaves = append(leaves, leaf)
		}

		var (
			found bool
			index int
		)
		for i, leaf := range leaves {
			if bytes.Equal(leaf[:], arg0[:]) {
				found = true
				index = i
				break
			}
		}
		require.True(t, found)

		require.NotNil(t, initialised)
		require.NotNil(t, restarted)
		if *initialised {
			if !*restarted {
				return activatedSessionEvents, nil
			} else {
				return resumeEvents, nil
			}
		} else {
			if index != 1 {
				return nil, nil
			}
			return initialEvents, nil
		}
	}).AnyTimes()
	ts.runtime.EXPECT().ParachainHostOnChainVotes(gomock.Any()).DoAndReturn(func(arg0 common.Hash) (*parachaintypes.ScrapedOnChainVotes, error) {
		return &parachaintypes.ScrapedOnChainVotes{
			Session: session,
		}, nil
	}).AnyTimes()
}

func (ts *TestState) mockResumeSync(t *testing.T,
	session *parachaintypes.SessionIndex,
) []disputetypes.UncheckedDisputeMessage {
	leaves := make([]common.Hash, 0, len(ts.headers))
	for leaf := range ts.headers {
		leaves = append(leaves, leaf)
	}

	var messages []disputetypes.UncheckedDisputeMessage
	var lock sync.Mutex
	var wg sync.WaitGroup
	for n, leaf := range leaves {
		wg.Add(1)
		go func(n int, leaf common.Hash) {
			defer wg.Done()
			t.Logf("mocking sync for leaf %d", n)
			newMessages := ts.mockSyncQueries(t, leaf, *session)
			lock.Lock()
			messages = append(messages, newMessages...)
			lock.Unlock()
		}(n, leaf)

		time.Sleep(1 * time.Second)
		activatedLeaf := overseer.ActivatedLeaf{
			Hash:   leaf,
			Number: uint32(n),
			Status: overseer.LeafStatusFresh,
		}
		err := sendMessage(ts.subsystemReceiver, overseer.Signal[overseer.ActiveLeavesUpdate]{
			Data: overseer.ActiveLeavesUpdate{Activated: &activatedLeaf},
		})
		require.NoError(t, err)
		wg.Wait()
		time.Sleep(2 * time.Second)
	}

	t.Logf("returning from mockResumeSyncWithEvents")
	return messages
}

func (ts *TestState) issueExplicitStatementWithIndex(t *testing.T,
	index parachaintypes.ValidatorIndex,
	candidateHash common.Hash,
	session parachaintypes.SessionIndex,
	valid bool,
) disputetypes.SignedDisputeStatement {
	t.Helper()
	keypair, err := disputetypes.GetValidatorKeyPair(ts.masterKeystore, ts.validatorPublic, index)
	require.NoError(t, err)

	signedDisputeStatement, err := disputetypes.NewSignedDisputeStatement(keypair, valid, candidateHash, session)
	require.NoError(t, err)

	return signedDisputeStatement
}

func (ts *TestState) issueBackingStatementWithIndex(t *testing.T,
	index parachaintypes.ValidatorIndex,
	candidateHash common.Hash,
	session parachaintypes.SessionIndex,
) disputetypes.SignedDisputeStatement {
	t.Helper()
	keypair, err := disputetypes.GetValidatorKeyPair(ts.masterKeystore, ts.validatorPublic, index)
	require.NoError(t, err)

	signingContext := disputetypes.SigningContext{
		SessionIndex:  session,
		CandidateHash: candidateHash,
	}

	statementVDT := disputetypes.NewCompactStatement()
	err = statementVDT.Set(disputetypes.ValidCompactStatement{
		CandidateHash: candidateHash,
	})
	require.NoError(t, err)

	disputeStatement, err := disputetypes.NewSignedDisputeStatementFromBackingStatement(statementVDT,
		signingContext,
		keypair,
	)
	require.NoError(t, err)

	return disputeStatement
}

func (ts *TestState) issueApprovalVoteWithIndex(t *testing.T,
	index parachaintypes.ValidatorIndex,
	candidateHash common.Hash,
	session parachaintypes.SessionIndex,
) disputetypes.SignedDisputeStatement {
	t.Helper()
	keypair, err := disputetypes.GetValidatorKeyPair(ts.masterKeystore, ts.validatorPublic, index)
	require.NoError(t, err)

	vote := disputetypes.ApprovalVote{
		CandidateHash: candidateHash,
	}
	payload, err := vote.SigningPayload(session)
	require.NoError(t, err)
	signature, err := keypair.Sign(payload)
	require.NoError(t, err)

	disputeStatement := inherents.NewDisputeStatement()
	validDisputeStatementKind := inherents.NewValidDisputeStatementKind()
	err = validDisputeStatementKind.Set(inherents.ApprovalChecking{})
	require.NoError(t, err)

	return disputetypes.SignedDisputeStatement{
		DisputeStatement:   disputeStatement,
		CandidateHash:      candidateHash,
		ValidatorPublic:    parachaintypes.ValidatorID(keypair.Public().Encode()),
		ValidatorSignature: parachaintypes.ValidatorSignature(signature),
		SessionIndex:       session,
	}
}

func (ts *TestState) handleGetBlockNumber(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case msg := <-ts.mockOverseer:
		switch message := msg.(type) {
		case overseer.ChainAPIMessage[overseer.BlockNumber]:
			message.ResponseChannel <- uint32(ts.headers[message.Message.Hash].Number)
		default:
			t.Fatalf("unexpected message type: %T", msg)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for block number request")
	}
}

func (ts *TestState) resume(t *testing.T) {
	t.Helper()
	ts.knownSession = nil
	dbBackend := newOverlayBackend(ts.db.inner)
	ts.subsystem = newDisputesCoordinator(dbBackend, ts.mockOverseer, ts.subsystemReceiver, ts.runtime, ts.subsystemKeystore)
	err := ts.subsystem.Run()
	require.NoError(t, err)
}

func (ts *TestState) conclude(t *testing.T) {
	t.Helper()
	concludeSignal := overseer.Signal[overseer.Conclude]{
		Data:            overseer.Conclude{},
		ResponseChannel: nil,
	}
	err := sendMessage(ts.subsystemReceiver, concludeSignal)
	require.NoError(t, err)
}

func (ts *TestState) awaitConclude(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case msg := <-ts.mockOverseer:
		err := fmt.Errorf("unexpected message: %T", msg)
		require.NoError(t, err)
	case <-ctx.Done():
		return
	}
}

func getValidCandidateReceipt(t *testing.T) parachaintypes.CandidateReceipt {
	t.Helper()
	candidateReceipt, err := dummyCandidateReceiptBadSignature(common.Hash{}, &common.Hash{})
	require.NoError(t, err)
	candidateReceipt.CommitmentsHash, err = parachaintypes.CandidateCommitments{}.Hash()
	require.NoError(t, err)
	return candidateReceipt
}

func getInvalidCandidateReceipt(t *testing.T) parachaintypes.CandidateReceipt {
	t.Helper()
	candidateReceipt, err := dummyCandidateReceiptBadSignature(common.Hash{}, &common.Hash{})
	require.NoError(t, err)
	return candidateReceipt
}

func getCandidateBackedEvent(
	t *testing.T,
	candidateReceipt parachaintypes.CandidateReceipt,
) parachaintypes.CandidateEventVDT {
	t.Helper()
	candidateEvent, err := parachaintypes.NewCandidateEventVDT()
	require.NoError(t, err)
	err = candidateEvent.Set(parachaintypes.CandidateBacked{
		CandidateReceipt: candidateReceipt,
		HeadData:         parachaintypes.HeadData{},
		CoreIndex:        parachaintypes.CoreIndex{},
		GroupIndex:       0,
	})
	require.NoError(t, err)
	return candidateEvent
}

func getCandidateIncludedEvent(t *testing.T,
	candidateReceipt parachaintypes.CandidateReceipt,
) parachaintypes.CandidateEventVDT {
	t.Helper()
	candidateEvent, err := parachaintypes.NewCandidateEventVDT()
	require.NoError(t, err)

	err = candidateEvent.Set(parachaintypes.CandidateIncluded{
		CandidateReceipt: candidateReceipt,
		HeadData:         parachaintypes.HeadData{},
		CoreIndex:        parachaintypes.CoreIndex{},
		GroupIndex:       0,
	})
	require.NoError(t, err)

	return candidateEvent
}

func handleApprovalVoteRequest(t *testing.T, overseerChan chan any, expectedHash common.Hash, signature []overseer.ApprovalSignature) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case msg := <-overseerChan:
		switch message := msg.(type) {
		case overseer.ApprovalVotingMessage[overseer.ApprovalSignatureForCandidate]:
			require.Equal(t, expectedHash, message.Message.CandidateHash)
			message.ResponseChan <- overseer.ApprovalSignatureResponse{
				Signature: signature,
				Error:     nil,
			}
		default:
			err := fmt.Errorf("unexpected message type: %T", msg)
			require.NoError(t, err)
		}
	case <-ctx.Done():
		err := fmt.Errorf("timed out waiting for ApprovalSignatureForCandidate request")
		require.NoError(t, err)
	}
}

func recoverAvailableData(t *testing.T, mockOverseer chan any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case msg := <-mockOverseer:
		switch message := msg.(type) {
		case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
			availableData := overseer.AvailableData{
				POV:            []byte{},
				ValidationData: overseer.PersistedValidationData{},
			}
			message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
				AvailableData: &availableData,
				Error:         nil,
			}
			return
		default:
			err := fmt.Errorf("unexpected message type: %T", msg)
			require.NoError(t, err)
		}
	case <-ctx.Done():
		err := fmt.Errorf("timed out waiting for RecoverAvailableData request")
		require.NoError(t, err)
	}
}

func handleParticipationFullHappyPath(t *testing.T,
	mockOverseer chan any,
	runtime *MockRuntimeInstance,
	expectedCommitments common.Hash,
) {
	t.Helper()
	runtime.EXPECT().ParachainHostValidationCodeByHash(gomock.Any(), gomock.Any()).DoAndReturn(func(arg0 common.Hash, arg1 parachaintypes.ValidationCodeHash) (*parachaintypes.ValidationCode, error) {
		return &parachaintypes.ValidationCode{}, nil
	}).Times(1)
	recoverAvailableData(t, mockOverseer)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case msg := <-mockOverseer:
		switch message := msg.(type) {
		case overseer.CandidateValidationMessage[overseer.ValidateFromExhaustive]:
			if message.Data.PvfExecTimeoutKind == overseer.PvfExecTimeoutKindApproval {
				message.ResponseChannel <- overseer.ValidationResult{
					IsValid: expectedCommitments == message.Data.CandidateReceipt.CommitmentsHash,
					Error:   nil,
				}
			}
			return
		default:
			err := fmt.Errorf("unexpected message type: %T", msg)
			require.NoError(t, err)
		}
	case <-ctx.Done():
		err := fmt.Errorf("timed out waiting for ValidateFromExhaustive request")
		require.NoError(t, err)
	}
}

func handleParticipationWithDistribution(t *testing.T,
	mockOverseer chan any,
	runtime *MockRuntimeInstance,
	candidateHash common.Hash,
	expectedCommitmentsHash common.Hash,
) {
	t.Helper()
	handleParticipationFullHappyPath(t, mockOverseer, runtime, expectedCommitmentsHash)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case msg := <-mockOverseer:
		switch message := msg.(type) {
		case disputetypes.DisputeMessageVDT:
			value, err := message.Value()
			require.NoError(t, err)
			dispute, ok := value.(disputetypes.UncheckedDisputeMessage)
			require.True(t, ok)

			receivedHash, err := dispute.CandidateReceipt.Hash()
			require.NoError(t, err)
			require.Equal(t, candidateHash, receivedHash)
			return
		default:
			err := fmt.Errorf("unexpected message type: %T", msg)
			require.NoError(t, err)
		}
	case <-ctx.Done():
		err := fmt.Errorf("timed out waiting for DisputeMessageVDT request")
		require.NoError(t, err)
	}
}

func participationMissingAvailability(t *testing.T, mockOverseer chan any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case msg := <-mockOverseer:
		switch message := msg.(type) {
		case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
			err := overseer.RecoveryErrorUnavailable
			message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
				AvailableData: nil,
				Error:         &err,
			}
			return
		default:
			err := fmt.Errorf("unexpected message type: %T", msg)
			require.NoError(t, err)
		}
	case <-ctx.Done():
		err := fmt.Errorf("timed out waiting for RecoverAvailableData request")
		require.NoError(t, err)
	}
}

func newImportStatementsMessage(t *testing.T,
	candidateReceipt parachaintypes.CandidateReceipt,
	session parachaintypes.SessionIndex,
	statements []disputetypes.Statement,
	respChan chan any,
) disputetypes.Message[disputetypes.ImportStatements] {
	t.Helper()
	return disputetypes.Message[disputetypes.ImportStatements]{
		Data: disputetypes.ImportStatements{
			CandidateReceipt: candidateReceipt,
			Session:          session,
			Statements:       statements,
		},
		ResponseChannel: respChan,
	}
}

func newActiveDisputesQuery(t *testing.T, respChan chan any) disputetypes.Message[disputetypes.ActiveDisputes] {
	t.Helper()
	return disputetypes.Message[disputetypes.ActiveDisputes]{
		Data:            disputetypes.ActiveDisputes{},
		ResponseChannel: respChan,
	}
}

func newCandidateVotesQuery(t *testing.T,
	queries []disputetypes.CandidateVotesQuery,
	respChan chan any,
) disputetypes.Message[disputetypes.QueryCandidateVotes] {
	t.Helper()
	return disputetypes.Message[disputetypes.QueryCandidateVotes]{
		Data: disputetypes.QueryCandidateVotes{
			Queries: queries,
		},
		ResponseChannel: respChan,
	}
}

func newCandidateEvents(t *testing.T, events ...parachaintypes.CandidateEventVDT) scale.VaryingDataTypeSlice {
	t.Helper()
	candidateEvents, err := parachaintypes.NewCandidateEvents()
	require.NoError(t, err)
	for _, event := range events {
		value, err := event.Value()
		require.NoError(t, err)
		err = candidateEvents.Add(value)
		require.NoError(t, err)
	}
	return candidateEvents
}

// sendImportStatementsMessage sends an ImportStatements message to the subsystem. If responseChan is nil, the message
// is sent and the function returns. Otherwise, the message is sent and the response is returned.
func (ts *TestState) sendImportStatementsMessage(t *testing.T,
	candidateReceipt parachaintypes.CandidateReceipt,
	session parachaintypes.SessionIndex,
	statements []disputetypes.Statement,
	responseChan chan any,
) ImportStatementResult {
	t.Helper()
	importMessage := newImportStatementsMessage(t,
		candidateReceipt,
		session,
		statements,
		responseChan,
	)
	if responseChan == nil {
		err := sendMessage(ts.subsystemReceiver, importMessage)
		require.NoError(t, err)
		return ValidImport
	} else {
		res, err := call(ts.subsystemReceiver, importMessage, importMessage.ResponseChannel)
		require.NoError(t, err)
		importResult, ok := res.(ImportStatementResult)
		require.True(t, ok)
		return importResult
	}
}

func (ts *TestState) getActiveDisputes(t *testing.T) scale.BTree {
	disputesMessage := newActiveDisputesQuery(t, make(chan any))
	res, err := call(ts.subsystemReceiver, disputesMessage, disputesMessage.ResponseChannel)
	require.NoError(t, err)

	activeDisputes, ok := res.(scale.BTree)
	require.True(t, ok)
	return activeDisputes
}

func (ts *TestState) getRecentDisputes(t *testing.T) scale.BTree {
	message := disputetypes.Message[disputetypes.RecentDisputesMessage]{
		Data:            disputetypes.RecentDisputesMessage{},
		ResponseChannel: make(chan any),
	}
	res, err := call(ts.subsystemReceiver, message, message.ResponseChannel)
	require.NoError(t, err)
	recentDisputes, ok := res.(scale.BTree)
	require.True(t, ok)
	return recentDisputes
}

func (ts *TestState) getCandidateVotes(t *testing.T,
	session parachaintypes.SessionIndex,
	candidateHash common.Hash,
) []disputetypes.QueryCandidateVotesResponse {
	votesQuery := newCandidateVotesQuery(t,
		[]disputetypes.CandidateVotesQuery{
			{
				Session:       session,
				CandidateHash: candidateHash,
			},
		},
		make(chan any),
	)
	res, err := call(ts.subsystemReceiver, votesQuery, votesQuery.ResponseChannel)
	require.NoError(t, err)
	candidateVotes, ok := res.([]disputetypes.QueryCandidateVotesResponse)
	require.True(t, ok)
	return candidateVotes
}

func (ts *TestState) determineUndisputedChain(t *testing.T,
	baseBlock overseer.Block,
	blockDescriptions []disputetypes.BlockDescription,
) disputetypes.DetermineUndisputedChainResponse {
	message := disputetypes.Message[disputetypes.DetermineUndisputedChainMessage]{
		Data: disputetypes.DetermineUndisputedChainMessage{
			Base:              baseBlock,
			BlockDescriptions: blockDescriptions,
		},
		ResponseChannel: make(chan any),
	}
	res, err := call(ts.subsystemReceiver, message, message.ResponseChannel)
	require.NoError(t, err)
	response, ok := res.(disputetypes.DetermineUndisputedChainResponse)
	require.True(t, ok)
	return response
}

func TestDisputesCoordinator(t *testing.T) {
	t.Run("too_many_unconfirmed_statements_are_considered_spam", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t, session, nil, nil, nil, &initialised, &restarted)
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		candidateReceipt1 := getValidCandidateReceipt(t)
		candidateHash1, err := candidateReceipt1.Hash()
		require.NoError(t, err)
		candidateReceipt2 := getInvalidCandidateReceipt(t)
		candidateHash2, err := candidateReceipt2.Hash()
		require.NoError(t, err)
		validVote1, invalidVote1 := ts.generateOpposingVotesPair(t,
			3,
			1,
			candidateHash1,
			session,
			BackingVote,
		)
		validVote2, invalidVote2 := ts.generateOpposingVotesPair(t,
			3,
			1,
			candidateHash2,
			session,
			BackingVote,
		)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote1,
				ValidatorIndex:         3,
			},
			{
				SignedDisputeStatement: invalidVote1,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt1, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash1, []overseer.ApprovalSignature{})

		// Participation has to fail here, otherwise the dispute will be confirmed. However,
		// participation won't happen at all because the dispute is neither backed, not
		// confirmed nor the candidate is included. Or in other words - we'll refrain from
		// participation.
		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, session, activeDisputes.Min().(*disputetypes.Dispute).Comparator.SessionIndex)
		require.Equal(t, candidateHash1, activeDisputes.Min().(*disputetypes.Dispute).Comparator.CandidateHash)

		candidateVotes := ts.getCandidateVotes(t, session, candidateHash1)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, candidateVotes[0].Votes.Valid.Value.Len(), 1)
		require.Equal(t, candidateVotes[0].Votes.Invalid.Len(), 1)

		// Now we'll try to import a second statement for the same candidate. This should fail
		// because the candidate is already disputed.
		wg = sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash2, []overseer.ApprovalSignature{})
		}()
		go func() {
			defer wg.Done()
			statements = []disputetypes.Statement{
				{
					SignedDisputeStatement: validVote2,
					ValidatorIndex:         3,
				},
				{
					SignedDisputeStatement: invalidVote2,
					ValidatorIndex:         1,
				},
			}
			importResult := ts.sendImportStatementsMessage(t, candidateReceipt2, session, statements, make(chan any))
			require.Equal(t, InvalidImport, importResult)
		}()
		wg.Wait()

		candidateVotes = ts.getCandidateVotes(t, session, candidateHash2)
		require.Equal(t, 0, len(candidateVotes))
		ts.conclude(t)
	})
	t.Run("approval_vote_import_works", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t, session, nil, nil, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		candidateReceipt1 := getValidCandidateReceipt(t)
		candidateHash1, err := candidateReceipt1.Hash()
		require.NoError(t, err)
		validVote1, invalidVote1 := ts.generateOpposingVotesPair(t,
			3,
			1,
			candidateHash1,
			session,
			BackingVote,
		)
		wg = sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			approvalVote := ts.issueApprovalVoteWithIndex(t, 4, candidateHash1, session)
			handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash1, []overseer.ApprovalSignature{
				{
					ValidatorIndex:     4,
					ValidatorSignature: approvalVote.ValidatorSignature,
				},
			})
		}()
		go func() {
			defer wg.Done()
			statements := []disputetypes.Statement{
				{
					SignedDisputeStatement: validVote1,
					ValidatorIndex:         3,
				},
				{
					SignedDisputeStatement: invalidVote1,
					ValidatorIndex:         1,
				},
			}
			_ = ts.sendImportStatementsMessage(t, candidateReceipt1, session, statements, nil)
		}()
		wg.Wait()

		// Participation won't happen here because the dispute is neither backed, not confirmed
		// nor the candidate is included. Or in other words - we'll refrain from participation.
		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 1, activeDisputes.Len())
		activeDispute := activeDisputes.Min().(*disputetypes.Dispute)
		require.Equal(t, session, activeDispute.Comparator.SessionIndex)
		require.Equal(t, candidateHash1, activeDispute.Comparator.CandidateHash)
		isActive, err := activeDispute.DisputeStatus.IsActive()
		require.True(t, isActive)

		candidateVotes := ts.getCandidateVotes(t, session, candidateHash1)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, candidateVotes[0].Votes.Valid.Value.Len(), 2)
		require.Equal(t, candidateVotes[0].Votes.Invalid.Len(), 1)
		_, ok := candidateVotes[0].Votes.Valid.Value.Get(4)
		require.True(t, ok)
		ts.conclude(t)
	})
	t.Run("dispute_gets_confirmed_via_participation", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		candidateReceipt1 := getValidCandidateReceipt(t)
		candidateHash1, err := candidateReceipt1.Hash()
		require.NoError(t, err)
		candidateReceipt2 := getInvalidCandidateReceipt(t)
		candidateHash2, err := candidateReceipt2.Hash()
		require.NoError(t, err)
		sessionEvents := newCandidateEvents(t,
			getCandidateBackedEvent(t, candidateReceipt1),
			getCandidateBackedEvent(t, candidateReceipt2),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()

		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		validVote1, invalidVote1 := ts.generateOpposingVotesPair(t,
			3,
			1,
			candidateHash1,
			session,
			BackingVote,
		)
		validVote2, invalidVote2 := ts.generateOpposingVotesPair(t,
			3,
			1,
			candidateHash2,
			session,
			BackingVote,
		)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote1,
				ValidatorIndex:         3,
			},
			{
				SignedDisputeStatement: invalidVote1,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt1, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash1, []overseer.ApprovalSignature{})
		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash1, candidateReceipt1.CommitmentsHash)

		// after participation
		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 1, activeDisputes.Len())
		dispute := activeDisputes.Min().(*disputetypes.Dispute)
		require.Equal(t, session, dispute.Comparator.SessionIndex)
		require.Equal(t, candidateHash1, dispute.Comparator.CandidateHash)
		isActive, err := dispute.DisputeStatus.IsActive()
		require.NoError(t, err)
		require.True(t, isActive)

		candidateVotes := ts.getCandidateVotes(t, session, candidateHash1)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, 1, candidateVotes[0].Votes.Invalid.Len())
		require.Equal(t, 2, candidateVotes[0].Votes.Valid.Value.Len())

		statements = []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote2,
				ValidatorIndex:         3,
			},
			{
				SignedDisputeStatement: invalidVote2,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt2, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash2, []overseer.ApprovalSignature{})
		participationMissingAvailability(t, ts.mockOverseer)

		candidateVotes = ts.getCandidateVotes(t, session, candidateHash2)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, 1, candidateVotes[0].Votes.Invalid.Len())
		require.Equal(t, 1, candidateVotes[0].Votes.Valid.Value.Len())

		ts.conclude(t)
	})
	t.Run("dispute_gets_confirmed_at_byzantine_threshold", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t, session, nil, nil, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()

		candidateReceipt1 := getValidCandidateReceipt(t)
		candidateHash1, err := candidateReceipt1.Hash()
		require.NoError(t, err)
		candidateReceipt2 := getInvalidCandidateReceipt(t)
		candidateHash2, err := candidateReceipt2.Hash()
		require.NoError(t, err)
		validVote1, invalidVote1 := ts.generateOpposingVotesPair(t,
			3,
			1,
			candidateHash1,
			session,
			ExplicitVote,
		)
		validVote1a, invalidVote1a := ts.generateOpposingVotesPair(t,
			4,
			5,
			candidateHash1,
			session,
			ExplicitVote,
		)
		validVote2, invalidVote2 := ts.generateOpposingVotesPair(t,
			3,
			1,
			candidateHash2,
			session,
			ExplicitVote,
		)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote1,
				ValidatorIndex:         3,
			},
			{
				SignedDisputeStatement: invalidVote1,
				ValidatorIndex:         1,
			},
			{
				SignedDisputeStatement: validVote1a,
				ValidatorIndex:         4,
			},
			{
				SignedDisputeStatement: invalidVote1a,
				ValidatorIndex:         5,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt1, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash1, []overseer.ApprovalSignature{})

		// Participation won't happen here because the dispute is neither backed, not confirmed
		// nor the candidate is included. Or in other words - we'll refrain from participation.
		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 1, activeDisputes.Len())
		dispute := activeDisputes.Min().(*disputetypes.Dispute)
		require.Equal(t, session, dispute.Comparator.SessionIndex)
		require.Equal(t, candidateHash1, dispute.Comparator.CandidateHash)

		candidateVotes := ts.getCandidateVotes(t, session, candidateHash1)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, 2, candidateVotes[0].Votes.Valid.Value.Len())
		require.Equal(t, 2, candidateVotes[0].Votes.Invalid.Len())

		statements = []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote2,
				ValidatorIndex:         3,
			},
			{
				SignedDisputeStatement: invalidVote2,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt2, session, statements, nil)
		participationMissingAvailability(t, ts.mockOverseer)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash2, []overseer.ApprovalSignature{})

		candidateVotes = ts.getCandidateVotes(t, session, candidateHash2)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, 1, candidateVotes[0].Votes.Invalid.Len())
		require.Equal(t, 1, candidateVotes[0].Votes.Valid.Value.Len())
		ts.conclude(t)
	})
	t.Run("backing_statements_import_works_and_no_spam", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		sessionEvents := newCandidateEvents(t,
			getCandidateBackedEvent(t, candidateReceipt),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		validVote1 := ts.issueBackingStatementWithIndex(t, 3, candidateHash, session)
		validVote2 := ts.issueBackingStatementWithIndex(t, 4, candidateHash, session)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote1,
				ValidatorIndex:         3,
			},
			{
				SignedDisputeStatement: validVote2,
				ValidatorIndex:         4,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)

		// Just backing votes - we should not have any active disputes now.
		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 0, activeDisputes.Len())

		candidateVotes := ts.getCandidateVotes(t, session, candidateHash)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, 0, candidateVotes[0].Votes.Invalid.Len())
		require.Equal(t, 2, candidateVotes[0].Votes.Valid.Value.Len())

		candidateReceipt = getInvalidCandidateReceipt(t)
		candidateHash, err = candidateReceipt.Hash()
		require.NoError(t, err)

		validVote1 = ts.issueBackingStatementWithIndex(t, 3, candidateHash, session)
		validVote2 = ts.issueBackingStatementWithIndex(t, 4, candidateHash, session)

		ts.activateLeafAtSession(t, session, 1)

		importResult := ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, make(chan any))
		require.Equal(t, ValidImport, importResult)
		ts.conclude(t)
	})
	t.Run("conflicting_votes_lead_to_dispute_participation", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		sessionEvents := newCandidateEvents(t,
			getCandidateBackedEvent(t, candidateReceipt),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		validVote, invalidVote1 := ts.generateOpposingVotesPair(t,
			3,
			1,
			candidateHash,
			session,
			BackingVote,
		)
		invalidVote2 := ts.issueExplicitStatementWithIndex(t, 2, candidateHash, session, false)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         3,
			},
			{
				SignedDisputeStatement: invalidVote1,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, candidateReceipt.CommitmentsHash)

		// after participation
		activeDisputes := ts.getActiveDisputes(t)
		dispute := activeDisputes.Min().(*disputetypes.Dispute)
		require.Equal(t, session, dispute.Comparator.SessionIndex)
		require.Equal(t, candidateHash, dispute.Comparator.CandidateHash)
		isActive, err := dispute.DisputeStatus.IsActive()
		require.NoError(t, err)
		require.True(t, isActive)

		candidateVotes := ts.getCandidateVotes(t, session, candidateHash)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, 1, candidateVotes[0].Votes.Invalid.Len())
		require.Equal(t, 2, candidateVotes[0].Votes.Valid.Value.Len())

		statements = []disputetypes.Statement{
			{
				SignedDisputeStatement: invalidVote2,
				ValidatorIndex:         2,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)

		candidateVotes = ts.getCandidateVotes(t, session, candidateHash)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, 2, candidateVotes[0].Votes.Invalid.Len())
		require.Equal(t, 2, candidateVotes[0].Votes.Valid.Value.Len())
		ts.conclude(t)
	})
	t.Run("positive_votes_dont_trigger_participation", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		sessionEvents := newCandidateEvents(t,
			getCandidateBackedEvent(t, candidateReceipt),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		validVote1 := ts.issueExplicitStatementWithIndex(t, 2, candidateHash, session, true)
		validVote2 := ts.issueExplicitStatementWithIndex(t, 1, candidateHash, session, true)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote1,
				ValidatorIndex:         2,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)

		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 0, activeDisputes.Len())

		candidateVotes := ts.getCandidateVotes(t, session, candidateHash)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, 0, candidateVotes[0].Votes.Invalid.Len())
		require.Equal(t, 1, candidateVotes[0].Votes.Valid.Value.Len())

		statements = []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote2,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)

		activeDisputes = ts.getActiveDisputes(t)
		require.Equal(t, 0, activeDisputes.Len())

		candidateVotes = ts.getCandidateVotes(t, session, candidateHash)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, 0, candidateVotes[0].Votes.Invalid.Len())
		require.Equal(t, 2, candidateVotes[0].Votes.Valid.Value.Len())
		ts.conclude(t)
	})
	t.Run("wrong_validator_index_is_ignored", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t, session, nil, nil, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			2,
			1,
			candidateHash,
			session,
			ExplicitVote,
		)

		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         1,
			},
			{
				SignedDisputeStatement: invalidVote,
				ValidatorIndex:         2,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)

		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 0, activeDisputes.Len())
		ts.conclude(t)
	})
	t.Run("finality_votes_ignore_disputed_candidates", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		sessionEvents := newCandidateEvents(t,
			getCandidateBackedEvent(t, candidateReceipt),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			2,
			1,
			candidateHash,
			session,
			ExplicitVote,
		)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         2,
			},
			{
				SignedDisputeStatement: invalidVote,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, candidateReceipt.CommitmentsHash)

		baseBlockHash := bytes.Repeat([]byte{0x0f}, 32)
		blockHashA := bytes.Repeat([]byte{0x0a}, 32)
		blockHashB := bytes.Repeat([]byte{0x0b}, 32)

		baseBlock := overseer.Block{
			Number: 10,
			Hash:   common.Hash(baseBlockHash),
		}
		blockDescriptions := []disputetypes.BlockDescription{
			{
				BlockHash: common.Hash(blockHashA),
				Session:   session,
				Candidates: []parachaintypes.CandidateHash{
					{
						candidateHash,
					},
				},
			},
		}
		response := ts.determineUndisputedChain(t, baseBlock, blockDescriptions)
		require.Equal(t, uint32(10), response.Block.Number)
		require.Equal(t, common.Hash(baseBlockHash), response.Block.Hash)

		baseBlock = overseer.Block{
			Number: 10,
			Hash:   common.Hash(baseBlockHash),
		}
		blockDescriptions = []disputetypes.BlockDescription{
			{
				BlockHash:  common.Hash(blockHashA),
				Session:    session,
				Candidates: []parachaintypes.CandidateHash{},
			},
			{
				BlockHash:  common.Hash(blockHashB),
				Session:    session,
				Candidates: []parachaintypes.CandidateHash{{candidateHash}},
			},
		}
		response = ts.determineUndisputedChain(t, baseBlock, blockDescriptions)
		require.NoError(t, response.Err)
		require.Equal(t, uint32(11), response.Block.Number)
		require.Equal(t, common.Hash(blockHashA), response.Block.Hash)
		ts.conclude(t)
	})

	//// supermajority checks
	t.Run("supermajority_valid_dispute_may_be_finalized", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		sessionEvents := newCandidateEvents(t,
			getCandidateBackedEvent(t, candidateReceipt),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		superMajorityThreshold := getSuperMajorityThreshold(len(ts.validators))
		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			2,
			1,
			candidateHash,
			session,
			ExplicitVote,
		)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         2,
			},
			{
				SignedDisputeStatement: invalidVote,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, candidateReceipt.CommitmentsHash)

		statements = []disputetypes.Statement{}
		for i := 0; i < superMajorityThreshold-1; i++ {
			validatorIndex := parachaintypes.ValidatorIndex(i + 3)
			vote := ts.issueExplicitStatementWithIndex(t, validatorIndex, candidateHash, session, true)
			statements = append(statements, disputetypes.Statement{
				SignedDisputeStatement: vote,
				ValidatorIndex:         validatorIndex,
			})
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})

		blockHash := bytes.Repeat([]byte{0x0f}, 32)
		blockHashA := bytes.Repeat([]byte{0x0a}, 32)
		blockHashB := bytes.Repeat([]byte{0x0b}, 32)

		baseBlock := overseer.Block{
			Number: 10,
			Hash:   common.Hash(blockHash),
		}
		blockDescriptions := []disputetypes.BlockDescription{
			{
				BlockHash:  common.Hash(blockHashA),
				Session:    session,
				Candidates: []parachaintypes.CandidateHash{{candidateHash}},
			},
		}
		response := ts.determineUndisputedChain(t, baseBlock, blockDescriptions)
		require.NoError(t, response.Err)
		require.Equal(t, uint32(11), response.Block.Number)
		require.Equal(t, common.Hash(blockHashA), response.Block.Hash)

		baseBlock = overseer.Block{
			Number: 10,
			Hash:   common.Hash(blockHash),
		}
		blockDescriptions = []disputetypes.BlockDescription{
			{
				BlockHash:  common.Hash(blockHashA),
				Session:    session,
				Candidates: []parachaintypes.CandidateHash{{}},
			},
			{
				BlockHash:  common.Hash(blockHashB),
				Session:    session,
				Candidates: []parachaintypes.CandidateHash{{candidateHash}},
			},
		}
		response = ts.determineUndisputedChain(t, baseBlock, blockDescriptions)
		require.NoError(t, response.Err)
		require.Equal(t, uint32(12), response.Block.Number)
		require.Equal(t, common.Hash(blockHashB), response.Block.Hash)
		ts.conclude(t)
	})
	t.Run("concluded_supermajority_for_non_active_after_time", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		sessionEvents := newCandidateEvents(t,
			getCandidateBackedEvent(t, candidateReceipt),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		superMajorityThreshold := getSuperMajorityThreshold(len(ts.validators))
		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			2,
			1,
			candidateHash,
			session,
			ExplicitVote,
		)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         2,
			},
			{
				SignedDisputeStatement: invalidVote,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, candidateReceipt.CommitmentsHash)

		statements = []disputetypes.Statement{}
		for i := 0; i < superMajorityThreshold-2; i++ {
			validatorIndex := parachaintypes.ValidatorIndex(i + 3)
			vote := ts.issueExplicitStatementWithIndex(t, validatorIndex, candidateHash, session, true)
			statements = append(statements, disputetypes.Statement{
				SignedDisputeStatement: vote,
				ValidatorIndex:         validatorIndex,
			})
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})

		t.Logf("waiting for the dispute to conclude")
		time.Sleep(ActiveDuration + 1*time.Second)

		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 0, activeDisputes.Len())

		recentDisputes := ts.getRecentDisputes(t)
		require.Equal(t, 1, recentDisputes.Len())
		ts.conclude(t)
	})
	t.Run("concluded_supermajority_against_non_active_after_time", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		sessionEvents := newCandidateEvents(t,
			getCandidateBackedEvent(t, candidateReceipt),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		superMajorityThreshold := getSuperMajorityThreshold(len(ts.validators))
		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			2,
			1,
			candidateHash,
			session,
			ExplicitVote,
		)

		wg = sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
			// Use a different expected commitments hash to ensure the candidate validation returns
			// invalid.
			dummyHash := common.Hash{0x01}
			handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, dummyHash)
		}()
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         2,
			},
			{
				SignedDisputeStatement: invalidVote,
				ValidatorIndex:         1,
			},
		}
		importResult := ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, make(chan any))
		require.Equal(t, ValidImport, importResult)
		wg.Wait()

		statements = []disputetypes.Statement{}
		// minus 2, because of local vote and one previously imported invalid vote.
		for i := 0; i < superMajorityThreshold-2; i++ {
			validatorIndex := parachaintypes.ValidatorIndex(i + 3)
			vote := ts.issueExplicitStatementWithIndex(t, validatorIndex, candidateHash, session, false)
			statements = append(statements, disputetypes.Statement{
				SignedDisputeStatement: vote,
				ValidatorIndex:         validatorIndex,
			})
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})

		t.Logf("waiting for the dispute to conclude")
		time.Sleep(ActiveDuration + 1*time.Second)

		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 0, activeDisputes.Len())

		recentDisputes := ts.getRecentDisputes(t)
		require.Equal(t, 1, recentDisputes.Len())
		ts.conclude(t)
	})

	// restart tests
	t.Run("resume_dispute_without_local_statement", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		resumeEvents := newCandidateEvents(t,
			getCandidateBackedEvent(t, candidateReceipt),
		)
		ts.mockRuntimeCalls(t, session, nil, nil, &resumeEvents, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			1,
			2,
			candidateHash,
			session,
			ExplicitVote,
		)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         1,
			},
			{
				SignedDisputeStatement: invalidVote,
				ValidatorIndex:         2,
			},
		}
		wg = sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		}()
		importResult := ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, make(chan any))
		require.Equal(t, ValidImport, importResult)
		wg.Wait()

		// should refrain from participation
		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 1, activeDisputes.Len())
		ts.conclude(t)

		//time.Sleep(5 * time.Second)
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.resume(t)
		}()
		go func() {
			defer wg.Done()
			disputeMessages := ts.mockResumeSync(t, &session)
			require.Nil(t, disputeMessages)
		}()
		wg.Wait()

		wg.Add(2)
		go func() {
			defer wg.Done()
			handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, candidateReceipt.CommitmentsHash)
			handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		}()
		go func() {
			defer wg.Done()
			var statements []disputetypes.Statement
			for i := 3; i <= 7; i++ {
				vote := ts.issueExplicitStatementWithIndex(t, parachaintypes.ValidatorIndex(i), candidateHash, session, true)
				statements = append(statements, disputetypes.Statement{
					SignedDisputeStatement: vote,
					ValidatorIndex:         parachaintypes.ValidatorIndex(i),
				})
			}

			_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, make(chan any))
		}()
		wg.Wait()
		ts.conclude(t)
	})
	t.Run("resume_dispute_with_local_statement", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		sessionEvents, err := parachaintypes.NewCandidateEvents()
		require.NoError(t, err)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		localValidVote := ts.issueExplicitStatementWithIndex(t, 0, candidateHash, session, true)
		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			1,
			2,
			candidateHash,
			session,
			ExplicitVote,
		)
		wg = sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		}()
		go func() {
			defer wg.Done()
			statements := []disputetypes.Statement{
				{
					SignedDisputeStatement: localValidVote,
					ValidatorIndex:         0,
				},
				{
					SignedDisputeStatement: validVote,
					ValidatorIndex:         1,
				},
				{
					SignedDisputeStatement: invalidVote,
					ValidatorIndex:         2,
				},
			}
			importResult := ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, make(chan any))
			require.Equal(t, ValidImport, importResult)
		}()
		wg.Wait()

		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 1, activeDisputes.Len())
		ts.conclude(t)

		time.Sleep(5 * time.Second)
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.resume(t)
		}()
		go func() {
			defer wg.Done()
			sessionEvents = newCandidateEvents(t,
				getCandidateBackedEvent(t, candidateReceipt),
			)
			disputeMessages := ts.mockResumeSync(t, &session)
			require.NotNil(t, disputeMessages)
			require.Equal(t, 1, len(disputeMessages))
		}()
		wg.Wait()
		ts.conclude(t)
	})
	t.Run("resume_dispute_without_local_statement_or_local_key", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)

		subsystemKeyStore := keystore.NewBasicKeystore("overseer", crypto.Sr25519Type)
		pair4, err := sr25519.NewKeypairFromSeed(
			common.MustHexToBytes("0x1581798c69bade2b05f27852d63237eee80e2918a13a3a8b7b08863478b32076"),
		)
		require.NoError(t, err)
		err = subsystemKeyStore.Insert(pair4)
		require.NoError(t, err)
		ts.subsystemKeystore = subsystemKeyStore
		session := parachaintypes.SessionIndex(1)
		wg := sync.WaitGroup{}
		initialised := false
		restarted := false
		sessionEvents := newCandidateEvents(t,
			getCandidateIncludedEvent(t, getValidCandidateReceipt(t)),
		)
		require.NoError(t, err)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.resume(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			1,
			2,
			candidateHash,
			session,
			ExplicitVote,
		)
		wg = sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		}()
		go func() {
			defer wg.Done()
			statements := []disputetypes.Statement{
				{
					SignedDisputeStatement: validVote,
					ValidatorIndex:         1,
				},
				{
					SignedDisputeStatement: invalidVote,
					ValidatorIndex:         2,
				},
			}
			importResult := ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, make(chan any))
			require.Equal(t, ValidImport, importResult)
		}()
		wg.Wait()

		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 1, activeDisputes.Len())
		ts.awaitConclude(t)
		ts.conclude(t)

		time.Sleep(5 * time.Second)
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.resume(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()
		ts.awaitConclude(t)
		ts.conclude(t)
	})

	// Session info tests
	t.Run("session_info_caching_on_startup_works", func(t *testing.T) {
		// SessionInfo cache should be populated
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t, session, nil, nil, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			//time.Sleep(1 * time.Second)
			defer wg.Done()
			ts.run(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()
		ts.conclude(t)
	})
	t.Run("session_info_caching_doesnt_underflow", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(Window + 1)
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t, session, nil, nil, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()
		ts.conclude(t)
	})
	t.Run("session_info_is_requested_only_once", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		sessionEvents := newCandidateEvents(t,
			getCandidateIncludedEvent(t, getValidCandidateReceipt(t)),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()
		initialised = true

		ts.runtime.EXPECT().ParachainHostSessionInfo(gomock.Any(), gomock.Any()).
			DoAndReturn(func(arg0 common.Hash, arg1 parachaintypes.SessionIndex) (*parachaintypes.SessionInfo, error) {
				require.Equal(t, parachaintypes.SessionIndex(2), arg1)
				return ts.sessionInfo(), nil
			}).Times(1)

		ts.activateLeafAtSession(t, session, 3)
		ts.activateLeafAtSession(t, session+1, 4)
		ts.conclude(t)
	})
	t.Run("session_info_big_jump_works", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		sessionEvents := newCandidateEvents(t,
			getCandidateIncludedEvent(t, getValidCandidateReceipt(t)),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 3)

		sessionAfterJump := session + parachaintypes.SessionIndex(Window) + 10
		firstExpectedSession := saturatingSub(uint32(sessionAfterJump), uint32(Window-1))
		go func() {
			times := uint32(sessionAfterJump) - firstExpectedSession + 1
			expectedSession := parachaintypes.SessionIndex(firstExpectedSession)
			ts.runtime.EXPECT().ParachainHostSessionInfo(gomock.Any(), gomock.Any()).
				DoAndReturn(func(arg0 common.Hash, arg1 parachaintypes.SessionIndex) (*parachaintypes.SessionInfo, error) {
					require.Equal(t, expectedSession, arg1)
					// set the expected session for the next call
					if expectedSession < sessionAfterJump {
						expectedSession++
					}
					return ts.sessionInfo(), nil
				}).Times(int(times))
		}()
		ts.activateLeafAtSession(t, sessionAfterJump, 4)
		time.Sleep(2 * time.Second)
		ts.conclude(t)
	})
	t.Run("session_info_small_jump_works", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		sessionEvents := newCandidateEvents(t,
			getCandidateIncludedEvent(t, getValidCandidateReceipt(t)),
		)
		ts.mockRuntimeCalls(t, session, nil, &sessionEvents, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()
		initialised = true

		ts.activateLeafAtSession(t, session, 3)

		sessionAfterJump := session + Window - 1
		firstExpectedSession := session + 1
		go func() {
			times := sessionAfterJump - firstExpectedSession + 1
			expectedSession := firstExpectedSession
			ts.runtime.EXPECT().ParachainHostSessionInfo(gomock.Any(), gomock.Any()).
				DoAndReturn(func(arg0 common.Hash, arg1 parachaintypes.SessionIndex) (*parachaintypes.SessionInfo, error) {
					require.Equal(t, expectedSession, arg1)
					// set the expected session for the next call
					if expectedSession < sessionAfterJump {
						expectedSession++
					}
					return ts.sessionInfo(), nil
				}).Times(int(times))
		}()
		ts.activateLeafAtSession(t, sessionAfterJump, 4)

		time.Sleep(2 * time.Second)
		ts.conclude(t)
	})

	// LocalStatement
	t.Run("issue_valid_local_statement_does_cause_distribution_but_not_duplicate_participation", func(t *testing.T) {
		issueLocalStatementTest(t, true)
	})
	t.Run("issue_invalid_local_statement_does_cause_distribution_but_not_duplicate_participation", func(t *testing.T) {
		issueLocalStatementTest(t, false)
	})

	t.Run("own_approval_vote_gets_distributed_on_dispute", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t, session, nil, nil, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: ts.issueApprovalVoteWithIndex(t, 1, candidateHash, session),
				ValidatorIndex:         0,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)

		wg = sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		}()
		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			2,
			1,
			candidateHash,
			session,
			ExplicitVote,
		)
		statements = []disputetypes.Statement{
			{
				SignedDisputeStatement: invalidVote,
				ValidatorIndex:         1,
			},
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         2,
			},
		}
		importResult := ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, make(chan any))
		require.Equal(t, ValidImport, importResult)
		wg.Wait()
		ts.conclude(t)
	})
	t.Run("negative_issue_local_statement_only_triggers_import", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t, session, nil, nil, nil, &initialised, &restarted)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		message := disputetypes.Message[disputetypes.IssueLocalStatementMessage]{
			Data: disputetypes.IssueLocalStatementMessage{
				Session:          session,
				CandidateHash:    candidateHash,
				CandidateReceipt: candidateReceipt,
				Valid:            false,
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)

		// ensure no participations
		ts.awaitConclude(t)
		ts.conclude(t)

		votes, err := ts.db.GetCandidateVotes(session, candidateHash)
		require.NoError(t, err)
		require.Equal(t, 0, votes.Valid.Value.Len())
		require.Equal(t, 1, votes.Invalid.Len())
	})
	t.Run("redundant_votes_ignored", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t,
			session,
			nil,
			nil,
			nil,
			&initialised,
			&restarted,
		)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		validVote1 := ts.issueBackingStatementWithIndex(t, 1, candidateHash, session)
		validVote2 := ts.issueBackingStatementWithIndex(t, 1, candidateHash, session)
		require.NotEqual(t, validVote1.ValidatorSignature, validVote2.ValidatorSignature)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote1,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)

		statements = []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote2,
				ValidatorIndex:         1,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		ts.conclude(t)

		votes, err := ts.db.GetCandidateVotes(session, candidateHash)
		require.NoError(t, err)
		require.Equal(t, 1, votes.Valid.Value.Len())
		require.Equal(t, 0, votes.Invalid.Len())
		_, vote, ok := votes.Valid.Value.Min()
		require.True(t, ok)
		actualSignature := parachaintypes.ValidatorSignature(vote.ValidatorSignature)
		require.Equal(t, validVote1.ValidatorSignature, actualSignature)
	})
	t.Run("no_onesided_disputes", func(t *testing.T) {
		// Make sure no disputes are recorded when there are no opposing votes, even if we reached supermajority.
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		wg := sync.WaitGroup{}
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t,
			session,
			nil,
			nil,
			nil,
			&initialised,
			&restarted,
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		var statements []disputetypes.Statement
		for i := 0; i < 10; i++ {
			validatorIndex := parachaintypes.ValidatorIndex(i)
			vote := ts.issueBackingStatementWithIndex(t, validatorIndex, candidateHash, session)
			statements = append(statements, disputetypes.Statement{
				SignedDisputeStatement: vote,
				ValidatorIndex:         validatorIndex,
			})
		}
		importResult := ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, make(chan any))
		require.Equal(t, ValidImport, importResult)

		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 0, activeDisputes.Len())
		ts.conclude(t)
	})
	t.Run("refrain_from_participation", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		ts.mockRuntimeCalls(t,
			session,
			nil,
			nil,
			nil,
			&initialised,
			&restarted,
		)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			1,
			2,
			candidateHash,
			session,
			ExplicitVote,
		)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         1,
			},
			{
				SignedDisputeStatement: invalidVote,
				ValidatorIndex:         2,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})

		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 1, activeDisputes.Len())

		candidateVotes := ts.getCandidateVotes(t, session, candidateHash)
		require.Equal(t, 1, candidateVotes[0].Votes.Valid.Value.Len())
		require.Equal(t, 1, candidateVotes[0].Votes.Invalid.Len())

		ts.activateLeafAtSession(t, session, 2)
		ts.awaitConclude(t)
		ts.conclude(t)
	})
	t.Run("participation_for_included_candidates", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		sessionEvents := newCandidateEvents(t,
			getCandidateIncludedEvent(t, candidateReceipt),
		)
		ts.mockRuntimeCalls(t,
			session,
			nil,
			&sessionEvents,
			nil,
			&initialised,
			&restarted,
		)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			1,
			2,
			candidateHash,
			session,
			ExplicitVote,
		)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         1,
			},
			{
				SignedDisputeStatement: invalidVote,
				ValidatorIndex:         2,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, candidateReceipt.CommitmentsHash)

		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 1, activeDisputes.Len())

		candidateVotes := ts.getCandidateVotes(t, session, candidateHash)
		require.Equal(t, 2, candidateVotes[0].Votes.Valid.Value.Len())
		require.Equal(t, 1, candidateVotes[0].Votes.Invalid.Len())

		ts.conclude(t)
	})
	t.Run("local_participation_in_dispute_for_backed_candidate", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		initialised := false
		restarted := false
		sessionEvents, err := parachaintypes.NewCandidateEvents()
		require.NoError(t, err)
		ts.mockRuntimeCalls(t,
			session,
			nil,
			&sessionEvents,
			nil,
			&initialised,
			&restarted,
		)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			ts.mockResumeSync(t, &session)
		}()
		go func() {
			defer wg.Done()
			ts.run(t)
		}()
		wg.Wait()
		initialised = true
		ts.activateLeafAtSession(t, session, 1)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			1,
			2,
			candidateHash,
			session,
			ExplicitVote,
		)
		statements := []disputetypes.Statement{
			{
				SignedDisputeStatement: validVote,
				ValidatorIndex:         1,
			},
			{
				SignedDisputeStatement: invalidVote,
				ValidatorIndex:         2,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		time.Sleep(2 * time.Second)

		sessionEvents = newCandidateEvents(t,
			getCandidateBackedEvent(t, candidateReceipt),
		)
		ts.activateLeafAtSession(t, session, 1)

		statements = []disputetypes.Statement{
			{
				SignedDisputeStatement: ts.issueBackingStatementWithIndex(t, 3, candidateHash, session),
				ValidatorIndex:         3,
			},
		}
		_ = ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, nil)
		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, candidateReceipt.CommitmentsHash)

		activeDisputes := ts.getActiveDisputes(t)
		require.Equal(t, 1, activeDisputes.Len())

		candidateVotes := ts.getCandidateVotes(t, session, candidateHash)
		require.Equal(t, 1, len(candidateVotes))
		require.Equal(t, 3, candidateVotes[0].Votes.Valid.Value.Len())
		require.Equal(t, 1, candidateVotes[0].Votes.Invalid.Len())
	})
	// TODO: tests
	t.Run("participation_requests_reprioritized_for_newly_included", func(t *testing.T) {
		//t.Parallel()
		//ts := newTestState(t)
		//session := parachaintypes.SessionIndex(1)
		//wg := sync.WaitGroup{}
		//wg.Add(2)
		//go func() {
		//	defer wg.Done()
		//	ts.mockResumeSync(t, &session)
		//}()
		//go func() {
		//	defer wg.Done()
		//	ts.run(t)
		//}()
		//wg.Wait()
	})
	t.Run("informs_chain_selection_when_dispute_concluded_against", func(t *testing.T) {

	})
}

func issueLocalStatementTest(t *testing.T, valid bool) {
	t.Parallel()
	ts := newTestState(t)
	session := parachaintypes.SessionIndex(1)
	initialised := false
	restarted := false
	candidateReceipt := getValidCandidateReceipt(t)
	candidateHash, err := candidateReceipt.Hash()
	require.NoError(t, err)
	ts.mockRuntimeCalls(t, session, nil, nil, nil, &initialised, &restarted)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		ts.run(t)
	}()
	go func() {
		defer wg.Done()
		ts.mockResumeSync(t, &session)
	}()
	wg.Wait()
	initialised = true
	ts.activateLeafAtSession(t, session, 1)

	otherVote := ts.issueExplicitStatementWithIndex(t, 1, candidateHash, session, !valid)
	statements := []disputetypes.Statement{
		{
			SignedDisputeStatement: otherVote,
			ValidatorIndex:         1,
		},
	}
	importResult := ts.sendImportStatementsMessage(t, candidateReceipt, session, statements, make(chan any))
	require.Equal(t, ValidImport, importResult)

	// initiate dispute
	localStatement := disputetypes.Message[disputetypes.IssueLocalStatementMessage]{
		Data: disputetypes.IssueLocalStatementMessage{
			Session:          session,
			CandidateHash:    candidateHash,
			CandidateReceipt: candidateReceipt,
			Valid:            valid,
		},
		ResponseChannel: nil,
	}
	err = sendMessage(ts.subsystemReceiver, localStatement)
	require.NoError(t, err)

	// expect it in the distribution
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case msg := <-ts.mockOverseer:
		disputeMessageVDT, ok := msg.(disputetypes.DisputeMessageVDT)
		require.True(t, ok)
		value, err := disputeMessageVDT.Value()
		require.NoError(t, err)
		disputeMessage, ok := value.(disputetypes.UncheckedDisputeMessage)
		require.True(t, ok)
		require.Equal(t, session, disputeMessage.SessionIndex)
		require.Equal(t, candidateReceipt, disputeMessage.CandidateReceipt)
		break
	case <-ctx.Done():
		err := fmt.Errorf("timeout waiting for dispute message")
		require.NoError(t, err)
	}

	handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})

	// ensure no participations
	ts.awaitConclude(t)
	ts.conclude(t)
}
