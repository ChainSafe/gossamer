// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/system"
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

// TODO: fails atm, 2.5s
func Test_createBlockVerifier(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := newTestGenesisRawFile(t, cfg)

	cfg.Init.Genesis = genFile

	nodeInstance := nodeBuilder{}
	err := nodeInstance.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := nodeInstance.createStateService(cfg)
	require.NoError(t, err)

	stateSrvc.SetBlockState(&state.BlockState{})
	stateSrvc.SetEpochState(&state.EpochState{})

	type args struct {
		st state.Service
	}
	tests := []struct {
		name string
		args args
		want *babe.VerificationManager
		err  error
	}{
		{
			name: "nil BlockState test",
			args: args{st: stateSrvc},
			err:  errors.New("cannot have nil EpochState"),
		},
		{
			name: "working example",
			args: args{st: stateSrvc},
			want: &babe.VerificationManager{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nodeInstance.createBlockVerifier(tt.args.st)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// TODO: fails atm, 2.5s
func Test_createRuntimeStorage(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := newTestGenesisRawFile(t, cfg)

	cfg.Init.Genesis = genFile

	nodeInstance := nodeBuilder{}
	err := nodeInstance.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := nodeInstance.createStateService(cfg)
	require.NoError(t, err)

	type args struct {
		st state.Service
	}
	tests := []struct {
		name string
		args args
		want *runtime.NodeStorage
		err  error
	}{
		{
			name: "working example",
			args: args{st: stateSrvc},
			want: &runtime.NodeStorage{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nodeInstance.createRuntimeStorage(tt.args.st)
			assert.ErrorIs(t, err, tt.err)

			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// TODO: takes 2.5s
func Test_createSystemService(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := newTestGenesisRawFile(t, cfg)

	cfg.Init.Genesis = genFile

	nodeInstance := nodeBuilder{}
	err := nodeInstance.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := nodeInstance.createStateService(cfg)
	require.NoError(t, err)

	type args struct {
		cfg       *types.SystemInfo
		stateSrvc state.Service
	}
	tests := []struct {
		name string
		args args
		want *system.Service
		err  error
	}{
		{
			name: "working example",
			args: args{
				stateSrvc: stateSrvc,
			},
			want: &system.Service{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nodeInstance.createSystemService(tt.args.cfg, tt.args.stateSrvc)
			assert.ErrorIs(t, err, tt.err)

			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func Test_newInMemoryDB(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
		err  error
	}{
		{
			name: "working example",
			args: args{path: "test_data"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newInMemoryDB(tt.args.path)
			assert.ErrorIs(t, err, tt.err)

			if tt.want {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// TODO: takes 3.5s
func Test_nodeBuilder_createBABEService(t *testing.T) {
	t.Parallel()
	stateSrvc := newStateService(t)

	cfg := NewTestConfig(t)

	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Babe.Insert(kr.Alice())

	type args struct {
		cfg             *Config
		st              state.Service
		ks              keystore.Keystore
		cs              *core.Service
		telemetryMailer telemetry.Client
	}
	tests := []struct {
		name string
		args args
		want *babe.Service
		err  error
	}{
		{
			name: "base case",
			args: args{
				cfg: cfg,
				st:  stateSrvc,
				ks:  ks.Babe,
			},
			want: &babe.Service{},
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			no := nodeBuilder{}
			got, err := no.createBABEService(tt.args.cfg, tt.args.st, tt.args.ks, tt.args.cs, tt.args.telemetryMailer)

			assert.ErrorIs(t, err, tt.err)
			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func newStateService(t *testing.T) state.Service {
	t.Helper()

	ctrl := gomock.NewController(t)
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
	epochState, err := state.NewEpochStateFromGenesis(stateSrvc.DB(), stateSrvc.BlockState(), genesisBABEConfig)
	require.NoError(t, err)

	stateSrvc.SetEpochState(epochState)

	rtCfg := &wasmer.Config{}

	rtCfg.Storage, err = rtstorage.NewTrieState(genTrie)
	require.NoError(t, err)

	rtCfg.CodeHash, err = stateSrvc.StorageState().LoadCodeHash(nil)
	require.NoError(t, err)

	rtCfg.NodeStorage = runtime.NodeStorage{}

	rt, err := wasmer.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	stateSrvc.BlockState().StoreRuntime(stateSrvc.BlockState().BestBlockHash(), rt)

	return stateSrvc
}

// TODO: takes 3.5s
func Test_nodeBuilder_createCoreService(t *testing.T) {
	t.Parallel()
	cfg := NewTestConfig(t)
	stateSrvc := newStateService(t)
	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Babe.Insert(kr.Alice())

	networkService := &network.Service{}

	type args struct {
		cfg *Config
		ks  *keystore.GlobalKeystore
		st  state.Service
		net *network.Service
		dh  *digest.Handler
	}
	tests := []struct {
		name string
		args args
		want *core.Service
		err  error
	}{
		{
			name: "base case",
			args: args{
				cfg: cfg,
				ks:  ks,
				st:  stateSrvc,
				net: networkService,
			},
			want: &core.Service{},
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			no := nodeBuilder{}
			got, err := no.createCoreService(tt.args.cfg, tt.args.ks, tt.args.st, tt.args.net, tt.args.dh)

			assert.ErrorIs(t, err, tt.err)

			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// TODO: takes 6.5s
func Test_nodeBuilder_createNetworkService(t *testing.T) {
	t.Parallel()
	cfg := NewTestConfig(t)
	stateSrvc := newStateService(t)
	type args struct {
		cfg             *Config
		stateSrvc       state.Service
		telemetryMailer telemetry.Client
	}
	tests := []struct {
		name string
		args args
		want *network.Service
		err  error
	}{
		{
			name: "base case",
			args: args{
				cfg:       cfg,
				stateSrvc: stateSrvc,
			},
			want: &network.Service{},
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			no := nodeBuilder{}
			got, err := no.createNetworkService(tt.args.cfg, tt.args.stateSrvc, tt.args.telemetryMailer)
			assert.ErrorIs(t, err, tt.err)
			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// TODO: takes 3.6s
func Test_nodeBuilder_createRPCService(t *testing.T) {
	t.Parallel()
	cfg := NewTestConfig(t)
	stateSrvc := newStateService(t)

	type args struct {
		cfg       *Config
		stateSrvc state.Service
	}
	tests := []struct {
		name string
		args args
		want *rpc.HTTPServer
		err  error
	}{
		{
			name: "base state",
			args: args{
				cfg:       cfg,
				stateSrvc: stateSrvc,
			},
			want: &rpc.HTTPServer{},
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			no := nodeBuilder{}
			rpcParams := rpcServiceSettings{
				config: tt.args.cfg,
				state:  tt.args.stateSrvc,
			}
			got, err := no.createRPCService(rpcParams)
			assert.ErrorIs(t, err, tt.err)

			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// TODO: takes 6.5s
func Test_nodeBuilder_createGRANDPAService(t *testing.T) {
	t.Parallel()
	cfg := NewTestConfig(t)
	stateSrvc := newStateService(t)
	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	ks.Gran.Insert(kr.Alice())

	networkConfig := &network.Config{
		BasePath:   t.TempDir(),
		BlockState: stateSrvc.BlockState(),
		RandSeed:   2,
	}
	networkSrvc, err := network.NewService(networkConfig)
	require.NoError(t, err)
	type args struct {
		cfg *Config
		st  state.Service
		ks  keystore.Keystore
		net *network.Service
	}
	tests := []struct {
		name string
		args args
		want *grandpa.Service
		err  error
	}{
		{
			name: "base case",
			args: args{
				cfg: cfg,
				st:  stateSrvc,
				ks:  ks.Gran,
				net: networkSrvc,
			},
			want: &grandpa.Service{},
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			no := nodeBuilder{}
			got, err := no.createGRANDPAService(tt.args.cfg, tt.args.st, nil, tt.args.ks, tt.args.net,
				nil)
			assert.ErrorIs(t, err, tt.err)
			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// TODO: takes 6.5s
func Test_createRuntime(t *testing.T) {
	t.Parallel()
	cfg := NewTestConfig(t)
	stateSrvc := newStateService(t)
	code, err := stateSrvc.StorageState().LoadCode(nil)
	require.NoError(t, err)

	cfgLife := NewTestConfig(t)
	cfgLife.Core.WasmInterpreter = life.Name

	type args struct {
		cfg  *Config
		ns   runtime.NodeStorage
		st   state.Service
		code []byte
	}
	tests := []struct {
		name string
		args args
		want runtime.Instance
		err  error
	}{
		{
			name: "wasmer runtime",
			args: args{
				cfg:  cfg,
				ns:   runtime.NodeStorage{},
				st:   stateSrvc,
				code: code,
			},
			want: &wasmer.Instance{},
			err:  nil,
		},
		{
			name: "wasmer life",
			args: args{
				cfg:  cfgLife,
				ns:   runtime.NodeStorage{},
				st:   stateSrvc,
				code: code,
			},
			want: &life.Instance{},
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createRuntime(tt.args.cfg, tt.args.ns, tt.args.st, nil, nil, tt.args.code)
			assert.ErrorIs(t, err, tt.err)
			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// TODO: takes 6.5s
func Test_nodeBuilder_newSyncService(t *testing.T) {
	t.Parallel()
	cfg := NewTestConfig(t)
	stateSrvc := newStateService(t)
	finalityGadget := &grandpa.Service{}
	type args struct {
		cfg             *Config
		st              state.Service
		fg              sync.FinalityGadget
		verifier        *babe.VerificationManager
		cs              *core.Service
		net             *network.Service
		telemetryMailer telemetry.Client
	}
	tests := []struct {
		name string
		args args
		want *sync.Service
		err  error
	}{
		{
			name: "base case",
			args: args{
				cfg:             cfg,
				st:              stateSrvc,
				fg:              finalityGadget,
				verifier:        nil,
				cs:              nil,
				net:             nil,
				telemetryMailer: nil,
			},
			want: &sync.Service{},
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			no := nodeBuilder{}
			got, err := no.newSyncService(tt.args.cfg, tt.args.st, tt.args.fg, tt.args.verifier, tt.args.cs,
				tt.args.net, tt.args.telemetryMailer)
			assert.ErrorIs(t, err, tt.err)
			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// TODO: takes 3.6s
func Test_nodeBuilder_createDigestHandler(t *testing.T) {
	stateSrvc := newStateService(t)
	type args struct {
		lvl log.Level
		st  state.Service
	}
	tests := []struct {
		name string
		args args
		want *digest.Handler
		err  error
	}{
		{
			name: "base case",
			args: args{
				st: stateSrvc,
			},
			want: &digest.Handler{},
			err:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			no := nodeBuilder{}
			got, err := no.createDigestHandler(tt.args.lvl, tt.args.st)
			assert.ErrorIs(t, err, tt.err)
			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}
