// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package babe

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/pkg/scale"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func newTestVerificationManager(t *testing.T, genCfg *types.BabeConfiguration) *VerificationManager {
	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)

	config := state.Config{
		Path:     testDatadirPath,
		LogLevel: log.LvlInfo,
	}
	dbSrv := state.NewService(config)
	dbSrv.UseMemDB()

	if genCfg == nil {
		genCfg = genesisBABEConfig
	}

	gen, genTrie, genHeader := genesis.NewDevGenesisWithTrieAndHeader(t)
	err = dbSrv.Initialise(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = dbSrv.Start()
	require.NoError(t, err)
	dbSrv.Epoch, err = state.NewEpochStateFromGenesis(dbSrv.DB(), dbSrv.Block, genCfg)
	require.NoError(t, err)

	logger = log.New("pkg", "babe")
	h := log.StreamHandler(os.Stdout, log.TerminalFormat())
	h = log.CallerFileHandler(h)
	logger.SetHandler(log.LvlFilterHandler(defaultTestLogLvl, h))

	vm, err := NewVerificationManager(dbSrv.Block, dbSrv.Epoch)
	require.NoError(t, err)
	return vm
}

func TestVerificationManager_OnDisabled_InvalidIndex(t *testing.T) {
	vm := newTestVerificationManager(t, nil)

	babeService := createTestService(t, nil)
	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)
	err := vm.SetOnDisabled(1, &block.Header)
	require.Equal(t, err, ErrInvalidBlockProducerIndex)
}

func TestVerificationManager_OnDisabled_NewDigest(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)

	vm := newTestVerificationManager(t, nil)
	vm.epochInfo[testEpochIndex] = &verifierInfo{
		authorities: babeService.epochData.authorities,
		threshold:   babeService.epochData.threshold,
		randomness:  babeService.epochData.randomness,
	}

	parent, _ := babeService.blockState.BestBlockHeader()
	block, _ := createTestBlock(t, babeService, parent, [][]byte{}, 1, testEpochIndex)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)

	// create an OnDisabled change on a different branch
	block, _ = createTestBlock(t, babeService, parent, [][]byte{}, 2, testEpochIndex)
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

	vm := newTestVerificationManager(t, nil)
	vm.epochInfo[testEpochIndex] = &verifierInfo{
		authorities: babeService.epochData.authorities,
		threshold:   babeService.epochData.threshold,
		randomness:  babeService.epochData.randomness,
	}

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)

	// create an OnDisabled change on a different branch
	block2, _ := createTestBlock(t, babeService, &block.Header, [][]byte{}, 2, testEpochIndex)
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

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(babeService.epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1

	vm := newTestVerificationManager(t, cfg)

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)
	err = vm.VerifyBlock(&block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_Secondary(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(babeService.epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1
	cfg.SecondarySlots = 0

	vm := newTestVerificationManager(t, cfg)

	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	dig := createSecondaryVRFPreDigest(t, kp, 0, uint64(0), uint64(0), Randomness{})

	// todo (ed) remove after fix (see test below TestCreateSecondaryVRFPreDigestMarshal
	fmt.Printf("dig %T\n", dig)

	digEnc, err := scale.Marshal(dig)
	require.NoError(t, err)

	babePreDigest, err := types.DecodeBabePreDigest(digEnc)
	// todo (ed) remove after fix (see test below TestCreateSecondaryVRFPreDigestMarshal
	fmt.Printf("decoded %T\n", babePreDigest)

	// create pre-digest
	preDigest := &types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              digEnc,
	}

	// create new block header
	number := big.NewInt(1)
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

func TestCreateSecondaryVRFPreDigestMarshal(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	dig := createSecondaryVRFPreDigest(t, kp, 0, uint64(0), uint64(0), Randomness{})
	fmt.Printf("dig %T\n", dig)

	require.IsType(t, types.BabeSecondaryVRFPreDigest{}, *dig)

	// This used to be done with dig.Encode()
	digEnc, err := scale.Marshal(dig)
	require.NoError(t, err)

	babeDigest := types.NewBabeDigest()
	err = scale.Unmarshal(digEnc, &babeDigest)
	require.NoError(t, err)
	// Why isn't this type types.BabeSecondaryVRFPreDigest?
	require.IsType(t, types.BabePrimaryPreDigest{}, babeDigest)

	babePreDigest, err := types.DecodeBabePreDigest(digEnc)
	require.NoError(t, err)
	// Why isn't this type types.BabeSecondaryVRFPreDigest?
	require.IsType(t, types.BabePrimaryPreDigest{}, babePreDigest)
}

func TestVerificationManager_VerifyBlock_MultipleEpochs(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(babeService.epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1

	vm := newTestVerificationManager(t, cfg)

	futureEpoch := uint64(5)

	err = vm.epochState.(*state.EpochState).SetEpochData(futureEpoch, &types.EpochData{
		Authorities: babeService.epochData.authorities,
		Randomness:  babeService.epochData.randomness,
	})
	require.NoError(t, err)

	// create block in future epoch
	block1, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, cfg.EpochLength*futureEpoch+1, futureEpoch)
	block2, _ := createTestBlock(t, babeService, &block1.Header, [][]byte{}, cfg.EpochLength*futureEpoch+2, futureEpoch)

	err = vm.VerifyBlock(&block2.Header)
	require.NoError(t, err)

	// create block in epoch 1
	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, cfg.EpochLength-10, testEpochIndex)

	err = vm.VerifyBlock(&block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_InvalidBlockOverThreshold(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(babeService.epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 100

	vm := newTestVerificationManager(t, cfg)

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)

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

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)

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

	// create proof that we can authorize this block
	babeService.epochData.threshold = maxThreshold
	babeService.epochData.authorityIndex = 0

	builder, _ := NewBlockBuilder(
		babeService.keypair,
		babeService.transactionState,
		babeService.blockState,
		babeService.slotToProof,
		babeService.epochData.authorityIndex,
	)

	var slotNumber uint64 = 1

	addAuthorshipProof(t, babeService, slotNumber, testEpochIndex)
	duration, err := time.ParseDuration("1s")
	require.NoError(t, err)

	slot := Slot{
		start:    time.Now(),
		duration: duration,
		number:   slotNumber,
	}

	// create babe header
	babeHeader, err := builder.buildBlockBABEPrimaryPreDigest(slot)
	require.NoError(t, err)

	Authorities := make([]types.Authority, 1)
	Authorities[0] = types.Authority{
		Key: kp.Public().(*sr25519.PublicKey),
	}
	babeService.epochData.authorities = Authorities

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: babeService.epochData.authorities,
		threshold:   babeService.epochData.threshold,
		randomness:  babeService.epochData.randomness,
	})
	require.NoError(t, err)

	ok, err := verifier.verifyPrimarySlotWinner(babeHeader.AuthorityIndex, slot.number, babeHeader.VRFOutput, babeHeader.VRFProof)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestVerifyAuthorshipRight(t *testing.T) {
	babeService := createTestService(t, nil)
	babeService.epochData.threshold = maxThreshold

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: babeService.epochData.authorities,
		threshold:   babeService.epochData.threshold,
		randomness:  babeService.epochData.randomness,
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
	babeService.epochData.threshold = maxThreshold

	babeService.epochData.authorities = make([]types.Authority, 1)
	babeService.epochData.authorities[0] = types.Authority{
		Key: kp.Public().(*sr25519.PublicKey),
	}

	// create and add first block
	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)
	block.Header.Hash()

	err = babeService.blockState.AddBlock(block)
	require.NoError(t, err)

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: babeService.epochData.authorities,
		threshold:   babeService.epochData.threshold,
		randomness:  babeService.epochData.randomness,
	})
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(&block.Header)
	require.NoError(t, err)

	// create new block
	block2, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)
	block2.Header.Hash()

	err = babeService.blockState.AddBlock(block2)
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(&block2.Header)
	require.Equal(t, ErrProducerEquivocated, err)
}
