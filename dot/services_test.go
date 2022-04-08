// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/life"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_createRuntimeStorage(t *testing.T) {
	cfg := NewTestConfig(t)

	cfg.Init.Genesis = NewTestGenesisRawFile(t, cfg)

	builder := nodeBuilder{}
	err := builder.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := builder.createStateService(cfg)
	require.NoError(t, err)

	tests := []struct {
		name           string
		service        *state.Service
		expectedBaseDB *state.BaseState
		err            error
	}{
		{
			name:           "working example",
			service:        stateSrvc,
			expectedBaseDB: stateSrvc.Base,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builder.createRuntimeStorage(tt.service)
			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.expectedBaseDB, got.BaseDB)
			assert.NotNil(t, got.LocalStorage)
			assert.NotNil(t, got.PersistentStorage)
		})
	}
}

func Test_createSystemService(t *testing.T) {
	cfg := NewTestConfig(t)

	cfg.Init.Genesis = NewTestGenesisRawFile(t, cfg)

	builder := nodeBuilder{}
	err := builder.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := builder.createStateService(cfg)
	require.NoError(t, err)

	type args struct {
		cfg     *types.SystemInfo
		service *state.Service
	}
	tests := []struct {
		name      string
		args      args
		expectNil bool
		err       error
	}{
		{
			name: "working example",
			args: args{
				service: stateSrvc,
			},
			expectNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builder.createSystemService(tt.args.cfg, tt.args.service)
			assert.ErrorIs(t, err, tt.err)

			// TODO: change this check to assert.Equal after state.Service interface is implemented.
			if tt.expectNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func Test_newInMemoryDB(t *testing.T) {
	tests := []struct {
		name      string
		expectNil bool
		err       error
	}{
		{
			name:      "working example",
			expectNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newInMemoryDB()
			assert.ErrorIs(t, err, tt.err)

			if tt.expectNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

//go:generate mockgen -destination=mock_babe_builder_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/babe ServiceIFace
//go:generate mockgen -destination=mock_service_builder_test.go -package $GOPACKAGE . ServiceBuilder

func Test_nodeBuilder_createBABEService(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBabeIFace := NewMockServiceIFace(ctrl)
	t.Parallel()

	cfg := NewTestConfig(t)

	ks := keystore.NewGlobalKeystore()
	ks2 := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks2.Babe.Insert(kr.Alice())

	type args struct {
		cfg              *Config
		initStateService bool
		ks               keystore.Keystore
		cs               *core.Service
		telemetryMailer  telemetry.Client
	}
	tests := []struct {
		name     string
		args     args
		expected babe.ServiceIFace
		err      error
	}{
		{
			name: "invalid keystore",
			args: args{
				cfg:              cfg,
				initStateService: true,
				ks:               ks.Gran,
			},
			expected: nil,
			err:      ErrInvalidKeystoreType,
		},
		{
			name: "empty keystore",
			args: args{
				cfg:              cfg,
				initStateService: true,
				ks:               ks.Babe,
			},
			expected: nil,
			err:      ErrNoKeysProvided,
		},
		{
			name: "config error",
			args: args{
				cfg:              cfg,
				initStateService: false,
				ks:               ks2.Babe,
			},
			expected: nil,
			err:      babe.ErrNilBlockState,
		},
		{
			name: "base case",
			args: args{
				cfg:              cfg,
				initStateService: true,
				ks:               ks2.Babe,
			},
			expected: mockBabeIFace,
			err:      nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateSrvc := newStateService(t, ctrl)
			mockBabeBuilder := NewMockServiceBuilder(ctrl)
			mockBabeBuilder.EXPECT().NewServiceIFace(
				gomock.AssignableToTypeOf(&babe.ServiceConfig{})).DoAndReturn(func(cfg *babe.ServiceConfig) (babe.
				ServiceIFace, error) {
				if reflect.ValueOf(cfg.BlockState).Kind() == reflect.Ptr && reflect.ValueOf(cfg.BlockState).IsNil() {
					return nil, babe.ErrNilBlockState
				}
				return mockBabeIFace, nil
			}).AnyTimes()
			builder := nodeBuilder{}
			var got babe.ServiceIFace
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

func newStateService(t *testing.T, ctrl *gomock.Controller) *state.Service {
	t.Helper()

	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	stateConfig := state.Config{
		Path:      t.TempDir(),
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
	}
	stateSrvc := state.NewService(stateConfig)
	stateSrvc.UseMemDB()
	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialise(genData, genesisHeader, genTrie)
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

	rtCfg := &wasmer.Config{}

	rtCfg.Storage, err = rtstorage.NewTrieState(genTrie)
	require.NoError(t, err)

	rtCfg.CodeHash, err = stateSrvc.Storage.LoadCodeHash(nil)
	require.NoError(t, err)

	rtCfg.NodeStorage = runtime.NodeStorage{}

	rt, err := wasmer.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	stateSrvc.Block.StoreRuntime(stateSrvc.Block.BestBlockHash(), rt)

	return stateSrvc
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
		dh  *digest.Handler
	}
	tests := []struct {
		name      string
		args      args
		expectNil bool
		err       error
	}{
		{
			name: "base case",
			args: args{
				ks:  ks,
				net: networkService,
			},
			expectNil: false,
			err:       nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewTestConfig(t)
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)

			builder := nodeBuilder{}
			got, err := builder.createCoreService(cfg, tt.args.ks, stateSrvc, tt.args.net, tt.args.dh)

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
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewTestConfig(t)
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)
			no := nodeBuilder{}
			got, err := no.createNetworkService(cfg, stateSrvc, nil)
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
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewTestConfig(t)
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)
			no := nodeBuilder{}
			rpcParams := rpcServiceSettings{
				config: cfg,
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
		ks        keystore.Keystore
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
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewTestConfig(t)
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
			got, err := builder.createGRANDPAService(cfg, stateSrvc, nil, tt.ks, networkSrvc,
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
	cfg := NewTestConfig(t)

	cfgLife := NewTestConfig(t)
	cfgLife.Core.WasmInterpreter = life.Name

	type args struct {
		cfg *Config
		ns  runtime.NodeStorage
	}
	tests := []struct {
		name         string
		args         args
		expectedType interface{}
		err          error
	}{
		{
			name: "wasmer runtime",
			args: args{
				cfg: cfg,
				ns:  runtime.NodeStorage{},
			},
			expectedType: &wasmer.Instance{},
			err:          nil,
		},
		{
			name: "wasmer life",
			args: args{
				cfg: cfgLife,
				ns:  runtime.NodeStorage{},
			},
			expectedType: &life.Instance{},
			err:          nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)
			code, err := stateSrvc.Storage.LoadCode(nil)
			require.NoError(t, err)

			got, err := createRuntime(tt.args.cfg, tt.args.ns, stateSrvc, nil, nil, code)
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
	type args struct {
		fg              sync.FinalityGadget
		verifier        *babe.VerificationManager
		cs              *core.Service
		net             *network.Service
		telemetryMailer telemetry.Client
	}
	tests := []struct {
		name      string
		args      args
		expectNil bool
		err       error
	}{
		{
			name: "base case",
			args: args{
				fg:              finalityGadget,
				verifier:        nil,
				cs:              nil,
				net:             nil,
				telemetryMailer: nil,
			},
			expectNil: false,
			err:       nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewTestConfig(t)
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)
			no := nodeBuilder{}
			got, err := no.newSyncService(cfg, stateSrvc, tt.args.fg, tt.args.verifier, tt.args.cs,
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
