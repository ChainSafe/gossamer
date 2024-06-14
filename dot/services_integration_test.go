// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package dot

import (
	"net/url"
	"testing"
	"time"

	cfg "github.com/ChainSafe/gossamer/config"

	core "github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/network"
	rpc "github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	babe "github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func Test_nodeBuilder_createBABEService(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	config := DefaultTestWestendDevConfig(t)

	ks := keystore.NewGlobalKeystore()
	ks2 := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks2.Babe.Insert(kr.Alice())

	type args struct {
		cfg              *cfg.Config
		initStateService bool
		ks               KeyStore
		cs               *core.Service
		telemetryMailer  Telemetry
	}
	tests := []struct {
		name     string
		args     args
		expected *babe.Service
		err      error
	}{
		{
			name: "invalid_keystore",
			args: args{
				cfg:              config,
				initStateService: true,
				ks:               ks.Gran,
			},
			expected: nil,
			err:      ErrInvalidKeystoreType,
		},
		{
			name: "empty_keystore",
			args: args{
				cfg:              config,
				initStateService: true,
				ks:               ks.Babe,
			},
			expected: nil,
			err:      ErrNoKeysProvided,
		},
		{
			name: "base_case",
			args: args{
				cfg:              config,
				initStateService: true,
				ks:               ks2.Babe,
			},
			expected: &babe.Service{},
			err:      nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stateSrvc := newStateService(t, ctrl)
			mockBabeBuilder := NewMockServiceBuilder(ctrl)
			if tt.err == nil {
				mockBabeBuilder.EXPECT().NewServiceIFace(
					gomock.AssignableToTypeOf(&babe.ServiceConfig{})).
					DoAndReturn(
						func(cfg *babe.ServiceConfig) (*babe.Service, error) {
							return &babe.Service{}, nil
						})
			}

			builder := nodeBuilder{}
			var got *babe.Service
			if tt.args.initStateService {
				got, err = builder.createBABEServiceWithBuilder(tt.args.cfg, stateSrvc, tt.args.ks, tt.args.cs,
					tt.args.telemetryMailer, mockBabeBuilder)
			} else {
				got, err = builder.createBABEServiceWithBuilder(tt.args.cfg, &state.Service{}, tt.args.ks, tt.args.cs,
					tt.args.telemetryMailer, mockBabeBuilder)
			}

			assert.Equal(t, tt.expected, got)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func Test_nodeBuilder_createCoreService(t *testing.T) {
	t.Parallel()

	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Babe.Insert(kr.Alice())

	networkService := &network.Service{}

	type args struct {
		ks  *keystore.GlobalKeystore
		net *network.Service
	}
	tests := []struct {
		name      string
		args      args
		expectNil bool
		err       error
	}{
		{
			name: "base_case",
			args: args{
				ks:  ks,
				net: networkService,
			},
			expectNil: false,
			err:       nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := DefaultTestWestendDevConfig(t)
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)

			builder := nodeBuilder{}
			got, err := builder.createCoreService(config, tt.args.ks, stateSrvc, tt.args.net)

			assert.ErrorIs(t, err, tt.err)

			// TODO: create interface for core.NewService sa that we can assert.Equal the results
			if tt.expectNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.IsType(t, &core.Service{}, got)
			}
		})
	}
}

func Test_nodeBuilder_createNetworkService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       *Config
		expectNil bool
		err       error
	}{
		{
			name:      "base case",
			expectNil: false,
			err:       nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			config := DefaultTestWestendDevConfig(t)
			stateSrvc := newStateService(t, ctrl)
			no := nodeBuilder{}
			got, err := no.createNetworkService(config, stateSrvc, nil)
			assert.ErrorIs(t, err, tt.err)
			// TODO: create interface for network.NewService to handle assert.Equal test
			if tt.expectNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.IsType(t, &network.Service{}, got)
			}
		})
	}
}

func Test_nodeBuilder_createRPCService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expectNil bool
		err       error
	}{
		{
			name:      "base state",
			expectNil: false,
			err:       nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := DefaultTestWestendDevConfig(t)
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)
			no := nodeBuilder{}
			rpcParams := rpcServiceSettings{
				config: config,
				state:  stateSrvc,
			}
			got, err := no.createRPCService(rpcParams)
			assert.ErrorIs(t, err, tt.err)

			// TODO: create interface for rpc.HTTPServer to handle assert.Equal test
			if tt.expectNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.IsType(t, &rpc.HTTPServer{}, got)
			}
		})
	}
}

func Test_nodeBuilder_createGRANDPAService(t *testing.T) {
	t.Parallel()
	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	ks.Gran.Insert(kr.Alice())

	require.NoError(t, err)
	tests := []struct {
		name      string
		ks        KeyStore
		expectNil bool
		err       error
	}{
		{
			name:      "wrong key type",
			ks:        ks.Babe,
			expectNil: true,
			err:       ErrInvalidKeystoreType,
		},
		{
			name:      "base case",
			ks:        ks.Gran,
			expectNil: false,
			err:       nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := DefaultTestWestendDevConfig(t)
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)
			networkConfig := &network.Config{
				BasePath:   t.TempDir(),
				BlockState: stateSrvc.Block,
				RandSeed:   2,
			}
			networkSrvc, err := network.NewService(networkConfig)
			require.NoError(t, err)
			builder := nodeBuilder{}
			got, err := builder.createGRANDPAService(config, stateSrvc, tt.ks, networkSrvc,
				nil)
			assert.ErrorIs(t, err, tt.err)
			// TODO: create interface for grandpa.NewService to enable testing with assert.Equal
			if tt.expectNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.IsType(t, &grandpa.Service{}, got)
			}
		})
	}
}

func Test_createRuntime(t *testing.T) {
	t.Parallel()
	config := DefaultTestWestendDevConfig(t)

	type args struct {
		config *cfg.Config
		ns     runtime.NodeStorage
	}
	tests := []struct {
		name         string
		args         args
		expectedType interface{}
		err          error
	}{
		{
			name: "wasmer_runtime",
			args: args{
				config: config,
				ns:     runtime.NodeStorage{},
			},
			expectedType: &wazero_runtime.Instance{},
			err:          nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)
			code, err := stateSrvc.Storage.LoadCode(nil)
			require.NoError(t, err)

			got, err := createRuntime(tt.args.config, tt.args.ns, stateSrvc, nil, nil, code)
			assert.ErrorIs(t, err, tt.err)
			if tt.expectedType == nil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.IsType(t, tt.expectedType, got)
			}
		})
	}
}

func Test_nodeBuilder_newSyncService(t *testing.T) {
	t.Parallel()
	finalityGadget := &grandpa.Service{}

	ctrl := gomock.NewController(t)
	stateSrvc := newStateService(t, ctrl)
	networkConfig := &network.Config{
		BasePath:   t.TempDir(),
		BlockState: stateSrvc.Block,
		RandSeed:   2,
	}
	networkService, err := network.NewService(networkConfig)
	require.NoError(t, err)

	type args struct {
		fg              BlockJustificationVerifier
		verifier        *babe.VerificationManager
		cs              *core.Service
		net             *network.Service
		telemetryMailer Telemetry
	}
	tests := []struct {
		name      string
		args      args
		expectNil bool
		err       error
	}{
		{
			name: "base_case",
			args: args{
				fg:              finalityGadget,
				verifier:        nil,
				cs:              nil,
				net:             networkService,
				telemetryMailer: nil,
			},
			expectNil: false,
			err:       nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := DefaultTestWestendDevConfig(t)
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)
			no := nodeBuilder{}
			got, err := no.newSyncService(config, stateSrvc, tt.args.fg, tt.args.verifier, tt.args.cs,
				tt.args.net, tt.args.telemetryMailer)
			assert.ErrorIs(t, err, tt.err)
			if tt.expectNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestCreateStateService(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)
	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	builder := nodeBuilder{}
	stateSrvc, err := builder.createStateService(config)
	require.NoError(t, err)
	require.NotNil(t, stateSrvc)

	err = stateSrvc.DB().Close()
	require.NoError(t, err)
}

func newStateServiceWithoutMock(t *testing.T) *state.Service {
	t.Helper()

	stateConfig := state.Config{
		Path:      t.TempDir(),
		LogLevel:  log.Error,
		Telemetry: telemetry.NoopClient{},
	}
	stateSrvc := state.NewService(stateConfig)
	stateSrvc.UseMemDB()
	genData, genTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialise(&genData, &genesisHeader, &genTrie)
	require.NoError(t, err)

	err = stateSrvc.SetupBase()
	require.NoError(t, err)

	genesisBABEConfig := &types.BabeConfiguration{
		SlotDuration:       1000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: []types.AuthorityRaw{},
		Randomness:         [32]byte{},
		SecondarySlots:     0,
	}
	epochState, err := state.NewEpochStateFromGenesis(stateSrvc.DB(), stateSrvc.Block, genesisBABEConfig)
	require.NoError(t, err)

	stateSrvc.Epoch = epochState

	var rtCfg wazero_runtime.Config

	rtCfg.Storage = rtstorage.NewTrieState(&genTrie)

	rtCfg.CodeHash, err = stateSrvc.Storage.LoadCodeHash(nil)
	require.NoError(t, err)

	rtCfg.NodeStorage = runtime.NodeStorage{}

	rt, err := wazero_runtime.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	stateSrvc.Block.StoreRuntime(stateSrvc.Block.BestBlockHash(), rt)

	return stateSrvc
}

func TestCreateCoreService(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)
	config.Core.Role = common.FullNodeRole
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	stateSrvc := newStateServiceWithoutMock(t)

	ks := keystore.NewGlobalKeystore()
	require.NotNil(t, ks)
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())

	networkSrvc := &network.Service{}

	builder := nodeBuilder{}

	coreSrvc, err := builder.createCoreService(config, ks, stateSrvc, networkSrvc)
	require.NoError(t, err)
	require.NotNil(t, coreSrvc)
}

func TestCreateBlockVerifier(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	builder := nodeBuilder{}
	stateSrvc, err := builder.createStateService(config)
	require.NoError(t, err)
	stateSrvc.Epoch = &state.EpochState{}

	_ = builder.createBlockVerifier(stateSrvc)
	err = stateSrvc.DB().Close()
	require.NoError(t, err)
}

func TestCreateSyncService(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	builder := nodeBuilder{}
	stateSrvc := newStateServiceWithoutMock(t)

	ks := keystore.NewGlobalKeystore()
	require.NotNil(t, ks)

	ver := builder.createBlockVerifier(stateSrvc)

	networkService, err := network.NewService(&network.Config{
		BlockState: stateSrvc.Block,
		BasePath:   config.BasePath,
	})
	require.NoError(t, err)

	coreSrvc, err := builder.createCoreService(config, ks, stateSrvc, networkService)
	require.NoError(t, err)

	_, err = builder.newSyncService(config, stateSrvc, &grandpa.Service{}, ver, coreSrvc, networkService, nil)
	require.NoError(t, err)
}

func TestCreateNetworkService(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	builder := nodeBuilder{}
	stateSrvc := newStateServiceWithoutMock(t)

	networkSrvc, err := builder.createNetworkService(config, stateSrvc, nil)
	require.NoError(t, err)
	require.NotNil(t, networkSrvc)
}

func TestCreateRPCService(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.Core.Role = common.FullNodeRole
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	builder := nodeBuilder{}
	stateSrvc := newStateServiceWithoutMock(t)

	networkSrvc := &network.Service{}

	ks := keystore.NewGlobalKeystore()
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())

	ns, err := builder.createRuntimeStorage(stateSrvc)
	require.NoError(t, err)
	err = builder.loadRuntime(config, ns, stateSrvc, ks, networkSrvc)
	require.NoError(t, err)

	coreSrvc, err := builder.createCoreService(config, ks, stateSrvc, networkSrvc)
	require.NoError(t, err)

	systemInfo := &types.SystemInfo{
		SystemName:    config.System.SystemName,
		SystemVersion: config.System.SystemVersion,
	}
	sysSrvc, err := builder.createSystemService(systemInfo, stateSrvc)
	require.NoError(t, err)

	rpcSettings := rpcServiceSettings{
		config:      config,
		nodeStorage: ns,
		state:       stateSrvc,
		core:        coreSrvc,
		network:     networkSrvc,
		system:      sysSrvc,
	}
	rpcSrvc, err := builder.createRPCService(rpcSettings)
	require.NoError(t, err)
	require.NotNil(t, rpcSrvc)
}

func TestCreateBABEService_Integration(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.Core.Role = common.FullNodeRole
	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	builder := nodeBuilder{}
	stateSrvc := newStateServiceWithoutMock(t)

	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Babe.Insert(kr.Alice())

	ns, err := builder.createRuntimeStorage(stateSrvc)
	require.NoError(t, err)
	err = builder.loadRuntime(config, ns, stateSrvc, ks, &network.Service{})
	require.NoError(t, err)

	coreSrvc, err := builder.createCoreService(config, ks, stateSrvc, &network.Service{})
	require.NoError(t, err)

	bs, err := builder.createBABEService(config, stateSrvc, ks.Babe, coreSrvc, nil)
	require.NoError(t, err)
	require.NotNil(t, bs)
}

func TestCreateGrandpaService(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.Core.Role = common.AuthorityRole
	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	builder := nodeBuilder{}
	stateSrvc := newStateServiceWithoutMock(t)

	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	ks.Gran.Insert(kr.Alice())

	ns, err := builder.createRuntimeStorage(stateSrvc)
	require.NoError(t, err)

	err = builder.loadRuntime(config, ns, stateSrvc, ks, &network.Service{})
	require.NoError(t, err)

	networkConfig := &network.Config{
		BasePath:    t.TempDir(),
		NoBootstrap: true,
		NoMDNS:      true,
	}
	setConfigTestDefaults(t, networkConfig)

	testNetworkService, err := network.NewService(networkConfig)
	require.NoError(t, err)

	gs, err := builder.createGRANDPAService(config, stateSrvc, ks.Gran, testNetworkService, nil)
	require.NoError(t, err)
	require.NotNil(t, gs)
}

func TestNewWebSocketServer(t *testing.T) {
	const addr = "localhost:9546"
	testCalls := []struct {
		call     []byte
		expected []byte
	}{
		{
			call:     []byte(`{"jsonrpc":"2.0","method":"system_name","params":[],"id":1}`),
			expected: []byte(`{"id":1,"jsonrpc":"2.0","result":"gossamer"}` + "\n")}, // working request
		{
			call: []byte(`{"jsonrpc":"2.0","method":"unknown","params":[],"id":2}`),
			// unknown method
			expected: []byte(`{"error":{"code":-32000,"data":null,` +
				`"message":"rpc error method unknown not found"},"id":2,` +
				`"jsonrpc":"2.0"}` + "\n")},
		{
			call: []byte{},
			// empty request
			expected: []byte(`{"jsonrpc":"2.0","error":{"code":-32600,` +
				`"message":"Invalid request"},"id":0}` + "\n")},
		{
			call:     []byte(`{"jsonrpc":"2.0","method":"chain_subscribeNewHeads","params":[],"id":3}`),
			expected: []byte(`{"jsonrpc":"2.0","result":1,"id":3}` + "\n")},
		{
			call:     []byte(`{"jsonrpc":"2.0","method":"state_subscribeStorage","params":[],"id":4}`),
			expected: []byte(`{"jsonrpc":"2.0","result":2,"id":4}` + "\n"),
		},
	}

	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.Core.Role = common.FullNodeRole
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	config.ChainSpec = genFile
	config.RPC.Port = 9545
	config.RPC.WSPort = 9546
	config.RPC.WSExternal = true
	config.System.SystemName = "gossamer"

	err := InitNode(config)
	require.NoError(t, err)

	builder := nodeBuilder{}
	stateSrvc := newStateServiceWithoutMock(t)

	networkSrvc := &network.Service{}

	ks := keystore.NewGlobalKeystore()
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())

	ns, err := builder.createRuntimeStorage(stateSrvc)
	require.NoError(t, err)
	err = builder.loadRuntime(config, ns, stateSrvc, ks, networkSrvc)
	require.NoError(t, err)

	coreSrvc, err := builder.createCoreService(config, ks, stateSrvc, networkSrvc)
	require.NoError(t, err)

	systemInfo := &types.SystemInfo{
		SystemName:    config.System.SystemName,
		SystemVersion: config.System.SystemVersion,
	}
	sysSrvc, err := builder.createSystemService(systemInfo, stateSrvc)
	require.NoError(t, err)

	rpcSettings := rpcServiceSettings{
		config:      config,
		nodeStorage: ns,
		state:       stateSrvc,
		core:        coreSrvc,
		network:     networkSrvc,
		system:      sysSrvc,
	}
	rpcSrvc, err := builder.createRPCService(rpcSettings)
	require.NoError(t, err)
	err = rpcSrvc.Start()
	require.NoError(t, err)

	time.Sleep(time.Second) // give server a second to start

	u := url.URL{Scheme: "ws", Host: addr, Path: "/"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer c.Close()

	for _, item := range testCalls {
		err = c.WriteMessage(websocket.TextMessage, item.call)
		require.NoError(t, err)

		_, message, err := c.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, item.expected, message)
	}
}

func Test_createPprofService(t *testing.T) {
	tests := []struct {
		name     string
		settings cfg.PprofConfig
		notNil   bool
	}{
		{
			name:   "base case",
			notNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createPprofService(tt.settings)
			if tt.notNil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func Test_createDigestHandler(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.Core.Role = common.AuthorityRole
	config.ChainSpec = genFile

	err := InitNode(config)
	require.NoError(t, err)

	builder := nodeBuilder{}
	stateSrvc, err := builder.createStateService(config)
	require.NoError(t, err)

	err = startStateService(*config.State, stateSrvc)
	require.NoError(t, err)

	_, err = builder.createDigestHandler(stateSrvc)
	require.NoError(t, err)
}
