// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	gsync "github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/metrics"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitNode(t *testing.T) {
	cfg := NewTestConfig(t)
	genFile := newTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	cfg.Init.Genesis = genFile

	type args struct {
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		err  error
	}{
		{
			name: "test config",
			args: args{cfg: cfg},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitNode(tt.args.cfg)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestLoadGlobalNodeName(t *testing.T) {
	t.Parallel()

	// initialise database using data directory
	basePath := t.TempDir()
	db, err := utils.SetupDatabase(basePath, false)
	require.NoError(t, err)

	basestate := state.NewBaseState(db)
	basestate.Put(common.NodeNameKey, []byte(`nodeName`))

	err = db.Close()
	require.NoError(t, err)

	type args struct {
		basepath string
	}
	tests := []struct {
		name         string
		args         args
		wantNodename string
		err          error
	}{
		{
			name:         "working example",
			args:         args{basepath: basePath},
			wantNodename: "nodeName",
		},
		{
			name: "wrong basepath test",
			args: args{basepath: "wrong_path"},
			err:  errors.New("Key not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodename, err := LoadGlobalNodeName(tt.args.basepath)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
				err := os.RemoveAll(tt.args.basepath)
				require.NoError(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantNodename, gotNodename)
		})
	}
}

func TestNewNode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTelemetryClient := NewMockClient(ctrl)
	mockTelemetryClient.EXPECT().SendMessage(gomock.Any())

	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := newTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)
	networkConfig := &network.Config{
		BasePath:    t.TempDir(),
		NoBootstrap: true,
		NoMDNS:      true,
	}
	testNetwork := createTestService(t, networkConfig)

	config := state.Config{
		Path:     cfg.Global.BasePath,
		LogLevel: cfg.Log.StateLvl,
	}

	type args struct {
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want *Node
		err  error
	}{
		{
			name: "minimal config",
			args: args{
				cfg: &Config{
					Global:  GlobalConfig{BasePath: cfg.Global.BasePath},
					Init:    InitConfig{Genesis: genFile},
					Account: AccountConfig{Key: "alice"},
					Core: CoreConfig{
						Roles:           types.FullNodeRole,
						WasmInterpreter: wasmer.Name,
					},
				},
			},
			want: &Node{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks, err := initKeystore(tt.args.cfg)
			assert.NoError(t, err)
			m := NewMocknodeBuilderIface(ctrl)
			m.EXPECT().nodeInitialised(tt.args.cfg.Global.BasePath).Return(true)
			m.EXPECT().createStateService(tt.args.cfg).DoAndReturn(func(cfg *Config) (*state.Service, error) {
				stateSrvc := state.NewService(config)
				// create genesis from configuration file
				gen, err := genesis.NewGenesisFromJSONRaw(cfg.Init.Genesis)
				if err != nil {
					return nil, fmt.Errorf("failed to load genesis from file: %w", err)
				}
				// create trie from genesis
				trie, err := genesis.NewTrieFromGenesis(gen)
				if err != nil {
					return nil, fmt.Errorf("failed to create trie from genesis: %w", err)
				}
				// create genesis block from trie
				header, err := genesis.NewGenesisBlockFromTrie(trie)
				if err != nil {
					return nil, fmt.Errorf("failed to create genesis block from trie: %w", err)
				}
				stateSrvc.Telemetry = mockTelemetryClient
				err = stateSrvc.Initialise(gen, header, trie)
				if err != nil {
					return nil, fmt.Errorf("failed to initialise state service: %s", err)
				}

				err = stateSrvc.SetupBase()
				if err != nil {
					return nil, fmt.Errorf("cannot setup base: %w", err)
				}
				return stateSrvc, nil
			})

			m.EXPECT().createRuntimeStorage(gomock.AssignableToTypeOf(&state.Service{})).Return(&runtime.
				NodeStorage{}, nil)
			m.EXPECT().loadRuntime(tt.args.cfg, &runtime.NodeStorage{}, gomock.AssignableToTypeOf(&state.Service{}),
				ks, gomock.AssignableToTypeOf(&network.Service{})).Return(nil)
			m.EXPECT().createBlockVerifier(gomock.AssignableToTypeOf(&state.Service{})).Return(&babe.
				VerificationManager{}, nil)
			m.EXPECT().createDigestHandler(log.Critical, gomock.AssignableToTypeOf(&state.Service{})).Return(&digest.
				Handler{}, nil)
			m.EXPECT().createCoreService(tt.args.cfg, ks, gomock.AssignableToTypeOf(&state.Service{}),
				gomock.AssignableToTypeOf(&network.Service{}), &digest.Handler{}).Return(&core.Service{}, nil)
			m.EXPECT().createGRANDPAService(tt.args.cfg, gomock.AssignableToTypeOf(&state.Service{}),
				&digest.Handler{}, ks.Gran, gomock.AssignableToTypeOf(&network.Service{}),
				gomock.AssignableToTypeOf(&telemetry.Mailer{})).Return(&grandpa.Service{}, nil)
			m.EXPECT().newSyncService(tt.args.cfg, gomock.AssignableToTypeOf(&state.Service{}), &grandpa.Service{},
				&babe.VerificationManager{}, &core.Service{}, gomock.AssignableToTypeOf(&network.Service{}),
				gomock.AssignableToTypeOf(&telemetry.Mailer{})).Return(&gsync.Service{}, nil)
			m.EXPECT().createBABEService(tt.args.cfg, gomock.AssignableToTypeOf(&state.Service{}), ks.Babe,
				&core.Service{}, gomock.AssignableToTypeOf(&telemetry.Mailer{})).Return(&babe.Service{}, nil)
			m.EXPECT().createSystemService(&tt.args.cfg.System, gomock.AssignableToTypeOf(&state.Service{})).
				DoAndReturn(func(cfg *types.SystemInfo, stateSrvc *state.Service) (*system.Service, error) {
					gd, err := stateSrvc.Base.LoadGenesisData()
					return system.NewService(cfg, gd), err
				})
			m.EXPECT().createNetworkService(tt.args.cfg, gomock.AssignableToTypeOf(&state.Service{}),
				gomock.AssignableToTypeOf(&telemetry.Mailer{})).Return(testNetwork, nil)

			got, err := newNode(tt.args.cfg, ks, m)
			assert.ErrorIs(t, err, tt.err)

			if tt.want != nil {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func createTestService(t *testing.T, cfg *network.Config) (srvc *network.Service) {
	t.Helper()
	ctrl := gomock.NewController(t)

	if cfg == nil {
		cfg = &network.Config{
			BasePath:     t.TempDir(),
			NoBootstrap:  true,
			NoMDNS:       true,
			LogLvl:       log.Warn,
			SlotDuration: time.Second,
		}
	}
	if cfg.BlockState == nil {
		blockstate := network.NewMockBlockState(ctrl)

		cfg.BlockState = blockstate
	}

	cfg.SlotDuration = time.Second

	if cfg.Telemetry == nil {
		telemetryMock := NewMockClient(ctrl)
		telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()
		cfg.Telemetry = telemetryMock
	}

	srvc, err := network.NewService(cfg)
	require.NoError(t, err)

	return srvc
}

func TestNodeInitialized(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := newTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	cfg.Init.Genesis = genFile

	nodeInstance := nodeBuilder{}
	err := nodeInstance.initNode(cfg)
	require.NoError(t, err)

	type args struct {
		basepath string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "blank base path",
			args: args{basepath: ""},
			want: false,
		},
		{
			name: "working example",
			args: args{basepath: cfg.Global.BasePath},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NodeInitialized(tt.args.basepath); got != tt.want {
				t.Errorf("NodeInitialized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func initKeystore(cfg *Config) (*keystore.GlobalKeystore, error) {
	ks := keystore.NewGlobalKeystore()
	// load built-in test keys if specified by `cfg.Account.Key`
	err := keystore.LoadKeystore(cfg.Account.Key, ks.Acco)
	if err != nil {
		logger.Errorf("failed to load account keystore: %s", err)
		return nil, err
	}

	err = keystore.LoadKeystore(cfg.Account.Key, ks.Babe)
	if err != nil {
		logger.Errorf("failed to load BABE keystore: %s", err)
		return nil, err
	}

	err = keystore.LoadKeystore(cfg.Account.Key, ks.Gran)
	if err != nil {
		logger.Errorf("failed to load grandpa keystore: %s", err)
		return nil, err
	}

	// if authority node, should have at least 1 key in keystore
	if cfg.Core.Roles == types.AuthorityRole && (ks.Babe.Size() == 0 || ks.Gran.Size() == 0) {
		return nil, ErrNoKeysProvided
	}

	return ks, nil
}

func TestNode_StartStop(t *testing.T) {
	serviceRegistryLogger := logger.New(log.AddContext("pkg", "services"))
	type fields struct {
		Name          string
		Services      *services.ServiceRegistry
		started       chan struct{}
		metricsServer *metrics.Server
	}
	tests := []struct {
		name   string
		fields fields
		err    error
	}{
		{
			name: "base case",
			fields: fields{
				Name:     "Node",
				Services: services.NewServiceRegistry(serviceRegistryLogger),
				started:  make(chan struct{}),
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				Name:          tt.fields.Name,
				Services:      tt.fields.Services,
				started:       tt.fields.started,
				metricsServer: tt.fields.metricsServer,
			}

			go func() {
				<-n.started
				n.Stop()
			}()
			err := n.Start()
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func Test_nodeBuilder_loadRuntime(t *testing.T) {
	stateSrvc := newStateService(t)
	cfg := NewTestConfig(t)

	type args struct {
		cfg       *Config
		ns        *runtime.NodeStorage
		stateSrvc *state.Service
		ks        *keystore.GlobalKeystore
		net       *network.Service
	}
	tests := []struct {
		name string
		args args
		err  error
	}{
		{
			name: "base case",
			args: args{
				cfg:       cfg,
				ns:        &runtime.NodeStorage{},
				stateSrvc: stateSrvc,
				ks:        nil,
				net:       nil,
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			no := nodeBuilder{}
			err := no.loadRuntime(tt.args.cfg, tt.args.ns, tt.args.stateSrvc, tt.args.ks, tt.args.net)

			assert.ErrorIs(t, err, tt.err)
		})
	}
}
