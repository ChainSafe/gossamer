// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration
// +build integration

package modules

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core"
	coremocks "github.com/ChainSafe/gossamer/dot/core/mocks"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type useRuntimeInstace func(*testing.T, *wasmer.Config) (runtime.Instance, error)

// useInstanceFromGenesis creates a new runtime instance given a genesis file
func useInstanceFromGenesis(t *testing.T, cfg *wasmer.Config) (instance runtime.Instance, err error) {
	t.Helper()
	return wasmer.NewRuntimeFromGenesis(cfg)
}

func useInstanceFromRuntimeV0910(t *testing.T, cfg *wasmer.Config) (instance runtime.Instance, err error) {
	t.Helper()
	runtimePath := runtime.GetAbsolutePath(runtime.POLKADOT_RUNTIME_FP_v0910)
	return wasmer.NewInstanceFromFile(runtimePath, cfg)
}

func TestMain(m *testing.M) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	if err != nil {
		log.Errorf("failed to generate runtime wasm file: %s", err)
		os.Exit(1)
	}

	// Start all tests
	code := m.Run()

	runtime.RemoveFiles(wasmFilePaths)
	os.Exit(code)
}

func TestAuthorModule_Pending_Integration(t *testing.T) {
	t.Parallel()

	tmpdir := t.TempDir()

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.
		EXPECT().
		SendMessage(gomock.Any()).
		AnyTimes()

	state2test := state.NewService(state.Config{LogLevel: log.DoNotChange, Path: tmpdir})
	state2test.UseMemDB()
	state2test.Transaction = state.NewTransactionState(telemetryMock)

	auth := newAuthorModule(t, &integrationTestController{stateSrv: state2test})
	res := new(PendingExtrinsicsResponse)
	err := auth.PendingExtrinsics(nil, nil, res)

	require.NoError(t, err)
	require.Equal(t, PendingExtrinsicsResponse([]string{}), *res)

	vtx := &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic([]byte{0x01, 0x02}),
		Validity:  new(transaction.Validity),
	}

	_, err = state2test.Transaction.Push(vtx)
	require.NoError(t, err)

	err = auth.PendingExtrinsics(nil, nil, res)
	require.NoError(t, err)

	expected := common.BytesToHex(vtx.Extrinsic)
	require.Equal(t, PendingExtrinsicsResponse([]string{expected}), *res)
}

func TestAuthorModule_SubmitExtrinsic_Integration(t *testing.T) {
	t.Parallel()
	tmpbasepath := t.TempDir()

	intCtrl := setupStateAndPopulateTrieState(t, tmpbasepath, useInstanceFromRuntimeV0910)
	intCtrl.stateSrv.Transaction = state.NewTransactionState()

	genesisHash := intCtrl.genesisHeader.Hash()
	blockHash := intCtrl.stateSrv.Block.BestBlockHash()

	// creating an extrisinc to the System.remark call using a sample argument
	extHex := runtime.NewTestExtrinsic(t,
		intCtrl.runtime, genesisHash, blockHash, 0, "System.remark", []byte{0xab, 0xcd})

	extBytes := common.MustHexToBytes(extHex)

	ctrl := gomock.NewController(t)
	net2test := coremocks.NewMockNetwork(ctrl)
	net2test.EXPECT().GossipMessage(&network.TransactionMessage{Extrinsics: []types.Extrinsic{extBytes}})
	intCtrl.network = net2test

	// setup auth module
	auth := newAuthorModule(t, intCtrl)

	ext := Extrinsic{extHex}

	res := new(ExtrinsicHashResponse)
	err := auth.SubmitExtrinsic(nil, &ext, res)
	require.Nil(t, err)

	expectedExtrinsic := types.NewExtrinsic(extBytes)
	expected := &transaction.ValidTransaction{
		Extrinsic: expectedExtrinsic,
		Validity: &transaction.Validity{
			Priority:  39325240425794630,
			Requires:  nil,
			Provides:  [][]byte{{212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 0, 0, 0, 0}}, // nolint:lll
			Longevity: 18446744073709551614,
			Propagate: true,
		},
	}

	expectedHash := ExtrinsicHashResponse(expectedExtrinsic.Hash().String())
	txOnPool := intCtrl.stateSrv.Transaction.PendingInPool()

	// compare results
	require.Len(t, txOnPool, 1)
	require.Equal(t, expected, txOnPool[0])
	require.Equal(t, expectedHash, *res)
}

func TestAuthorModule_SubmitExtrinsic_invalid(t *testing.T) {
	t.Parallel()
	tmpbasepath := t.TempDir()

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.
		EXPECT().
		SendMessage(gomock.Any()).
		AnyTimes()

	intCtrl := setupStateAndRuntime(t, tmpbasepath, useInstanceFromRuntimeV0910)
	intCtrl.stateSrv.Transaction = state.NewTransactionState(telemetryMock)

	genesisHash := intCtrl.genesisHeader.Hash()

	// creating an extrisinc to the System.remark call using a sample argument
	extHex := runtime.NewTestExtrinsic(t,
		intCtrl.runtime, genesisHash, genesisHash, 0, "System.remark", []byte{})

	ctrl := gomock.NewController(t)
	net2test := coremocks.NewMockNetwork(ctrl)
	net2test.EXPECT().GossipMessage(nil).MaxTimes(0)

	intCtrl.network = net2test

	// setup auth module
	auth := newAuthorModule(t, intCtrl)

	ext := Extrinsic{extHex}

	res := new(ExtrinsicHashResponse)
	err := auth.SubmitExtrinsic(nil, &ext, res)
	require.EqualError(t, err, runtime.ErrInvalidTransaction.Message)

	txOnPool := intCtrl.stateSrv.Transaction.PendingInPool()
	require.Len(t, txOnPool, 0)
}

func TestAuthorModule_SubmitExtrinsic_invalid_input(t *testing.T) {
	t.Parallel()
	tmppath := t.TempDir()

	// setup service
	// setup auth module
	intctrl := setupStateAndRuntime(t, tmppath, useInstanceFromGenesis)
	auth := newAuthorModule(t, intctrl)

	// create and submit extrinsic
	ext := Extrinsic{fmt.Sprintf("%x", "1")}

	res := new(ExtrinsicHashResponse)
	err := auth.SubmitExtrinsic(nil, &ext, res)
	require.EqualError(t, err, "could not byteify non 0x prefixed string: 0x31")
}

func TestAuthorModule_SubmitExtrinsic_AlreadyInPool(t *testing.T) {
	t.Parallel()

	tmpbasepath := t.TempDir()
	intCtrl := setupStateAndRuntime(t, tmpbasepath, useInstanceFromGenesis)
	intCtrl.stateSrv.Transaction = state.NewTransactionState(nil)

	genesisHash := intCtrl.genesisHeader.Hash()

	// creating an extrisinc to the System.remark call using a sample argument
	extHex := runtime.NewTestExtrinsic(t,
		intCtrl.runtime, genesisHash, genesisHash, 0, "System.remark", []byte{})
	extBytes := common.MustHexToBytes(extHex)

	ctrl := gomock.NewController(t)

	storageState := coremocks.NewMockStorageState(ctrl)
	// should not call storage.TrieState
	storageState.EXPECT().TrieState(nil).MaxTimes(0)
	intCtrl.storageState = storageState

	net2test := coremocks.NewMockNetwork(ctrl)
	// should not call network.GossipMessage
	net2test.EXPECT().GossipMessage(nil).MaxTimes(0)
	intCtrl.network = net2test

	// setup auth module
	auth := newAuthorModule(t, intCtrl)

	// create and submit extrinsic
	ext := Extrinsic{extHex}

	res := new(ExtrinsicHashResponse)

	expectedExtrinsic := types.NewExtrinsic(extBytes)
	expected := &transaction.ValidTransaction{
		Extrinsic: expectedExtrinsic,
		Validity: &transaction.Validity{
			Priority:  39325240425794630,
			Requires:  nil,
			Provides:  [][]byte{{212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 0, 0, 0, 0}}, // nolint:lll
			Longevity: 18446744073709551614,
			Propagate: true,
		},
	}

	intCtrl.stateSrv.Transaction.AddToPool(expected)

	// should not cause error, since a transaction
	err := auth.SubmitExtrinsic(nil, &ext, res)
	require.NoError(t, err)
}

func TestAuthorModule_InsertKey_Integration(t *testing.T) {
	tmppath := t.TempDir()

	intctrl := setupStateAndRuntime(t, tmppath, useInstanceFromGenesis)
	intctrl.keystore = keystore.NewGlobalKeystore()

	auth := newAuthorModule(t, intctrl)

	const seed = "0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309"

	babeKp, err := sr25519.NewKeypairFromSeed(common.MustHexToBytes(seed))
	require.NoError(t, err)

	grandKp, err := ed25519.NewKeypairFromSeed(common.MustHexToBytes(seed))
	require.NoError(t, err)

	testcases := map[string]struct {
		ksType, seed string
		kp           interface{}
		waitErr      error
	}{
		"insert a valid babe key type": {
			ksType: "babe",
			seed:   seed,
			kp:     babeKp,
		},

		"insert a valid gran key type": {
			ksType: "gran",
			seed:   seed,
			kp:     grandKp,
		},

		"invalid babe key type": {
			ksType:  "babe",
			seed:    seed,
			kp:      "0x0000000000000000000000000000000000000000000000000000000000000000",
			waitErr: errors.New("generated public key does not equal provide public key"),
		},

		"unknown key type": {
			ksType:  "someothertype",
			seed:    seed,
			kp:      grandKp,
			waitErr: errors.New("cannot decode key: invalid key type"),
		},
	}

	for tname, tt := range testcases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			var expectedKp crypto.Keypair
			var pubkey string

			if kp, ok := tt.kp.(crypto.Keypair); ok {
				expectedKp = kp
				pubkey = kp.Public().Hex()
			} else {
				pubkey = tt.kp.(string)
			}

			req := &KeyInsertRequest{tt.ksType, tt.seed, pubkey}
			res := new(KeyInsertResponse)
			err = auth.InsertKey(nil, req, res)

			if tt.waitErr != nil {
				require.EqualError(t, tt.waitErr, err.Error())
				return
			}

			require.Nil(t, err)

			ks, err := intctrl.keystore.GetKeystore([]byte(tt.ksType))
			require.NoError(t, err)

			foundKp := ks.GetKeypairFromAddress(expectedKp.Public().Address())
			require.NotNil(t, foundKp)
			require.Equal(t, expectedKp, foundKp)
		})
	}

}

func TestAuthorModule_HasKey_Integration(t *testing.T) {
	tmppath := t.TempDir()

	intctrl := setupStateAndRuntime(t, tmppath, useInstanceFromGenesis)

	ks := keystore.NewGlobalKeystore()

	kr, err := keystore.NewSr25519Keyring()
	require.Nil(t, err)

	ks.Babe.Insert(kr.Alice())

	intctrl.keystore = ks

	auth := newAuthorModule(t, intctrl)

	testcases := map[string]struct {
		pub, keytype string
		hasKey       bool
		waitErr      error
	}{
		"key exists and should return true": {
			pub:     kr.Alice().Public().Hex(),
			keytype: "babe",
			hasKey:  true,
		},

		"key does not exists and should return false": {
			pub:     kr.Bob().Public().Hex(),
			keytype: "babe",
			hasKey:  false,
		},

		"invalid key should return error": {
			pub:     "0xaa11",
			keytype: "babe",
			hasKey:  false,
			waitErr: errors.New("cannot create public key: input is not 32 bytes"),
		},
		"invalid key type should return error": {
			pub:     kr.Alice().Public().Hex(),
			keytype: "xxxx",
			hasKey:  false,
			waitErr: keystore.ErrInvalidKeystoreName,
		},
	}

	for tname, tt := range testcases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			var res bool
			req := []string{tt.pub, tt.keytype}

			err = auth.HasKey(nil, &req, &res)

			if tt.waitErr != nil {
				require.EqualError(t, tt.waitErr, err.Error())
				require.False(t, res)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.hasKey, res)
		})
	}
}

func TestAuthorModule_HasSessionKeys_Integration(t *testing.T) {
	tmpdir := t.TempDir()

	intCtrl := setupStateAndRuntime(t, tmpdir, useInstanceFromGenesis)
	intCtrl.stateSrv.Transaction = state.NewTransactionState()
	intCtrl.keystore = keystore.NewGlobalKeystore()

	auth := newAuthorModule(t, intCtrl)

	const granSeed = "0xf25586ceb64a043d887631fa08c2ed790ef7ae3c7f28de5172005f8b9469e529"
	const granPubK = "0x6b802349d948444d41397da09ec597fbd8ae8fdd3dfa153b2bb2bddcf020457c"

	const sr25519Seed = "0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a"
	const sr25519Pubk = "0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"

	insertSessionKeys := []struct {
		ktype      []string
		seed, pubk string
	}{
		{
			ktype: []string{"gran"},
			seed:  granSeed,
			pubk:  granPubK,
		},
		{
			ktype: []string{"babe", "imon", "audi"},
			seed:  sr25519Seed,
			pubk:  sr25519Pubk,
		},
	}

	for _, toInsert := range insertSessionKeys {
		for _, keytype := range toInsert.ktype {
			err := auth.InsertKey(nil, &KeyInsertRequest{
				Type:      keytype,
				Seed:      toInsert.seed,
				PublicKey: toInsert.pubk,
			}, nil)
			require.NoError(t, err)
		}
	}

	testcases := map[string]struct {
		pubSessionKeys string
		expect         bool
		waitErr        error
	}{
		"public keys are in the right order, should return true": {
			pubSessionKeys: "0x6b802349d948444d41397da09ec597fbd8ae8fdd3dfa153b2bb2bddcf020457c" + // gran
				"d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d" + // babe
				"d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d" + // imon
				"d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d", // audi
			expect: true,
		},
		"unknown public keys in the right order, should return false": {
			pubSessionKeys: "0x740550da19ef14023ea3e903545a6700160a55be2e4b733b577c91b053e38b8d" + // gran
				"de6fa0da51c52cc117d77aeb329595b15070db444e7ed4c4adec714b291c1845" + // babe
				"de6fa0da51c52cc117d77aeb329595b15070db444e7ed4c4adec714b291c1845" + // imon
				"de6fa0da51c52cc117d77aeb329595b15070db444e7ed4c4adec714b291c1845", // audi
			expect: false,
		},
		"public keys are not in the right order, should return false": {
			pubSessionKeys: "0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d" + // gran
				"6b802349d948444d41397da09ec597fbd8ae8fdd3dfa153b2bb2bddcf020457c" + // babe
				"d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d" + // imon
				"d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d", // audi
			expect: false,
		},
		"incomplete keys": {
			pubSessionKeys: "0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d" + // gran
				"6b802349d948444d41397da09ec597fbd8ae8fdd3dfa153b2bb2bddcf020457c", // babe
			expect: false,
		},
		"empty public keys": {
			pubSessionKeys: "", // babe
			expect:         false,
			waitErr:        errors.New("invalid string"),
		},
	}

	for tname, tt := range testcases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			req := HasSessionKeyRequest{
				PublicKeys: tt.pubSessionKeys,
			}

			var res HasSessionKeyResponse

			err := auth.HasSessionKeys(nil, &req, &res)

			if tt.waitErr != nil {
				require.EqualError(t, tt.waitErr, err.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expect, bool(res))
		})
	}
}

type integrationTestController struct {
	genesis       *genesis.Genesis
	genesisTrie   *trie.Trie
	genesisHeader *types.Header
	runtime       runtime.Instance
	stateSrv      *state.Service
	network       core.Network
	storageState  core.StorageState
	keystore      *keystore.GlobalKeystore
}

func setupStateAndRuntime(t *testing.T, basepath string, useInstance useRuntimeInstace) *integrationTestController {
	t.Helper()

	state2test := state.NewService(state.Config{LogLevel: log.DoNotChange, Path: basepath})
	state2test.UseMemDB()
	state2test.Transaction = state.NewTransactionState()

	gen, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)

	err := state2test.Initialise(gen, genesisHeader, genTrie)
	require.NoError(t, err)

	err = state2test.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		state2test.Stop()
	})

	rtStorage, err := state2test.Storage.TrieState(nil)
	require.NoError(t, err)

	cfg := &wasmer.Config{}
	cfg.Storage = rtStorage
	cfg.LogLvl = 4
	nodeStorage := runtime.NodeStorage{}
	nodeStorage.BaseDB = runtime.NewInMemoryDB(t)
	cfg.NodeStorage = nodeStorage

	rt, err := useInstance(t, cfg)
	require.NoError(t, err)

	genesisHash := genesisHeader.Hash()
	state2test.Block.StoreRuntime(genesisHash, rt)

	return &integrationTestController{
		genesis:       gen,
		genesisTrie:   genTrie,
		genesisHeader: genesisHeader,
		stateSrv:      state2test,
		storageState:  state2test.Storage,
		runtime:       rt,
	}
}

func setupStateAndPopulateTrieState(t *testing.T, basepath string,
	useInstance useRuntimeInstace) *integrationTestController {
	t.Helper()

	state2test := state.NewService(state.Config{LogLevel: log.DoNotChange, Path: basepath})
	state2test.UseMemDB()
	state2test.Transaction = state.NewTransactionState()

	gen, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)

	err := state2test.Initialise(gen, genesisHeader, genTrie)
	require.NoError(t, err)

	err = state2test.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		state2test.Stop()
	})

	rtStorage, err := state2test.Storage.TrieState(nil)
	require.NoError(t, err)

	cfg := &wasmer.Config{}
	cfg.Role = 0
	cfg.LogLvl = log.Warn
	cfg.Storage = rtStorage
	cfg.Keystore = keystore.NewGlobalKeystore()
	cfg.Network = new(runtime.TestRuntimeNetwork)
	cfg.NodeStorage = runtime.NodeStorage{
		LocalStorage:      runtime.NewInMemoryDB(t),
		PersistentStorage: runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
		BaseDB:            runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
	}

	rt, err := useInstance(t, cfg)
	require.NoError(t, err)

	genesisHash := genesisHeader.Hash()
	state2test.Block.StoreRuntime(genesisHash, rt)

	b := runtime.InitializeRuntimeToTest(t, rt, genesisHash)

	err = state2test.Block.AddBlock(b)
	require.NoError(t, err)

	err = state2test.Storage.StoreTrie(rtStorage, &b.Header)
	require.NoError(t, err)

	state2test.Block.StoreRuntime(b.Header.Hash(), rt)

	return &integrationTestController{
		genesis:       gen,
		genesisTrie:   genTrie,
		genesisHeader: genesisHeader,
		stateSrv:      state2test,
		storageState:  state2test.Storage,
		runtime:       rt,
	}
}

func newAuthorModule(t *testing.T, intCtrl *integrationTestController) *AuthorModule {
	t.Helper()

	cfg := &core.Config{
		TransactionState: intCtrl.stateSrv.Transaction,
		BlockState:       intCtrl.stateSrv.Block,
		StorageState:     intCtrl.storageState,
		Network:          intCtrl.network,
		Keystore:         intCtrl.keystore,
	}

	uselessCh := make(chan *types.Block, 256)
	core2test := core.NewService2Test(context.TODO(), t, cfg, uselessCh)
	return NewAuthorModule(log.New(log.SetLevel(log.Debug)), core2test, intCtrl.stateSrv.Transaction)
}
