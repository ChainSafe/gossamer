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
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

const TransferTest = "0xb9018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0112acc21fe3445996c2c5810c3321dd1324b7ba5eb327fb1b148c289fc7f77e44045ebb72189222d126db2bd45b78747e32a33f464df289487c6eb3c2ba022f87d6000400000110736f6d65"

// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
const TestExtrinsic = "0x410284ffd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01f8efbe48487e57a22abf7e3acd491b7f3528a33a111b1298601554863d27eb129eaa4e718e1365414ff3d028b62bebc651194c6b5001e5c2839b982757e08a8c0000000600ff8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a480b00c465f14670"

var testExt = common.MustHexToBytes(TestExtrinsic)

// invalid transaction (above tx, with last byte changed)
var testInvalidExt = []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 143} //nolint:lll

func TestMain(m *testing.M) {
	_, err := runtime.GenerateRuntimeWasmFile()
	if err != nil {
		log.Errorf("failed to generate runtime wasm file: %s", err)
		os.Exit(1)
	}

	// Start all tests
	code := m.Run()

	//runtime.RemoveFiles(wasmFilePaths)
	os.Exit(code)
}

func TestAuthorModule_Pending_Integration(t *testing.T) {
	t.Parallel()

	tmpdir := t.TempDir()

	state2test := state.NewService2Test(t, &state.Config{LogLevel: log.DoNotChange, Path: tmpdir})
	state2test.Transaction = state.NewTransactionState()

	auth := setupAuhtorModule2Test(t, &integrationTestController{stateSrv: state2test})

	res := new(PendingExtrinsicsResponse)
	err := auth.PendingExtrinsics(nil, nil, res)

	require.NoError(t, err)
	require.Equal(t, PendingExtrinsicsResponse([]string{}), *res)

	vtx := &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic(testExt),
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

	intCtrl := setupStateAndPopulateTrieState(t, tmpbasepath)
	intCtrl.stateSrv.Transaction = state.NewTransactionState()

	genesisHash := intCtrl.genesisHeader.Hash()
	blockHash := intCtrl.stateSrv.Block.BestBlockHash()

	// creating an extrisinc to the System.remark call using a sample argument
	extHex := runtime.CreateTestExtrinsic(t,
		intCtrl.runtime, genesisHash, blockHash, 0, "System.remark", []byte{0xab, 0xcd})

	extBytes := common.MustHexToBytes(extHex)

	ctrl := gomock.NewController(t)
	net2test := coremocks.NewMockNetwork(ctrl)
	net2test.EXPECT().GossipMessage(&network.TransactionMessage{Extrinsics: []types.Extrinsic{extBytes}})
	intCtrl.network = net2test

	// setup auth module
	auth := setupAuhtorModule2Test(t, intCtrl)

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

	intCtrl := setupStateAndRuntime(t, tmpbasepath)
	intCtrl.stateSrv.Transaction = state.NewTransactionState()

	genesisHash := intCtrl.genesisHeader.Hash()

	// creating an extrisinc to the System.remark call using a sample argument
	extHex := runtime.CreateTestExtrinsic(t,
		intCtrl.runtime, genesisHash, genesisHash, 0, "System.remark", []byte{})

	ctrl := gomock.NewController(t)
	net2test := coremocks.NewMockNetwork(ctrl)
	net2test.EXPECT().GossipMessage(nil).MaxTimes(0)

	intCtrl.network = net2test

	// setup auth module
	auth := setupAuhtorModule2Test(t, intCtrl)

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
	intctrl := setupStateAndRuntime(t, tmppath)
	auth := setupAuhtorModule2Test(t, intctrl)

	// create and submit extrinsic
	ext := Extrinsic{fmt.Sprintf("%x", "1")}

	res := new(ExtrinsicHashResponse)
	err := auth.SubmitExtrinsic(nil, &ext, res)
	require.EqualError(t, err, "could not byteify non 0x prefixed string")
}

func TestAuthorModule_SubmitExtrinsic_AlreadyInPool(t *testing.T) {
	t.Parallel()

	tmpbasepath := t.TempDir()
	intCtrl := setupStateAndRuntime(t, tmpbasepath)
	intCtrl.stateSrv.Transaction = state.NewTransactionState()

	genesisHash := intCtrl.genesisHeader.Hash()

	// creating an extrisinc to the System.remark call using a sample argument
	extHex := runtime.CreateTestExtrinsic(t,
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
	auth := setupAuhtorModule2Test(t, intCtrl)

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

	intctrl := setupStateAndRuntime(t, tmppath)
	intctrl.keystore = keystore.NewGlobalKeystore()

	auth := setupAuhtorModule2Test(t, intctrl)

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

		"unkown key type": {
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

	intctrl := setupStateAndRuntime(t, tmppath)

	ks := keystore.NewGlobalKeystore()

	kr, err := keystore.NewSr25519Keyring()
	require.Nil(t, err)

	ks.Babe.Insert(kr.Alice())

	intctrl.keystore = ks

	auth := setupAuhtorModule2Test(t, intctrl)

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

func setupStateAndRuntime(t *testing.T, basepath string) *integrationTestController {
	t.Helper()

	config := &state.Config{
		Path:     basepath,
		LogLevel: log.Info,
	}

	state2test := state.NewService2Test(t, config)

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

	rt, err := wasmer.NewRuntimeFromGenesis(cfg)
	require.NoError(t, err)

	genesisHash := genesisHeader.Hash()
	state2test.Block.StoreRuntime(genesisHash, rt)

	return &integrationTestController{
		genesis:       gen,
		genesisTrie:   genTrie,
		genesisHeader: genesisHeader,
		stateSrv:      state2test,
		runtime:       rt,
	}
}

func setupStateAndPopulateTrieState(t *testing.T, basepath string) *integrationTestController {
	t.Helper()

	config := &state.Config{
		Path:     basepath,
		LogLevel: log.Info,
	}

	state2test := state.NewService2Test(t, config)

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

	rt, err := wasmer.NewRuntimeFromGenesis(cfg)
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

func setupAuhtorModule2Test(t *testing.T, intCtrl *integrationTestController) *AuthorModule {
	t.Helper()

	cfg := &core.Config{
		TransactionState: intCtrl.stateSrv.Transaction,
		BlockState:       intCtrl.stateSrv.Block,
		StorageState:     intCtrl.storageState,
		Network:          intCtrl.network,
		Keystore:         intCtrl.keystore,
	}

	core2test := core.NewService2Test(t, context.TODO(), cfg, nil)
	return NewAuthorModule(log.New(log.SetLevel(log.Debug)), core2test, intCtrl.stateSrv.Transaction)
}
