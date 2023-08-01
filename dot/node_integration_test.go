// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package dot

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/chain/westend"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot/core"
	digest "github.com/ChainSafe/gossamer/dot/digest"
	network "github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	dotsync "github.com/ChainSafe/gossamer/dot/sync"
	system "github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/parachain"
	"github.com/ChainSafe/gossamer/lib/runtime"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/trie"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNode(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockTelemetryClient := NewMockTelemetry(ctrl)
	mockTelemetryClient.EXPECT().SendMessage(gomock.Any())

	basepath := t.TempDir()
	initConfig := westend.DefaultConfig()
	genFile := NewTestGenesisRawFile(t, initConfig)

	initConfig.Name = "TestNode"
	initConfig.BasePath = basepath
	initConfig.ChainSpec = genFile
	initConfig.Account.Key = "alice"
	initConfig.Core.Role = common.FullNodeRole
	initConfig.Core.WasmInterpreter = wazero_runtime.Name
	initConfig.Log.Digest = "critical"

	networkConfig := &network.Config{
		BasePath:    basepath,
		NoBootstrap: true,
		NoMDNS:      true,
	}
	setConfigTestDefaults(t, networkConfig)

	testNetworkService, err := network.NewService(networkConfig)
	require.NoError(t, err)

	logLevel, err := log.ParseLevel(initConfig.Log.State)
	require.NoError(t, err)
	stateConfig := state.Config{
		Path:     initConfig.BasePath,
		LogLevel: logLevel,
	}

	systemInfo := &types.SystemInfo{
		SystemName:    initConfig.System.SystemName,
		SystemVersion: initConfig.System.SystemVersion,
	}

	ks, err := initKeystore(t, initConfig)
	assert.NoError(t, err)

	mockServiceRegistry := NewMockServiceRegisterer(ctrl)
	mockServiceRegistry.EXPECT().RegisterService(gomock.Any()).Times(9)

	m := NewMocknodeBuilderIface(ctrl)
	m.EXPECT().isNodeInitialised(initConfig.BasePath).Return(true, nil)
	m.EXPECT().createStateService(initConfig).DoAndReturn(func(config *cfg.Config) (*state.Service, error) {
		stateSrvc := state.NewService(stateConfig)
		// create genesis from configuration file
		gen, err := genesis.NewGenesisFromJSONRaw(config.ChainSpec)
		if err != nil {
			return nil, fmt.Errorf("failed to load genesis from file: %w", err)
		}
		// create trie from genesis
		trie, err := runtime.NewTrieFromGenesis(*gen)
		if err != nil {
			return nil, fmt.Errorf("failed to create trie from genesis: %w", err)
		}
		// create genesis block from trie
		header, err := trie.GenesisBlock()
		if err != nil {
			return nil, fmt.Errorf("failed to create genesis block from trie: %w", err)
		}
		stateSrvc.Telemetry = mockTelemetryClient
		err = stateSrvc.Initialise(gen, &header, &trie)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise state service: %s", err)
		}

		err = stateSrvc.SetupBase()
		if err != nil {
			return nil, fmt.Errorf("cannot setup base: %w", err)
		}
		return stateSrvc, nil
	})

	phs, err := parachain.NewService(testNetworkService, "random_fork_id", common.Hash{})
	require.NoError(t, err)

	m.EXPECT().createRuntimeStorage(gomock.AssignableToTypeOf(&state.Service{})).Return(&runtime.
		NodeStorage{}, nil)
	m.EXPECT().loadRuntime(initConfig, &runtime.NodeStorage{}, gomock.AssignableToTypeOf(&state.Service{}),
		ks, gomock.AssignableToTypeOf(&network.Service{})).Return(nil)
	m.EXPECT().createBlockVerifier(gomock.AssignableToTypeOf(&state.Service{})).
		Return(&babe.VerificationManager{})
	m.EXPECT().createDigestHandler(gomock.AssignableToTypeOf(&state.Service{})).
		Return(&digest.Handler{}, nil)
	m.EXPECT().createCoreService(initConfig, ks, gomock.AssignableToTypeOf(&state.Service{}),
		gomock.AssignableToTypeOf(&network.Service{}), &digest.Handler{}).
		Return(&core.Service{}, nil)
	m.EXPECT().createGRANDPAService(initConfig, gomock.AssignableToTypeOf(&state.Service{}),
		ks.Gran, gomock.AssignableToTypeOf(&network.Service{}),
		gomock.AssignableToTypeOf(&telemetry.Mailer{})).
		Return(&grandpa.Service{}, nil)
	m.EXPECT().newSyncService(initConfig, gomock.AssignableToTypeOf(&state.Service{}), &grandpa.Service{},
		&babe.VerificationManager{}, &core.Service{}, gomock.AssignableToTypeOf(&network.Service{}),
		gomock.AssignableToTypeOf(&telemetry.Mailer{})).
		Return(&dotsync.Service{}, nil)
	m.EXPECT().createBABEService(initConfig, gomock.AssignableToTypeOf(&state.Service{}), ks.Babe,
		&core.Service{}, gomock.AssignableToTypeOf(&telemetry.Mailer{})).
		Return(&babe.Service{}, nil)
	m.EXPECT().createSystemService(systemInfo, gomock.AssignableToTypeOf(&state.Service{})).
		DoAndReturn(func(cfg *types.SystemInfo, stateSrvc *state.Service) (*system.Service, error) {
			gd, err := stateSrvc.Base.LoadGenesisData()
			systemService := system.NewService(cfg, gd)
			return systemService, err
		})
	m.EXPECT().createNetworkService(initConfig, gomock.AssignableToTypeOf(&state.Service{}),
		gomock.AssignableToTypeOf(&telemetry.Mailer{})).Return(testNetworkService, nil)
	m.EXPECT().createParachainHostService(
		gomock.AssignableToTypeOf(&network.Service{}),
		gomock.AssignableToTypeOf("random_fork_id"),
		gomock.AssignableToTypeOf(common.Hash{}),
	).Return(phs, nil)

	got, err := newNode(initConfig, ks, m, mockServiceRegistry)
	assert.NoError(t, err)

	expected := &Node{
		Name: "TestNode",
	}

	assert.Equal(t, expected.Name, got.Name)
}

func Test_nodeBuilder_loadRuntime(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	type args struct {
		config *cfg.Config
		ns     *runtime.NodeStorage
		ks     *keystore.GlobalKeystore
		net    *network.Service
	}
	tests := []struct {
		name string
		args args
		err  error
	}{
		{
			name: "base_case",
			args: args{
				config: config,
				ns:     &runtime.NodeStorage{},
				ks:     nil,
				net:    nil,
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)
			no := nodeBuilder{}
			err := no.loadRuntime(tt.args.config, tt.args.ns, stateSrvc, tt.args.ks, tt.args.net)
			assert.ErrorIs(t, err, tt.err)
			blocks := stateSrvc.Block.GetNonFinalisedBlocks()
			for i := range blocks {
				hash := &blocks[i]
				code, err := stateSrvc.Storage.GetStorageByBlockHash(hash, []byte(":code"))
				require.NoError(t, err)
				require.NotEmpty(t, code)
			}
		})
	}
}

func TestInitNode_Integration(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	// confirm database was setup

	db, err := database.LoadDatabase(config.BasePath, false)
	require.NoError(t, err)
	require.NotNil(t, db)
	err = db.Close()
	require.NoError(t, err)
}

func TestInitNode_GenesisSpec(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)
	// confirm database was setup
	db, err := database.LoadDatabase(config.BasePath, false)
	require.NoError(t, err)
	require.NotNil(t, db)

	err = db.Close()
	require.NoError(t, err)
}

func TestNodeInitializedIntegration(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.ChainSpec = genFile

	result, err := IsNodeInitialised(config.BasePath)
	require.NoError(t, err)
	require.False(t, result)

	err = InitNode(config)
	require.NoError(t, err)

	result, err = IsNodeInitialised(config.BasePath)
	require.NoError(t, err)
	require.True(t, result)
}

func TestNewNodeIntegration(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ed25519keyRing, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	err = keystore.LoadKeystore("alice", ks.Gran, ed25519keyRing)
	require.NoError(t, err)
	sr25519keyRing, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	err = keystore.LoadKeystore("alice", ks.Babe, sr25519keyRing)
	require.NoError(t, err)

	config.Core.Role = common.FullNodeRole

	node, err := NewNode(config, ks)
	require.NoError(t, err)

	bp := node.ServiceRegistry.Get(&babe.Service{})
	require.IsType(t, &babe.Service{}, bp)
	fg := node.ServiceRegistry.Get(&grandpa.Service{})
	require.IsType(t, &grandpa.Service{}, fg)
}

func TestNewNode_Authority(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ed25519keyRing, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	err = keystore.LoadKeystore("alice", ks.Gran, ed25519keyRing)
	require.NoError(t, err)
	require.Equal(t, 1, ks.Gran.Size())

	sr25519keyRing, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	err = keystore.LoadKeystore("alice", ks.Babe, sr25519keyRing)
	require.NoError(t, err)
	require.Equal(t, 1, ks.Babe.Size())

	config.Core.Role = common.AuthorityRole

	node, err := NewNode(config, ks)
	require.NoError(t, err)

	bp := node.ServiceRegistry.Get(&babe.Service{})
	require.NotNil(t, bp)
	fg := node.ServiceRegistry.Get(&grandpa.Service{})
	require.NotNil(t, fg)
}

func TestStartStopNode(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.ChainSpec = genFile
	config.Core.GrandpaAuthority = false
	config.Core.BabeAuthority = false

	err := InitNode(config)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()

	ed25519keyRing, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	err = keystore.LoadKeystore("alice", ks.Gran, ed25519keyRing)
	require.NoError(t, err)

	sr25519keyRing, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	err = keystore.LoadKeystore("alice", ks.Babe, sr25519keyRing)
	require.NoError(t, err)

	config.Core.Role = common.FullNodeRole

	node, err := NewNode(config, ks)
	require.NoError(t, err)

	go func() {
		<-node.started
		node.Stop()
	}()
	err = node.Start()
	require.NoError(t, err)
}

func TestInitNode_LoadStorageRoot(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genPath := newTestGenesisAndRuntime(t)

	config.Core.Role = common.FullNodeRole
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	config.ChainSpec = genPath

	gen, err := genesis.NewGenesisFromJSONRaw(genPath)
	require.NoError(t, err)

	err = InitNode(config)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())
	sr25519Keyring, _ := keystore.NewSr25519Keyring()
	ks.Babe.Insert(sr25519Keyring.Alice())
	node, err := NewNode(config, ks)
	require.NoError(t, err)

	expected, err := trie.LoadFromMap(gen.GenesisFields().Raw["top"])
	require.NoError(t, err)

	expectedRoot, err := expected.Hash()
	require.NoError(t, err)

	coreServiceInterface := node.ServiceRegistry.Get(&core.Service{})

	coreSrvc, ok := coreServiceInterface.(*core.Service)
	require.True(t, ok, "could not find core service")
	require.NotNil(t, coreSrvc)

	stateRoot, err := coreSrvc.StorageRoot()
	require.NoError(t, err)
	require.Equal(t, expectedRoot, stateRoot)
}

func balanceKey(t *testing.T, publicKey [32]byte) (storageTrieKey []byte) {
	accountKey := append([]byte("balance:"), publicKey[:]...)
	hash, err := common.Blake2bHash(accountKey)
	require.NoError(t, err)
	return hash[:]
}

func TestInitNode_LoadBalances(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genPath := newTestGenesisAndRuntime(t)

	config.Core.Role = common.FullNodeRole
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	config.ChainSpec = genPath

	err := InitNode(config)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())

	node, err := NewNode(config, ks)
	require.NoError(t, err)

	mgr := node.ServiceRegistry.Get(&state.Service{})

	stateSrv, ok := mgr.(*state.Service)
	require.True(t, ok, "could not find core service")
	require.NotNil(t, stateSrv)

	kr, _ := keystore.NewSr25519Keyring()
	alice := kr.Alice().Public().(*sr25519.PublicKey).AsBytes()

	bal, err := stateSrv.Storage.GetStorage(nil, balanceKey(t, alice))
	require.NoError(t, err)

	const genesisBalance = "0x0000000000000001"
	expected, err := common.HexToBytes(genesisBalance)
	require.NoError(t, err)
	require.Equal(t, expected, bal)
}

func TestNode_PersistGlobalName_WhenInitialize(t *testing.T) {
	globalName := RandomNodeName()

	config := DefaultTestWestendDevConfig(t)
	config.Name = globalName

	config.Core.Role = common.FullNodeRole
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	config.ChainSpec = newTestGenesisAndRuntime(t)

	err := InitNode(config)
	require.NoError(t, err)

	storedName, err := LoadGlobalNodeName(config.BasePath)
	require.NoError(t, err)
	require.Equal(t, globalName, storedName)
}

// newTestGenesisAndRuntime create a new test runtime and a new test genesis
// file with the test runtime stored in raw data and returns the genesis file
func newTestGenesisAndRuntime(t *testing.T) (filename string) {
	runtimeFilePath, err := runtime.GetRuntime(context.Background(), runtime.WESTEND_RUNTIME_v0929)
	require.NoError(t, err)
	runtimeData, err := os.ReadFile(runtimeFilePath)
	require.NoError(t, err)

	gen := NewTestGenesis(t)
	hex := hex.EncodeToString(runtimeData)

	gen.Genesis.Raw = map[string]map[string]string{
		"top": {
			"0x3a636f6465": "0x" + hex,
			"0xcf722c0832b5231d35e29f319ff27389f5032bfc7bfc3ba5ed7839f2042fb99f": "0x0000000000000001",
		},
	}

	genData, err := json.Marshal(gen)
	require.NoError(t, err)

	filename = filepath.Join(t.TempDir(), "genesis.json")
	err = os.WriteFile(filename, genData, os.ModePerm)
	require.NoError(t, err)

	return filename
}
