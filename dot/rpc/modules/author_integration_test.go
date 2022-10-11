// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package modules

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core"
	network "github.com/ChainSafe/gossamer/dot/network"
	peerset "github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/state"
	telemetry "github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	cscale "github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"

	"github.com/golang/mock/gomock"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

type useRuntimeInstance func(*testing.T, *storage.TrieState) Runtime

// useInstanceFromGenesis creates a new runtime instance given a trie state
func useInstanceFromGenesis(t *testing.T, rtStorage *storage.TrieState) (instance Runtime) {
	t.Helper()

	cfg := wasmer.Config{
		Storage: rtStorage,
		LogLvl:  log.Warn,
		NodeStorage: runtime.NodeStorage{
			BaseDB: runtime.NewInMemoryDB(t),
		},
	}

	runtimeInstance, err := wasmer.NewRuntimeFromGenesis(cfg)
	require.NoError(t, err)

	return runtimeInstance
}

func useInstanceFromRuntimeV0910(t *testing.T, rtStorage *storage.TrieState) (instance Runtime) {
	testRuntimeFilePath, err := runtime.GetRuntime(context.Background(), runtime.POLKADOT_RUNTIME_v0910)
	require.NoError(t, err)
	bytes, err := os.ReadFile(testRuntimeFilePath)
	require.NoError(t, err)

	rtStorage.Put(common.CodeKey, bytes)

	cfg := wasmer.Config{
		Role:     0,
		LogLvl:   log.Critical,
		Storage:  rtStorage,
		Keystore: keystore.NewGlobalKeystore(),
		NodeStorage: runtime.NodeStorage{
			LocalStorage:      runtime.NewInMemoryDB(t),
			PersistentStorage: runtime.NewInMemoryDB(t),
			BaseDB:            runtime.NewInMemoryDB(t),
		},
	}

	runtimeInstance, err := wasmer.NewInstanceFromTrie(rtStorage.Trie(), cfg)
	require.NoError(t, err)

	return runtimeInstance
}

func createExtrinsic(t *testing.T, rt runtime.Instance, genHash common.Hash, nonce uint64) types.Extrinsic {
	t.Helper()
	rawMeta, err := rt.Metadata()
	require.NoError(t, err)

	var decoded []byte
	err = scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}

	err = codec.Decode(decoded, meta)
	require.NoError(t, err)

	runtimeVersion := rt.Version()

	metaCall, err := ctypes.NewCall(meta, "System.remark", []byte{0xab, 0xcd})
	require.NoError(t, err)

	extrinsic := ctypes.NewExtrinsic(metaCall)
	options := ctypes.SignatureOptions{
		BlockHash:          ctypes.Hash(genHash),
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: false},
		GenesisHash:        ctypes.Hash(genHash),
		Nonce:              ctypes.NewUCompactFromUInt(nonce),
		SpecVersion:        ctypes.U32(runtimeVersion.SpecVersion),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(runtimeVersion.TransactionVersion),
	}

	// Sign the transaction using Alice's key
	err = extrinsic.Sign(signature.TestKeyringPairAlice, options)
	require.NoError(t, err)

	extEnc := bytes.NewBuffer(nil)
	encoder := cscale.NewEncoder(extEnc)
	err = extrinsic.Encode(*encoder)
	require.NoError(t, err)

	return extEnc.Bytes()
}

func TestAuthorModule_Pending_Integration(t *testing.T) {
	t.Parallel()
	integrationTestController := setupStateAndRuntime(t, t.TempDir(), nil)

	auth := newAuthorModule(t, integrationTestController)
	res := new(PendingExtrinsicsResponse)
	err := auth.PendingExtrinsics(nil, nil, res)

	require.NoError(t, err)
	require.Equal(t, PendingExtrinsicsResponse([]string{}), *res)

	vtx := &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic([]byte{0x01, 0x02}),
		Validity:  new(transaction.Validity),
	}

	_, err = integrationTestController.stateSrv.Transaction.Push(vtx)
	require.NoError(t, err)

	err = auth.PendingExtrinsics(nil, nil, res)
	require.NoError(t, err)

	expected := common.BytesToHex(vtx.Extrinsic)
	require.Equal(t, PendingExtrinsicsResponse([]string{expected}), *res)
}

func TestAuthorModule_SubmitExtrinsic_Integration(t *testing.T) {
	t.Parallel()
	integrationTestController := setupStateAndPopulateTrieState(t, t.TempDir(), useInstanceFromGenesis)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().
		SendMessage(
			telemetry.NewTxpoolImport(0, 1),
		)

	integrationTestController.stateSrv.Transaction = state.NewTransactionState(telemetryMock)

	genesisHash := integrationTestController.genesisHeader.Hash()

	// creating an extrisinc to the System.remark call using a sample argument
	extHex := runtime.NewTestExtrinsic(t,
		integrationTestController.runtime, genesisHash, genesisHash, 0, "System.remark", []byte{0xab, 0xcd})

	extBytes := common.MustHexToBytes(extHex)

	net2test := NewMockNetwork(ctrl)
	net2test.EXPECT().GossipMessage(&network.TransactionMessage{Extrinsics: []types.Extrinsic{extBytes}})
	integrationTestController.network = net2test

	// setup auth module
	auth := newAuthorModule(t, integrationTestController)

	res := new(ExtrinsicHashResponse)
	err := auth.SubmitExtrinsic(nil, &Extrinsic{extHex}, res)
	require.NoError(t, err)

	expectedExtrinsic := types.NewExtrinsic(extBytes)
	expected := &transaction.ValidTransaction{
		Extrinsic: expectedExtrinsic,
		Validity: &transaction.Validity{
			Priority: 39325240425794630,
			Requires: nil,
			Provides: [][]byte{
				common.MustHexToBytes("0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d00000000"),
			},
			Longevity: 18446744073709551614,
			Propagate: true,
		},
	}

	expectedHash := ExtrinsicHashResponse(expectedExtrinsic.Hash().String())
	txOnPool := integrationTestController.stateSrv.Transaction.PendingInPool()

	// compare results
	require.Len(t, txOnPool, 1)
	require.Equal(t, expected, txOnPool[0])
	require.Equal(t, expectedHash, *res)
}

func TestAuthorModule_SubmitExtrinsic_invalid(t *testing.T) {
	t.Parallel()
	integrationTestController := setupStateAndRuntime(t, t.TempDir(), useInstanceFromGenesis)

	genesisHash := integrationTestController.genesisHeader.Hash()

	// creating an extrisinc to the System.remark call using a sample argument
	extHex := runtime.NewTestExtrinsic(t,
		integrationTestController.runtime, genesisHash, genesisHash, 0, "System.remark", []byte{})

	ctrl := gomock.NewController(t)
	net2test := NewMockNetwork(ctrl)
	net2test.EXPECT().GossipMessage(nil).MaxTimes(0)

	integrationTestController.network = net2test

	// setup auth module
	auth := newAuthorModule(t, integrationTestController)

	res := new(ExtrinsicHashResponse)
	err := auth.SubmitExtrinsic(nil, &Extrinsic{extHex}, res)
	require.EqualError(t, err, "ancient birth block")

	txOnPool := integrationTestController.stateSrv.Transaction.PendingInPool()
	require.Len(t, txOnPool, 0)
}

func TestAuthorModule_SubmitExtrinsic_invalid_input(t *testing.T) {
	t.Parallel()

	// setup service
	// setup auth module
	integrationTestController := setupStateAndRuntime(t, t.TempDir(), useInstanceFromGenesis)
	auth := newAuthorModule(t, integrationTestController)

	res := new(ExtrinsicHashResponse)
	err := auth.SubmitExtrinsic(nil, &Extrinsic{fmt.Sprintf("%x", "1")}, res)
	require.EqualError(t, err, "could not byteify non 0x prefixed string: 31")
}

func TestAuthorModule_SubmitExtrinsic_AlreadyInPool(t *testing.T) {
	t.Parallel()
	integrationTestController := setupStateAndRuntime(t, t.TempDir(), useInstanceFromGenesis)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().
		SendMessage(
			telemetry.NewTxpoolImport(0, 1),
		)

	integrationTestController.stateSrv.Transaction = state.NewTransactionState(telemetryMock)

	genesisHash := integrationTestController.genesisHeader.Hash()

	// creating an extrisinc to the System.remark call using a sample argument
	extHex := runtime.NewTestExtrinsic(t,
		integrationTestController.runtime, genesisHash, genesisHash, 0, "System.remark", []byte{})
	extBytes := common.MustHexToBytes(extHex)

	integrationTestController.network = NewMockNetwork(nil)

	// setup auth module
	auth := newAuthorModule(t, integrationTestController)

	res := new(ExtrinsicHashResponse)

	expectedExtrinsic := types.NewExtrinsic(extBytes)
	expected := &transaction.ValidTransaction{
		Extrinsic: expectedExtrinsic,
		Validity: &transaction.Validity{
			Priority: 39325240425794630,
			Requires: nil,
			Provides: [][]byte{
				common.MustHexToBytes("0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d00000000"),
			},
			Longevity: 18446744073709551614,
			Propagate: true,
		},
	}

	integrationTestController.stateSrv.Transaction.AddToPool(expected)

	err := auth.SubmitExtrinsic(nil, &Extrinsic{extHex}, res)
	require.NoError(t, err)
}

func TestAuthorModule_InsertKey_Integration(t *testing.T) {
	t.Parallel()
	integrationTestController := setupStateAndRuntime(t, t.TempDir(), useInstanceFromGenesis)
	auth := newAuthorModule(t, integrationTestController)

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
			waitErr: ErrProvidedKeyDoesNotMatch,
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

			type keyPair interface {
				Public() crypto.PublicKey
			}

			var expectedKp keyPair
			var pubkey string

			if kp, ok := tt.kp.(keyPair); ok {
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

			require.NoError(t, err)

			ks, err := integrationTestController.keystore.GetKeystore([]byte(tt.ksType))
			require.NoError(t, err)

			foundKp := ks.GetKeypairFromAddress(expectedKp.Public().Address())
			require.NotNil(t, foundKp)
			require.Equal(t, expectedKp, foundKp)
		})
	}

}

func TestAuthorModule_HasKey_Integration(t *testing.T) {
	t.Parallel()
	integrationTestController := setupStateAndRuntime(t, t.TempDir(), useInstanceFromGenesis)

	ks := keystore.NewGlobalKeystore()

	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	ks.Babe.Insert(kr.Alice())

	integrationTestController.keystore = ks

	auth := newAuthorModule(t, integrationTestController)

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
	t.Parallel()

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
			waitErr:        errors.New("could not byteify non 0x prefixed string: "),
		},
	}

	for tname, tt := range testcases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			integrationTestController := setupStateAndRuntime(t, t.TempDir(), useInstanceFromGenesis)
			auth := newAuthorModule(t, integrationTestController)
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

func TestAuthorModule_SubmitExtrinsic_WithVersion_V0910(t *testing.T) {
	t.Parallel()
	integrationTestController := setupStateAndPopulateTrieState(t, t.TempDir(), useInstanceFromRuntimeV0910)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().
		SendMessage(
			telemetry.NewTxpoolImport(0, 1),
		)

	integrationTestController.stateSrv.Transaction = state.NewTransactionState(telemetryMock)

	genesisHash := integrationTestController.genesisHeader.Hash()
	extrinsic := createExtrinsic(t, integrationTestController.runtime, genesisHash, 0)
	extHex := common.BytesToHex(extrinsic)

	net2test := NewMockNetwork(ctrl)
	net2test.EXPECT().GossipMessage(&network.TransactionMessage{Extrinsics: []types.Extrinsic{extrinsic}})
	integrationTestController.network = net2test

	// setup auth module
	auth := newAuthorModule(t, integrationTestController)

	res := new(ExtrinsicHashResponse)
	err := auth.SubmitExtrinsic(nil, &Extrinsic{extHex}, res)
	require.NoError(t, err)

	expectedExtrinsic := types.NewExtrinsic(extrinsic)
	expected := &transaction.ValidTransaction{
		Extrinsic: expectedExtrinsic,
		Validity: &transaction.Validity{
			Priority: 4295664014726,
			Provides: [][]byte{
				common.MustHexToBytes("0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d00000000"),
			},
			Longevity: 18446744073709551613,
			Propagate: true,
		},
	}

	expectedHash := ExtrinsicHashResponse(expectedExtrinsic.Hash().String())
	txOnPool := integrationTestController.stateSrv.Transaction.PendingInPool()

	// compare results
	require.Len(t, txOnPool, 1)
	require.Equal(t, expected, txOnPool[0])
	require.Equal(t, expectedHash, *res)
}

type coreNetwork interface {
	GossipMessage(network.NotificationsMessage)
	IsSynced() bool
	ReportPeer(change peerset.ReputationChange, p peer.ID)
}

type coreStorageState interface {
	TrieState(root *common.Hash) (*storage.TrieState, error)
	StoreTrie(*storage.TrieState, *types.Header) error
	GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error)
	GenerateTrieProof(stateRoot common.Hash, keys [][]byte) ([][]byte, error)
	sync.Locker
}

type integrationTestController struct {
	genesis       *genesis.Genesis
	genesisTrie   *trie.Trie
	genesisHeader *types.Header
	runtime       Runtime
	stateSrv      *state.Service
	network       coreNetwork
	storageState  coreStorageState
	keystore      *keystore.GlobalKeystore
}

func setupStateAndRuntime(t *testing.T, basepath string, useInstance useRuntimeInstance) *integrationTestController {
	t.Helper()

	gen, genesisTrie, genesisHeader := newTestGenesisWithTrieAndHeader(t)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().
		SendMessage(
			telemetry.NewNotifyFinalized(
				genesisHeader.Hash(),
				"0",
			),
		)

	state2test := state.NewService(state.Config{
		LogLevel:  log.DoNotChange,
		Path:      basepath,
		Telemetry: telemetryMock,
	})
	state2test.UseMemDB()

	state2test.Transaction = state.NewTransactionState(telemetryMock)
	err := state2test.Initialise(&gen, &genesisHeader, &genesisTrie)
	require.NoError(t, err)

	err = state2test.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		state2test.Stop()
	})

	ks := keystore.NewGlobalKeystore()
	net2test := NewMockNetwork(nil)
	integrationTestController := &integrationTestController{
		genesis:       &gen,
		genesisTrie:   &genesisTrie,
		genesisHeader: &genesisHeader,
		stateSrv:      state2test,
		storageState:  state2test.Storage,
		keystore:      ks,
		network:       net2test,
	}

	if useInstance != nil {
		rtStorage, err := state2test.Storage.TrieState(nil)
		require.NoError(t, err)

		rt := useInstance(t, rtStorage)

		genesisHash := genesisHeader.Hash()
		state2test.Block.StoreRuntime(genesisHash, rt)
		integrationTestController.runtime = rt
	}

	return integrationTestController
}

func setupStateAndPopulateTrieState(t *testing.T, basepath string,
	useInstance useRuntimeInstance) *integrationTestController {
	t.Helper()

	gen, genesisTrie, genesisHeader := newTestGenesisWithTrieAndHeader(t)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().
		SendMessage(
			telemetry.NewNotifyFinalized(
				genesisHeader.Hash(),
				"0",
			),
		)

	state2test := state.NewService(state.Config{
		LogLevel:  log.DoNotChange,
		Path:      basepath,
		Telemetry: telemetryMock,
	})
	state2test.UseMemDB()

	state2test.Transaction = state.NewTransactionState(telemetryMock)

	err := state2test.Initialise(&gen, &genesisHeader, &genesisTrie)
	require.NoError(t, err)

	err = state2test.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		state2test.Stop()
	})

	net2test := NewMockNetwork(nil)
	ks := keystore.NewGlobalKeystore()
	integrationTestController := &integrationTestController{
		genesis:       &gen,
		genesisTrie:   &genesisTrie,
		genesisHeader: &genesisHeader,
		stateSrv:      state2test,
		storageState:  state2test.Storage,
		keystore:      ks,
		network:       net2test,
	}

	if useInstance != nil {
		rtStorage, err := state2test.Storage.TrieState(nil)
		require.NoError(t, err)

		rt := useInstance(t, rtStorage)

		integrationTestController.runtime = rt

		genesisHash := genesisHeader.Hash()
		state2test.Block.StoreRuntime(genesisHash, rt)

		b := runtime.InitializeRuntimeToTest(t, rt, genesisHash)

		err = state2test.Block.AddBlock(b)
		require.NoError(t, err)

		err = state2test.Storage.StoreTrie(rtStorage, &b.Header)
		require.NoError(t, err)

		state2test.Block.StoreRuntime(b.Header.Hash(), rt)
	}

	return integrationTestController
}

func newAuthorModule(t *testing.T, integrationTestController *integrationTestController) *AuthorModule {
	t.Helper()

	codeSubstitutedStateMock := NewMockCodeSubstitutedState(nil)

	cfg := &core.Config{
		TransactionState:     integrationTestController.stateSrv.Transaction,
		BlockState:           integrationTestController.stateSrv.Block,
		StorageState:         integrationTestController.storageState,
		Network:              integrationTestController.network,
		Keystore:             integrationTestController.keystore,
		CodeSubstitutedState: codeSubstitutedStateMock,
	}

	core2test, err := core.NewService(cfg)
	require.NoError(t, err)
	return NewAuthorModule(log.New(log.SetLevel(log.Debug)), core2test, integrationTestController.stateSrv.Transaction)
}
