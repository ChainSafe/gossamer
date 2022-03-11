// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration
// +build integration

package babe

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func newTestVerificationManager(t *testing.T, genCfg *types.BabeConfiguration) *VerificationManager {
	testDatadirPath := t.TempDir()

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	config := state.Config{
		Path:      testDatadirPath,
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
	}

	dbSrv := state.NewService(config)
	dbSrv.UseMemDB()

	gen, genTrie, genHeader := genesis.NewDevGenesisWithTrieAndHeader(t)
	err := dbSrv.Initialise(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = dbSrv.Start()
	require.NoError(t, err)

	if genCfg == nil {
		genCfg = genesisBABEConfig
	}

	dbSrv.Epoch, err = state.NewEpochStateFromGenesis(dbSrv.DB(), dbSrv.Block, genCfg)
	require.NoError(t, err)

	logger.Patch(log.SetLevel(defaultTestLogLvl))

	vm, err := NewVerificationManager(dbSrv.Block, dbSrv.Epoch)
	require.NoError(t, err)
	return vm
}

func TestVerificationManager_OnDisabled_InvalidIndex(t *testing.T) {
	vm := newTestVerificationManager(t, nil)
	babeService := createTestService(t, nil)
	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	err = vm.SetOnDisabled(1, &block.Header)
	require.Equal(t, err, ErrInvalidBlockProducerIndex)
}

func TestVerificationManager_OnDisabled_NewDigest(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)
	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	vm := newTestVerificationManager(t, nil)
	vm.epochInfo[testEpochIndex] = &verifierInfo{
		authorities: epochData.authorities,
		threshold:   epochData.threshold,
		randomness:  epochData.randomness,
	}

	parent, _ := babeService.blockState.BestBlockHeader()

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)

	// create an OnDisabled change on a different branch
	block = createTestBlock(t, babeService, parent, [][]byte{}, 2, testEpochIndex, epochData)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_OnDisabled_DuplicateDigest(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)
	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	vm := newTestVerificationManager(t, nil)
	vm.epochInfo[testEpochIndex] = &verifierInfo{
		authorities: epochData.authorities,
		threshold:   epochData.threshold,
		randomness:  epochData.randomness,
	}

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)

	// create an OnDisabled change on a different branch
	block2 := createTestBlock(t, babeService, &block.Header, [][]byte{}, 2, testEpochIndex, epochData)
	err = vm.blockState.AddBlock(block2)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block2.Header)
	require.Equal(t, ErrAuthorityAlreadyDisabled, err)
}

func TestVerificationManager_VerifyBlock_Ok(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1

	vm := newTestVerificationManager(t, cfg)

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	err = vm.VerifyBlock(&block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_Secondary(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1
	cfg.SecondarySlots = 0

	vm := newTestVerificationManager(t, cfg)

	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	dig := createSecondaryVRFPreDigest(t, kp, 0, uint64(0), uint64(0), Randomness{})

	bd := types.NewBabeDigest()
	err = bd.Set(dig)
	require.NoError(t, err)

	bdEnc, err := scale.Marshal(bd)
	require.NoError(t, err)

	// create pre-digest
	preDigest := &types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              bdEnc,
	}

	// create new block header
	const number uint = 1
	digest := types.NewDigest()
	err = digest.Add(*preDigest)
	require.NoError(t, err)

	// create seal and add to digest
	seal := &types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{0},
	}
	require.NoError(t, err)

	err = digest.Add(*seal)
	require.NoError(t, err)

	header, err := types.NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, number, digest)
	require.NoError(t, err)

	block := types.Block{
		Header: *header,
		Body:   nil,
	}
	err = vm.VerifyBlock(&block.Header)
	require.EqualError(t, err, "failed to verify pre-runtime digest: could not verify slot claim VRF proof")
}

func TestVerificationManager_VerifyBlock_MultipleEpochs(t *testing.T) {
	t.Skip() // TODO: no idea why it's complaining it can't find the epoch data. fix later
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1

	vm := newTestVerificationManager(t, cfg)

	futureEpoch := uint64(5)

	err = vm.epochState.(*state.EpochState).SetEpochData(futureEpoch, &types.EpochData{
		Authorities: epochData.authorities,
		Randomness:  epochData.randomness,
	})
	require.NoError(t, err)

	futureEpochData, err := babeService.initiateEpoch(futureEpoch)
	require.NoError(t, err)

	// create block in future epoch
	block1 := createTestBlock(t, babeService, genesisHeader, [][]byte{},
		cfg.EpochLength*futureEpoch+1, futureEpoch, futureEpochData)
	block2 := createTestBlock(t, babeService, &block1.Header, [][]byte{},
		cfg.EpochLength*futureEpoch+2, futureEpoch, futureEpochData)

	err = vm.VerifyBlock(&block2.Header)
	require.NoError(t, err)

	// create block in epoch 1
	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, cfg.EpochLength-10, testEpochIndex, epochData)

	err = vm.VerifyBlock(&block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_InvalidBlockOverThreshold(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 100

	vm := newTestVerificationManager(t, cfg)

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)

	err = vm.VerifyBlock(&block.Header)
	require.Equal(t, ErrVRFOutputOverThreshold, errors.Unwrap(err))
}

func TestVerificationManager_VerifyBlock_InvalidBlockAuthority(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	cfg.C1 = 1
	cfg.C2 = 1
	cfg.GenesisAuthorities = []types.AuthorityRaw{}

	vm := newTestVerificationManager(t, cfg)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)

	err = vm.VerifyBlock(&block.Header)
	require.Equal(t, ErrInvalidBlockProducerIndex, errors.Unwrap(err))
}

func TestVerifyPimarySlotWinner(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)
	epochData, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	// create proof that we can authorize this block
	epochData.threshold = maxThreshold
	epochData.authorityIndex = 0

	const slotNumber uint64 = 1

	preRuntimeDigest, err := claimSlot(testEpochIndex, slotNumber, epochData, babeService.keypair)
	require.NoError(t, err)

	babePreDigest, err := types.DecodeBabePreDigest(preRuntimeDigest.Data)
	require.NoError(t, err)

	d, ok := babePreDigest.(types.BabePrimaryPreDigest)
	require.True(t, ok)

	Authorities := make([]types.Authority, 1)
	Authorities[0] = types.Authority{
		Key: kp.Public().(*sr25519.PublicKey),
	}
	epochData.authorities = Authorities

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: epochData.authorities,
		threshold:   epochData.threshold,
		randomness:  epochData.randomness,
	})
	require.NoError(t, err)

	ok, err = verifier.verifyPrimarySlotWinner(d.AuthorityIndex, slotNumber, d.VRFOutput, d.VRFProof)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestVerifyAuthorshipRight(t *testing.T) {
	babeService := createTestService(t, nil)
	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)
	epochData.threshold = maxThreshold

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: epochData.authorities,
		threshold:   epochData.threshold,
		randomness:  epochData.randomness,
	})
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(&block.Header)
	require.NoError(t, err)
}

func TestVerifyAuthorshipRight_Equivocation(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)
	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	epochData.threshold = maxThreshold
	epochData.authorities = []types.Authority{
		{
			Key: kp.Public().(*sr25519.PublicKey),
		},
	}

	// create and add first block
	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	block.Header.Hash()

	err = babeService.blockState.AddBlock(block)
	require.NoError(t, err)

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: epochData.authorities,
		threshold:   epochData.threshold,
		randomness:  epochData.randomness,
	})
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(&block.Header)
	require.NoError(t, err)

	// create new block
	block2 := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	block2.Header.Hash()

	err = babeService.blockState.AddBlock(block2)
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(&block2.Header)
	require.Equal(t, ErrProducerEquivocated, err)
}

func TestVerifyForkBlocksWithRespectiveEpochData(t *testing.T) {
	/*
	* Setup the services: StateService, DigestHandler, EpochState
	* and VerificationManager
	 */
	authorities := []types.AuthorityRaw{
		{
			Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes(),
		},
		{
			Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes(),
		},
		{
			Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes(),
		},
		{
			Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes(),
		},
		{
			Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes(),
		},
		{
			Key: keyring.KeyFerdie.Public().(*sr25519.PublicKey).AsBytes(),
		},
		{
			Key: keyring.KeyGeorge.Public().(*sr25519.PublicKey).AsBytes(),
		},
		{
			Key: keyring.KeyHeather.Public().(*sr25519.PublicKey).AsBytes(),
		},
		{
			Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes(),
		},
	}

	// starts with only 3 authorities in the authority set
	epochBABEConfig := &types.BabeConfiguration{
		SlotDuration:       1000,
		EpochLength:        10,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: authorities[:3],
		Randomness:         [32]byte{},
		SecondarySlots:     0,
	}

	genesis, trie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(
		telemetry.NewNotifyFinalized(
			genesisHeader.Hash(),
			fmt.Sprint(genesisHeader.Number),
		),
	)

	stateService := state.NewService(state.Config{
		Path:      t.TempDir(),
		Telemetry: telemetryMock,
	})

	stateService.UseMemDB()

	err := stateService.Initialise(genesis, genesisHeader, trie)
	require.NoError(t, err)

	inMemoryDB, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
		DataDir:  t.TempDir(),
	})
	require.NoError(t, err)

	epochState, err := state.NewEpochStateFromGenesis(inMemoryDB, stateService.Block, epochBABEConfig)
	require.NoError(t, err)

	digestHandler, err := digest.NewHandler(log.DoNotChange, stateService.Block, epochState, stateService.Grandpa)
	require.NoError(t, err)

	digestHandler.Start()

	verificationManager, err := NewVerificationManager(stateService.Block, epochState)
	require.NoError(t, err)

	/*
	* lets issue different blocks starting from genesis (a fork)
	 */
	aliceBlockNextEpoch := types.NextEpochData{
		Authorities: authorities[3:],
	}
	aliceBlockNextConfigData := types.NextConfigData{
		C1:             9,
		C2:             10,
		SecondarySlots: 1,
	}
	aliceBlockHeader := issueConsensusDigestsBlockFromGenesis(t, genesisHeader, keyring.KeyAlice,
		stateService, aliceBlockNextEpoch, aliceBlockNextConfigData)

	bobBlockNextEpoch := types.NextEpochData{
		Authorities: authorities[6:],
	}
	bobBlockNextConfigData := types.NextConfigData{
		C1:             3,
		C2:             8,
		SecondarySlots: 1,
	}
	bobBlockHeader := issueConsensusDigestsBlockFromGenesis(t, genesisHeader, keyring.KeyBob,
		stateService, bobBlockNextEpoch, bobBlockNextConfigData)

	// wait for digest handleBlockImport goroutine gets the imported
	// block, process its digest and store the info at epoch state
	time.Sleep(time.Second * 2)

	/*
	* Simulate a fork from the genesis file, the fork alice and the fork bob
	* contains different digest handlers.
	 */
	const chainLen = 5
	forkAliceChain := make([]types.Header, chainLen)
	forkBobChain := make([]types.Header, chainLen)

	forkAliceLastHeader := aliceBlockHeader
	forkBobLastHeader := bobBlockHeader

	for idx := 0; idx < chainLen; idx++ {
		forkAliceLastHeader = issueNewBlockFrom(t, forkAliceLastHeader,
			keyring.KeyAlice, stateService)

		forkBobLastHeader = issueNewBlockFrom(t, forkBobLastHeader,
			keyring.KeyBob, stateService)

		forkAliceChain[idx] = *forkAliceLastHeader
		forkBobChain[idx] = *forkBobLastHeader
	}

	// verify if each block from the fork alice get the right digest
	const epochToTest = 1

	expectedThreshold, err := CalculateThreshold(aliceBlockNextConfigData.C1,
		aliceBlockNextConfigData.C2, len(authorities[3:]))
	for _, headerToVerify := range forkAliceChain {
		verifierInfo, err := verificationManager.getVerifierInfo(epochToTest, &headerToVerify)
		require.NoError(t, err)

		require.Equal(t, len(authorities[3:]), len(verifierInfo.authorities))
		rawAuthorities := make([]types.AuthorityRaw, len(verifierInfo.authorities))

		for i, auth := range verifierInfo.authorities {
			rawAuthorities[i] = *auth.ToRaw()
		}

		require.ElementsMatch(t, authorities[3:], rawAuthorities)
		require.True(t, verifierInfo.secondarySlots)
		require.Equal(t, expectedThreshold, verifierInfo.threshold)
	}

	// each block from the fork bob should use the right digest

	expectedThreshold, err = CalculateThreshold(bobBlockNextConfigData.C1,
		bobBlockNextConfigData.C2, len(authorities[6:]))

	for _, headerToVerify := range forkBobChain {
		verifierInfo, err := verificationManager.getVerifierInfo(epochToTest, &headerToVerify)
		require.NoError(t, err)

		require.Equal(t, len(authorities[6:]), len(verifierInfo.authorities))
		rawAuthorities := make([]types.AuthorityRaw, len(verifierInfo.authorities))

		for i, auth := range verifierInfo.authorities {
			rawAuthorities[i] = *auth.ToRaw()
		}

		// should keep the original authorities
		require.ElementsMatch(t, authorities[6:], rawAuthorities)
		require.True(t, verifierInfo.secondarySlots)
		require.Equal(t, expectedThreshold, verifierInfo.threshold)
	}

	telemetryMock.EXPECT().SendMessage(
		telemetry.NewNotifyFinalized(
			forkBobLastHeader.Hash(),
			fmt.Sprint(forkBobLastHeader.Number),
		),
	)
	err = stateService.Block.SetFinalisedHash(forkBobLastHeader.Hash(), 1, 1)
	require.NoError(t, err)

	// wait for digest handleBlockFinalize goroutine gets the finalized
	// block, clean up the in memory data and store the finalized digest in db
	time.Sleep(time.Second * 2)

	// should not exists in memory data as a chain was finalized
	_, err = epochState.GetEpochDataForHeader(epochToTest, forkBobLastHeader)
	require.EqualError(t, err, fmt.Sprintf("epoch %d not found in memory stored epoch data", epochToTest))

	_, err = epochState.GetConfigDataForHeader(epochToTest, forkBobLastHeader)
	require.EqualError(t, err, fmt.Sprintf("epoch %d not found in memory stored config data", epochToTest))

	// as a chain was finalized any block built upon it should use the database stored data
	blockUponFinalizedHeader := issueNewBlockFrom(t, forkBobLastHeader,
		keyring.KeyBob, stateService)

	verifierInfo, err := verificationManager.getVerifierInfo(epochToTest, blockUponFinalizedHeader)
	require.NoError(t, err)

	require.Equal(t, len(authorities[6:]), len(verifierInfo.authorities))
	rawAuthorities := make([]types.AuthorityRaw, len(verifierInfo.authorities))

	for i, auth := range verifierInfo.authorities {
		rawAuthorities[i] = *auth.ToRaw()
	}

	// should keep the original authorities
	require.ElementsMatch(t, authorities[6:], rawAuthorities)
	require.True(t, verifierInfo.secondarySlots)
	require.Equal(t, expectedThreshold, verifierInfo.threshold)
}

// issueConsensusDigestsBlocksFromGenesis will create diferent
// blocks that contains different consensus messages digests
func issueConsensusDigestsBlockFromGenesis(t *testing.T, genesisHeader *types.Header,
	kp *sr25519.Keypair, stateService *state.Service,
	nextEpoch types.NextEpochData, nextConfig types.NextConfigData) *types.Header {
	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(0), 0))
	require.NoError(t, err)

	babePrimaryPreDigest := types.BabePrimaryPreDigest{
		SlotNumber: 1,
		VRFOutput:  output,
		VRFProof:   proof,
	}

	preRuntimeDigest, err := babePrimaryPreDigest.ToPreRuntimeDigest()
	require.NoError(t, err)

	babeConsensusDigestNextEpoch := types.NewBabeConsensusDigest()
	require.NoError(t, babeConsensusDigestNextEpoch.Set(nextEpoch))

	babeConsensusDigestNextConfigData := types.NewBabeConsensusDigest()
	require.NoError(t, babeConsensusDigestNextConfigData.Set(nextConfig))

	nextEpochData, err := scale.Marshal(babeConsensusDigestNextEpoch)
	require.NoError(t, err)

	nextEpochConsensusDigest := types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              nextEpochData,
	}

	nextConfigData, err := scale.Marshal(babeConsensusDigestNextConfigData)
	require.NoError(t, err)

	nextConfigConsensusDigest := types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              nextConfigData,
	}

	digest := types.NewDigest()
	require.NoError(t, digest.Add(*preRuntimeDigest, nextEpochConsensusDigest, nextConfigConsensusDigest))

	headerWhoOwnsNextEpochDigest := &types.Header{
		ParentHash: genesisHeader.Hash(),
		Number:     1,
		Digest:     digest,
	}

	err = stateService.Block.AddBlock(&types.Block{
		Header: *headerWhoOwnsNextEpochDigest,
		Body:   *types.NewBody([]types.Extrinsic{}),
	})
	require.NoError(t, err)

	return headerWhoOwnsNextEpochDigest
}

// issueNewBlockFrom will create and store a new block following a chain
func issueNewBlockFrom(t *testing.T, parentHeader *types.Header, kp *sr25519.Keypair, stateService *state.Service) *types.Header {
	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	require.NoError(t, err)

	babePrimaryPreDigest := types.BabePrimaryPreDigest{
		SlotNumber: 1,
		VRFOutput:  output,
		VRFProof:   proof,
	}

	preRuntimeDigest, err := babePrimaryPreDigest.ToPreRuntimeDigest()
	require.NoError(t, err)

	digest := scale.NewVaryingDataTypeSlice(scale.MustNewVaryingDataType(
		types.PreRuntimeDigest{}))

	require.NoError(t, digest.Add(*preRuntimeDigest))

	header := &types.Header{
		ParentHash: parentHeader.Hash(),
		Number:     parentHeader.Number + 1,
		Digest:     digest,
	}

	err = stateService.Block.AddBlock(&types.Block{
		Header: *header,
		Body:   *types.NewBody([]types.Extrinsic{}),
	})
	require.NoError(t, err)

	return header
}
