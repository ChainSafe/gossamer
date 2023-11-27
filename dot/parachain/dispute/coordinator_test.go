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

	mockOverseer := make(chan any, 100)
	subsystemReceiver := make(chan any, 100)

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

func (ts *TestState) run(t *testing.T, session *parachaintypes.SessionIndex) {
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

	done := make(chan bool)
	go func() {
		ts.mockResumeSync(t, session)
		done <- true
	}()

	time.Sleep(20 * time.Second)
	go ts.subsystem.Run()
	<-done
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
	candidateEvents []parachaintypes.CandidateEventVDT,
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

	done := make(chan bool)
	go func() {
		ts.mockSyncQueries(t, blockHash, session, candidateEvents)
		done <- true
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
	<-done
}

func (ts *TestState) mockSyncQueries(t *testing.T,
	blockHash common.Hash,
	session parachaintypes.SessionIndex,
	candidateEvents []parachaintypes.CandidateEventVDT,
) []disputetypes.UncheckedDisputeMessage {
	var (
		gotSessionInformation  bool
		gotScrapingInformation bool
		notifySession          = make(chan bool, 1)
	)

	var sentDisputes []disputetypes.UncheckedDisputeMessage
	ts.runtime.EXPECT().ParachainHostSessionIndexForChild(gomock.Any()).DoAndReturn(func(arg0 common.Hash) (parachaintypes.SessionIndex, error) {
		require.False(t, gotSessionInformation, "session info already retrieved")
		gotSessionInformation = true
		require.Equal(t, blockHash, arg0)
		firstExpectedSession := saturatingSub(uint32(session), Window-1)

		counter := uint32(0)
		if ts.knownSession == nil {
			counter = uint32(session) - firstExpectedSession + 1
		}
		if counter > 0 {
			ts.runtime.EXPECT().ParachainHostSessionInfo(gomock.Any(), gomock.Any()).
				DoAndReturn(func(arg0 common.Hash, arg1 parachaintypes.SessionIndex) (*parachaintypes.SessionInfo, error) {
					require.Equal(t, blockHash, arg0)
					//require.Equal(t, i, arg1)
					return ts.sessionInfo(), nil
				}).Times(int(counter))
		}

		ts.knownSession = &session
		notifySession <- true
		return session, nil
	})

	ts.runtime.EXPECT().ParachainHostCandidateEvents(gomock.Any()).DoAndReturn(func(arg0 common.Hash) (*scale.VaryingDataTypeSlice, error) {
		events, err := parachaintypes.NewCandidateEvents()
		require.NoError(t, err)
		for _, event := range candidateEvents {
			value, err := event.Value()
			require.NoError(t, err)
			err = events.Add(value)
			require.NoError(t, err)
		}
		return &events, nil
	})
	ts.runtime.EXPECT().ParachainHostOnChainVotes(gomock.Any()).DoAndReturn(func(arg0 common.Hash) (*parachaintypes.ScrapedOnChainVotes, error) {
		return &parachaintypes.ScrapedOnChainVotes{
			Session: session,
		}, nil
	})

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
			if gotScrapingInformation {
				break
			}
		}

		if gotSessionInformation && gotScrapingInformation {
			break
		}
	}

	return sentDisputes
}

func (ts *TestState) mockResumeSyncWithEvents(t *testing.T,
	session *parachaintypes.SessionIndex,
	initialEvents []parachaintypes.CandidateEventVDT,
) []disputetypes.UncheckedDisputeMessage {
	leaves := make([]common.Hash, 0, len(ts.headers))
	for leaf := range ts.headers {
		leaves = append(leaves, leaf)
	}

	var messages []disputetypes.UncheckedDisputeMessage
	var mutex sync.Mutex
	var wg sync.WaitGroup
	for _, leaf := range leaves {
		wg.Add(1)
		go func(i int, leaf common.Hash) {
			defer wg.Done()
			var events []parachaintypes.CandidateEventVDT
			if i == 1 {
				events = initialEvents
			}
			newMessages := ts.mockSyncQueries(t, leaf, *session, events)
			mutex.Lock()
			messages = append(messages, newMessages...)
			mutex.Unlock()
		}(len(leaves), leaf)

		activatedLeaf := overseer.ActivatedLeaf{
			Hash:   leaf,
			Number: uint32(len(leaves)),
			Status: overseer.LeafStatusFresh,
		}
		err := sendMessage(ts.subsystemReceiver, overseer.Signal[overseer.ActiveLeavesUpdate]{
			Data: overseer.ActiveLeavesUpdate{Activated: &activatedLeaf},
		})
		require.NoError(t, err)
		wg.Wait()
	}

	return messages
}

func (ts *TestState) mockResumeSync(t *testing.T,
	session *parachaintypes.SessionIndex,
) []disputetypes.UncheckedDisputeMessage {
	return ts.mockResumeSyncWithEvents(t, session, nil)
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
	recoverAvailableData(t, mockOverseer)
	runtime.EXPECT().ParachainHostValidationCodeByHash(gomock.Any(), gomock.Any()).DoAndReturn(func(arg0 common.Hash, arg1 parachaintypes.ValidationCodeHash) (*parachaintypes.ValidationCode, error) {
		return &parachaintypes.ValidationCode{}, nil
	}).Times(1)

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

	//<-wait
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

func TestDisputesCoordinator(t *testing.T) {
	t.Run("too_many_unconfirmed_statements_are_considered_spam", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		ts.run(t, &session)

		candidateReceipt1 := getValidCandidateReceipt(t)
		candidateHash1, err := candidateReceipt1.Hash()
		require.NoError(t, err)
		candidateReceipt2 := getInvalidCandidateReceipt(t)
		candidateHash2, err := candidateReceipt2.Hash()
		require.NoError(t, err)

		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{})

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
		message := disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt1,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote1,
						ValidatorIndex:         3,
					},
					{
						SignedDisputeStatement: invalidVote1,
						ValidatorIndex:         1,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash1, []overseer.ApprovalSignature{})

		// Participation has to fail here, otherwise the dispute will be confirmed. However
		// participation won't happen at all because the dispute is neither backed, not
		// confirmed nor the candidate is included. Or in other words - we'll refrain from
		// participation.
		disputesMessage := disputetypes.Message[disputetypes.ActiveDisputes]{
			Data:            disputetypes.ActiveDisputes{},
			ResponseChannel: make(chan any),
		}
		res, err := call(ts.subsystemReceiver, disputesMessage, disputesMessage.ResponseChannel)
		require.NoError(t, err)

		activeDisputes, ok := res.(scale.BTree)
		require.True(t, ok)
		require.Equal(t, 1, activeDisputes.Len())
		require.Equal(t, session, activeDisputes.Min().(*disputetypes.Dispute).Comparator.SessionIndex)
		require.Equal(t, candidateHash1, activeDisputes.Min().(*disputetypes.Dispute).Comparator.CandidateHash)

		request := disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash1,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, request, request.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok := res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, queryResponse[0].Votes.Valid.Value.Len(), 1)
		require.Equal(t, queryResponse[0].Votes.Invalid.Len(), 1)

		// Now we'll try to import a second statement for the same candidate. This should fail
		// because the candidate is already disputed.
		done := make(chan bool)
		go func() {
			handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash2, []overseer.ApprovalSignature{})
			done <- true
		}()
		message = disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt2,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote2,
						ValidatorIndex:         3,
					},
					{
						SignedDisputeStatement: invalidVote2,
						ValidatorIndex:         1,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, message, message.ResponseChannel)
		require.NoError(t, err)

		<-done

		importResult, ok := res.(ImportStatementResult)
		require.True(t, ok)
		// Result should be invalid, because it should be considered spam.
		require.Equal(t, InvalidImport, importResult)

		query := disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash2,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, query, query.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok = res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 0, len(queryResponse))
	})

	t.Run("approval_vote_import_works", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		ts.run(t, &session)

		candidateReceipt1 := getValidCandidateReceipt(t)
		candidateHash1, err := candidateReceipt1.Hash()
		require.NoError(t, err)

		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{})

		validVote1, invalidVote1 := ts.generateOpposingVotesPair(t,
			3,
			1,
			candidateHash1,
			session,
			BackingVote,
		)

		done := make(chan bool)
		go func() {
			approvalVote := ts.issueApprovalVoteWithIndex(t, 4, candidateHash1, session)
			handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash1, []overseer.ApprovalSignature{
				{
					ValidatorIndex:     4,
					ValidatorSignature: approvalVote.ValidatorSignature,
				},
			})
			done <- true
		}()

		message := disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt1,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote1,
						ValidatorIndex:         3,
					},
					{
						SignedDisputeStatement: invalidVote1,
						ValidatorIndex:         1,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		_, err = call(ts.subsystemReceiver, message, message.ResponseChannel)
		require.NoError(t, err)

		// Participation won't happen here because the dispute is neither backed, not confirmed
		// nor the candidate is included. Or in other words - we'll refrain from participation.
		disputesMessage := disputetypes.Message[disputetypes.ActiveDisputes]{
			Data:            disputetypes.ActiveDisputes{},
			ResponseChannel: make(chan any),
		}
		res, err := call(ts.subsystemReceiver, disputesMessage, disputesMessage.ResponseChannel)
		require.NoError(t, err)

		activeDisputesBtree, ok := res.(scale.BTree)
		require.True(t, ok)
		require.Equal(t, 1, activeDisputesBtree.Len())
		activeDispute := activeDisputesBtree.Min().(*disputetypes.Dispute)
		require.Equal(t, session, activeDispute.Comparator.SessionIndex)
		require.Equal(t, candidateHash1, activeDispute.Comparator.CandidateHash)
		isActive, err := activeDispute.DisputeStatus.IsActive()
		require.True(t, isActive)

		request := disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash1,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, request, request.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok := res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, queryResponse[0].Votes.Valid.Value.Len(), 2)
		require.Equal(t, queryResponse[0].Votes.Invalid.Len(), 1)
		_, ok = queryResponse[0].Votes.Valid.Value.Get(4)
		require.True(t, ok)
	})

	t.Run("dispute_gets_confirmed_via_participation", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		ts.run(t, &session)

		candidateReceipt1 := getValidCandidateReceipt(t)
		candidateHash1, err := candidateReceipt1.Hash()
		require.NoError(t, err)
		candidateReceipt2 := getInvalidCandidateReceipt(t)
		candidateHash2, err := candidateReceipt2.Hash()
		require.NoError(t, err)

		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{
			getCandidateBackedEvent(t, candidateReceipt1),
			getCandidateBackedEvent(t, candidateReceipt2),
		})

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
		message := disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt1,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote1,
						ValidatorIndex:         3,
					},
					{
						SignedDisputeStatement: invalidVote1,
						ValidatorIndex:         1,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash1, []overseer.ApprovalSignature{})

		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash1, candidateReceipt1.CommitmentsHash)

		// after participation
		disputesMessage := disputetypes.Message[disputetypes.ActiveDisputes]{
			Data:            disputetypes.ActiveDisputes{},
			ResponseChannel: make(chan any),
		}
		res, err := call(ts.subsystemReceiver, disputesMessage, disputesMessage.ResponseChannel)
		require.NoError(t, err)
		disputeBtree, ok := res.(scale.BTree)
		require.True(t, ok)
		require.Equal(t, 1, disputeBtree.Len())
		dispute := disputeBtree.Min().(*disputetypes.Dispute)
		require.Equal(t, session, dispute.Comparator.SessionIndex)
		require.Equal(t, candidateHash1, dispute.Comparator.CandidateHash)
		isActive, err := dispute.DisputeStatus.IsActive()
		require.NoError(t, err)
		require.True(t, isActive)

		query := disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash1,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, query, query.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok := res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, 1, queryResponse[0].Votes.Invalid.Len())
		require.Equal(t, 2, queryResponse[0].Votes.Valid.Value.Len())

		message = disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt2,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote2,
						ValidatorIndex:         3,
					},
					{
						SignedDisputeStatement: invalidVote2,
						ValidatorIndex:         1,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash2, []overseer.ApprovalSignature{})
		participationMissingAvailability(t, ts.mockOverseer)

		query = disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash2,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, query, query.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok = res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, 1, queryResponse[0].Votes.Invalid.Len())
		require.Equal(t, 1, queryResponse[0].Votes.Valid.Value.Len())
	})

	t.Run("dispute_gets_confirmed_at_byzantine_threshold", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		ts.run(t, &session)

		candidateReceipt1 := getValidCandidateReceipt(t)
		candidateHash1, err := candidateReceipt1.Hash()
		require.NoError(t, err)
		candidateReceipt2 := getInvalidCandidateReceipt(t)
		candidateHash2, err := candidateReceipt2.Hash()
		require.NoError(t, err)

		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{})

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
		message := disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt1,
				Session:          session,
				Statements: []disputetypes.Statement{
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
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash1, []overseer.ApprovalSignature{})

		// Participation won't happen here because the dispute is neither backed, not confirmed
		// nor the candidate is included. Or in other words - we'll refrain from participation.
		disputesMessage := disputetypes.Message[disputetypes.ActiveDisputes]{
			Data:            disputetypes.ActiveDisputes{},
			ResponseChannel: make(chan any),
		}
		res, err := call(ts.subsystemReceiver, disputesMessage, disputesMessage.ResponseChannel)
		require.NoError(t, err)
		disputeBtree, ok := res.(scale.BTree)
		require.True(t, ok)
		require.Equal(t, 1, disputeBtree.Len())
		dispute := disputeBtree.Min().(*disputetypes.Dispute)
		require.Equal(t, session, dispute.Comparator.SessionIndex)
		require.Equal(t, candidateHash1, dispute.Comparator.CandidateHash)

		query := disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash1,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, query, query.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok := res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, 2, queryResponse[0].Votes.Valid.Value.Len())
		require.Equal(t, 2, queryResponse[0].Votes.Invalid.Len())

		message = disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt2,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote2,
						ValidatorIndex:         3,
					},
					{
						SignedDisputeStatement: invalidVote2,
						ValidatorIndex:         1,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)
		participationMissingAvailability(t, ts.mockOverseer)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash2, []overseer.ApprovalSignature{})

		query = disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash2,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, query, query.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok = res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, 1, queryResponse[0].Votes.Invalid.Len())
		require.Equal(t, 1, queryResponse[0].Votes.Valid.Value.Len())
	})

	t.Run("backing_statements_import_works_and_no_spam", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		ts.run(t, &session)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)

		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{})

		validVote1 := ts.issueBackingStatementWithIndex(t, 3, candidateHash, session)
		validVote2 := ts.issueBackingStatementWithIndex(t, 4, candidateHash, session)

		message := disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote1,
						ValidatorIndex:         3,
					},
					{
						SignedDisputeStatement: validVote2,
						ValidatorIndex:         4,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)

		// Just backing votes - we should not have any active disputes now.
		disputesMessage := disputetypes.Message[disputetypes.ActiveDisputes]{
			Data:            disputetypes.ActiveDisputes{},
			ResponseChannel: make(chan any),
		}
		res, err := call(ts.subsystemReceiver, disputesMessage, disputesMessage.ResponseChannel)
		require.NoError(t, err)
		disputeBtree, ok := res.(scale.BTree)
		require.True(t, ok)
		require.Equal(t, 0, disputeBtree.Len())

		query := disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, query, query.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok := res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, 0, queryResponse[0].Votes.Invalid.Len())
		require.Equal(t, 2, queryResponse[0].Votes.Valid.Value.Len())

		candidateReceipt = getInvalidCandidateReceipt(t)
		candidateHash, err = candidateReceipt.Hash()
		require.NoError(t, err)

		validVote1 = ts.issueBackingStatementWithIndex(t, 3, candidateHash, session)
		validVote2 = ts.issueBackingStatementWithIndex(t, 4, candidateHash, session)

		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{
			getCandidateBackedEvent(t, candidateReceipt),
		})

		message = disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote1,
						ValidatorIndex:         3,
					},
					{
						SignedDisputeStatement: validVote2,
						ValidatorIndex:         4,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, message, message.ResponseChannel)
		require.NoError(t, err)
		importResult, ok := res.(ImportStatementResult)
		require.True(t, ok)
		require.Equal(t, ValidImport, importResult)
	})

	t.Run("conflicting_votes_lead_to_dispute_participation", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		ts.run(t, &session)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)

		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{
			getCandidateBackedEvent(t, candidateReceipt),
		})

		validVote, invalidVote1 := ts.generateOpposingVotesPair(t,
			3,
			1,
			candidateHash,
			session,
			BackingVote,
		)
		invalidVote2 := ts.issueExplicitStatementWithIndex(t, 2, candidateHash, session, false)

		message := disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote,
						ValidatorIndex:         3,
					},
					{
						SignedDisputeStatement: invalidVote1,
						ValidatorIndex:         1,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)

		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, candidateReceipt.CommitmentsHash)

		// after participation
		disputesMessage := disputetypes.Message[disputetypes.ActiveDisputes]{
			Data:            disputetypes.ActiveDisputes{},
			ResponseChannel: make(chan any),
		}
		res, err := call(ts.subsystemReceiver, disputesMessage, disputesMessage.ResponseChannel)
		require.NoError(t, err)
		disputeBtree, ok := res.(scale.BTree)
		require.True(t, ok)
		require.Equal(t, 1, disputeBtree.Len())
		dispute := disputeBtree.Min().(*disputetypes.Dispute)
		require.Equal(t, session, dispute.Comparator.SessionIndex)
		require.Equal(t, candidateHash, dispute.Comparator.CandidateHash)
		isActive, err := dispute.DisputeStatus.IsActive()
		require.NoError(t, err)
		require.True(t, isActive)

		query := disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, query, query.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok := res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, 1, queryResponse[0].Votes.Invalid.Len())
		require.Equal(t, 2, queryResponse[0].Votes.Valid.Value.Len())

		message = disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: invalidVote2,
						ValidatorIndex:         2,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)

		query = disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, query, query.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok = res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, 2, queryResponse[0].Votes.Invalid.Len())
		require.Equal(t, 2, queryResponse[0].Votes.Valid.Value.Len())
	})

	t.Run("positive_votes_dont_trigger_participation", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		ts.run(t, &session)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)

		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{
			getCandidateBackedEvent(t, candidateReceipt),
		})

		validVote1 := ts.issueExplicitStatementWithIndex(t, 2, candidateHash, session, true)
		validVote2 := ts.issueExplicitStatementWithIndex(t, 1, candidateHash, session, true)

		message := disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote1,
						ValidatorIndex:         2,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)

		disputeQuery := disputetypes.Message[disputetypes.ActiveDisputes]{
			Data:            disputetypes.ActiveDisputes{},
			ResponseChannel: make(chan any),
		}
		res, err := call(ts.subsystemReceiver, disputeQuery, disputeQuery.ResponseChannel)
		require.NoError(t, err)
		disputeBtree, ok := res.(scale.BTree)
		require.True(t, ok)
		require.Equal(t, 0, disputeBtree.Len())

		query := disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, query, query.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok := res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, 0, queryResponse[0].Votes.Invalid.Len())
		require.Equal(t, 1, queryResponse[0].Votes.Valid.Value.Len())

		message = disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote2,
						ValidatorIndex:         1,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)

		disputeQuery = disputetypes.Message[disputetypes.ActiveDisputes]{
			Data:            disputetypes.ActiveDisputes{},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, disputeQuery, disputeQuery.ResponseChannel)
		require.NoError(t, err)
		disputeBtree, ok = res.(scale.BTree)
		require.True(t, ok)
		require.Equal(t, 0, disputeBtree.Len())

		query = disputetypes.Message[disputetypes.QueryCandidateVotes]{
			Data: disputetypes.QueryCandidateVotes{
				Queries: []disputetypes.CandidateVotesMessage{
					{
						Session:       session,
						CandidateHash: candidateHash,
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, query, query.ResponseChannel)
		require.NoError(t, err)
		queryResponse, ok = res.([]disputetypes.QueryCandidateVotesResponse)
		require.True(t, ok)
		require.Equal(t, 1, len(queryResponse))
		require.Equal(t, 0, queryResponse[0].Votes.Invalid.Len())
		require.Equal(t, 2, queryResponse[0].Votes.Valid.Value.Len())
	})

	t.Run("wrong_validator_index_is_ignored", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		ts.run(t, &session)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)

		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{})

		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			2,
			1,
			candidateHash,
			session,
			ExplicitVote,
		)

		message := disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote,
						ValidatorIndex:         1,
					},
					{
						SignedDisputeStatement: invalidVote,
						ValidatorIndex:         2,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)

		disputeQuery := disputetypes.Message[disputetypes.ActiveDisputes]{
			Data:            disputetypes.ActiveDisputes{},
			ResponseChannel: make(chan any),
		}
		res, err := call(ts.subsystemReceiver, disputeQuery, disputeQuery.ResponseChannel)
		require.NoError(t, err)
		disputeBtree, ok := res.(scale.BTree)
		require.True(t, ok)
		require.Equal(t, 0, disputeBtree.Len())
	})

	t.Run("finality_votes_ignore_disputed_candidates", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		ts.run(t, &session)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)
		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{
			getCandidateBackedEvent(t, candidateReceipt),
		})

		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			2,
			1,
			candidateHash,
			session,
			ExplicitVote,
		)

		message := disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote,
						ValidatorIndex:         2,
					},
					{
						SignedDisputeStatement: invalidVote,
						ValidatorIndex:         1,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, candidateReceipt.CommitmentsHash)

		baseBlock := bytes.Repeat([]byte{0x0f}, 32)
		blockHashA := bytes.Repeat([]byte{0x0a}, 32)
		blockHashB := bytes.Repeat([]byte{0x0b}, 32)

		chainMessage := disputetypes.Message[disputetypes.DetermineUndisputedChainMessage]{
			Data: disputetypes.DetermineUndisputedChainMessage{
				Base: overseer.Block{
					Number: 10,
					Hash:   common.Hash(baseBlock),
				},
				BlockDescriptions: []disputetypes.BlockDescription{
					{
						BlockHash: common.Hash(blockHashA),
						Session:   session,
						Candidates: []parachaintypes.CandidateHash{
							{
								candidateHash,
							},
						},
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err := call(ts.subsystemReceiver, chainMessage, chainMessage.ResponseChannel)
		require.NoError(t, err)
		chainResponse, ok := res.(disputetypes.DetermineUndisputedChainResponse)
		require.True(t, ok)
		require.Equal(t, uint32(10), chainResponse.Block.Number)
		require.Equal(t, common.Hash(baseBlock), chainResponse.Block.Hash)

		chainMessage = disputetypes.Message[disputetypes.DetermineUndisputedChainMessage]{
			Data: disputetypes.DetermineUndisputedChainMessage{
				Base: overseer.Block{
					Number: 10,
					Hash:   common.Hash(baseBlock),
				},
				BlockDescriptions: []disputetypes.BlockDescription{
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
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, chainMessage, chainMessage.ResponseChannel)
		require.NoError(t, err)

		chainResponse, ok = res.(disputetypes.DetermineUndisputedChainResponse)
		require.True(t, ok)
		require.NoError(t, chainResponse.Err)
		require.Equal(t, uint32(11), chainResponse.Block.Number)
		require.Equal(t, common.Hash(blockHashA), chainResponse.Block.Hash)
	})

	t.Run("supermajority_valid_dispute_may_be_finalized", func(t *testing.T) {
		t.Parallel()
		ts := newTestState(t)
		session := parachaintypes.SessionIndex(1)
		ts.run(t, &session)

		candidateReceipt := getValidCandidateReceipt(t)
		candidateHash, err := candidateReceipt.Hash()
		require.NoError(t, err)

		ts.activateLeafAtSession(t, session, 1, []parachaintypes.CandidateEventVDT{
			getCandidateBackedEvent(t, candidateReceipt),
		})

		superMajorityThreshold := getSuperMajorityThreshold(len(ts.validators))
		validVote, invalidVote := ts.generateOpposingVotesPair(t,
			2,
			1,
			candidateHash,
			session,
			ExplicitVote,
		)

		message := disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt,
				Session:          session,
				Statements: []disputetypes.Statement{
					{
						SignedDisputeStatement: validVote,
						ValidatorIndex:         2,
					},
					{
						SignedDisputeStatement: invalidVote,
						ValidatorIndex:         1,
					},
				},
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})
		handleParticipationWithDistribution(t, ts.mockOverseer, ts.runtime, candidateHash, candidateReceipt.CommitmentsHash)

		var statements []disputetypes.Statement
		for i := 0; i < superMajorityThreshold-1; i++ {
			validatorIndex := parachaintypes.ValidatorIndex(i + 3)
			vote := ts.issueExplicitStatementWithIndex(t, validatorIndex, candidateHash, session, true)
			statements = append(statements, disputetypes.Statement{
				SignedDisputeStatement: vote,
				ValidatorIndex:         validatorIndex,
			})
		}
		message = disputetypes.Message[disputetypes.ImportStatements]{
			Data: disputetypes.ImportStatements{
				CandidateReceipt: candidateReceipt,
				Session:          session,
				Statements:       statements,
			},
			ResponseChannel: nil,
		}
		err = sendMessage(ts.subsystemReceiver, message)
		require.NoError(t, err)
		handleApprovalVoteRequest(t, ts.mockOverseer, candidateHash, []overseer.ApprovalSignature{})

		blockHash := bytes.Repeat([]byte{0x0f}, 32)
		blockHashA := bytes.Repeat([]byte{0x0a}, 32)
		blockHashB := bytes.Repeat([]byte{0x0b}, 32)

		chainMessage := disputetypes.Message[disputetypes.DetermineUndisputedChainMessage]{
			Data: disputetypes.DetermineUndisputedChainMessage{
				Base: overseer.Block{
					Number: 10,
					Hash:   common.Hash(blockHash),
				},
				BlockDescriptions: []disputetypes.BlockDescription{
					{
						BlockHash:  common.Hash(blockHashA),
						Session:    session,
						Candidates: []parachaintypes.CandidateHash{{candidateHash}},
					},
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err := call(ts.subsystemReceiver, chainMessage, chainMessage.ResponseChannel)
		require.NoError(t, err)
		chainResponse, ok := res.(disputetypes.DetermineUndisputedChainResponse)
		require.True(t, ok)
		require.NoError(t, chainResponse.Err)
		require.Equal(t, uint32(11), chainResponse.Block.Number)
		require.Equal(t, common.Hash(blockHashA), chainResponse.Block.Hash)

		chainMessage = disputetypes.Message[disputetypes.DetermineUndisputedChainMessage]{
			Data: disputetypes.DetermineUndisputedChainMessage{
				Base: overseer.Block{
					Number: 10,
					Hash:   common.Hash(blockHash),
				},
				BlockDescriptions: []disputetypes.BlockDescription{
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
				},
			},
			ResponseChannel: make(chan any),
		}
		res, err = call(ts.subsystemReceiver, chainMessage, chainMessage.ResponseChannel)
		require.NoError(t, err)
		chainResponse, ok = res.(disputetypes.DetermineUndisputedChainResponse)
		require.True(t, ok)
		require.NoError(t, chainResponse.Err)
		require.Equal(t, uint32(12), chainResponse.Block.Number)
		require.Equal(t, common.Hash(blockHashB), chainResponse.Block.Hash)
	})
}
