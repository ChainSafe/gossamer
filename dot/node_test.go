// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	dotsync "github.com/ChainSafe/gossamer/dot/sync"
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
	cfg.Init.Genesis = NewTestGenesisRawFile(t, cfg)
	tests := []struct {
		name   string
		config *Config
		err    error
	}{
		{
			name:   "test config",
			config: cfg,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitNode(tt.config)
			assert.ErrorIs(t, err, tt.err)
			// confirm InitNode has created database dir
			registry := filepath.Join(tt.config.Global.BasePath, utils.DefaultDatabaseDir, "KEYREGISTRY")
			_, err = os.Stat(registry)
			require.NoError(t, err)
		})
	}
}

func TestLoadGlobalNodeName(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	db, err := utils.SetupDatabase(basePath, false)
	require.NoError(t, err)

	basestate := state.NewBaseState(db)
	basestate.Put(common.NodeNameKey, []byte(`nodeName`))

	err = db.Close()
	require.NoError(t, err)

	tests := []struct {
		name         string
		basepath     string
		wantNodename string
		err          error
	}{
		{
			name:         "working example",
			basepath:     basePath,
			wantNodename: "nodeName",
		},
		{
			name:     "wrong basepath test",
			basepath: t.TempDir(),
			err:      errors.New("Key not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodename, err := LoadGlobalNodeName(tt.basepath)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantNodename, gotNodename)
		})
	}
}

//go:generate mockgen -destination=mock_service_registry_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/lib/services Service,ServiceRegisterer

func TestNewNode(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockTelemetryClient := NewMockClient(ctrl)
	mockTelemetryClient.EXPECT().SendMessage(gomock.Any())

	initConfig := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, initConfig)

	networkConfig := &network.Config{
		BasePath:    t.TempDir(),
		NoBootstrap: true,
		NoMDNS:      true,
	}
	setConfigTestDefaults(t, networkConfig)

	testNetworkService, err := network.NewService(networkConfig)
	require.NoError(t, err)

	config := state.Config{
		Path:     initConfig.Global.BasePath,
		LogLevel: initConfig.Log.StateLvl,
	}

	dotConfig := &Config{
		Global:  GlobalConfig{BasePath: initConfig.Global.BasePath},
		Init:    InitConfig{Genesis: genFile},
		Account: AccountConfig{Key: "alice"},
		Core: CoreConfig{
			Roles:           types.FullNodeRole,
			WasmInterpreter: wasmer.Name,
		},
	}

	dotConfig.Init = InitConfig{Genesis: genFile}
	dotConfig.Account = AccountConfig{Key: "alice"}
	dotConfig.Core.Roles = types.FullNodeRole
	dotConfig.Core.WasmInterpreter = wasmer.Name
	dotConfig.Global.Name = "TestNode"

	ks, err := initKeystore(t, dotConfig)
	assert.NoError(t, err)

	mockServiceRegistry := NewMockServiceRegisterer(ctrl)
	mockServiceRegistry.EXPECT().RegisterService(gomock.Any()).Times(8)

	m := NewMocknodeBuilderIface(ctrl)
	m.EXPECT().nodeInitialised(dotConfig.Global.BasePath).Return(nil)
	m.EXPECT().createStateService(dotConfig).DoAndReturn(func(cfg *Config) (*state.Service, error) {
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
	m.EXPECT().loadRuntime(dotConfig, &runtime.NodeStorage{}, gomock.AssignableToTypeOf(&state.Service{}),
		ks, gomock.AssignableToTypeOf(&network.Service{})).Return(nil)
	m.EXPECT().createBlockVerifier(gomock.AssignableToTypeOf(&state.Service{})).Return(&babe.
		VerificationManager{}, nil)
	m.EXPECT().createDigestHandler(log.Critical, gomock.AssignableToTypeOf(&state.Service{})).
		Return(&digest.Handler{}, nil)
	m.EXPECT().createCoreService(dotConfig, ks, gomock.AssignableToTypeOf(&state.Service{}),
		gomock.AssignableToTypeOf(&network.Service{}), &digest.Handler{}).
		Return(&core.Service{}, nil)
	m.EXPECT().createGRANDPAService(dotConfig, gomock.AssignableToTypeOf(&state.Service{}),
		&digest.Handler{}, ks.Gran, gomock.AssignableToTypeOf(&network.Service{}),
		gomock.AssignableToTypeOf(&telemetry.Mailer{})).
		Return(&grandpa.Service{}, nil)
	m.EXPECT().newSyncService(dotConfig, gomock.AssignableToTypeOf(&state.Service{}), &grandpa.Service{},
		&babe.VerificationManager{}, &core.Service{}, gomock.AssignableToTypeOf(&network.Service{}),
		gomock.AssignableToTypeOf(&telemetry.Mailer{})).
		Return(&dotsync.Service{}, nil)
	m.EXPECT().createBABEService(dotConfig, gomock.AssignableToTypeOf(&state.Service{}), ks.Babe,
		&core.Service{}, gomock.AssignableToTypeOf(&telemetry.Mailer{})).
		Return(&babe.Service{}, nil)
	m.EXPECT().createSystemService(&dotConfig.System, gomock.AssignableToTypeOf(&state.Service{})).
		DoAndReturn(func(cfg *types.SystemInfo, stateSrvc *state.Service) (*system.Service, error) {
			gd, err := stateSrvc.Base.LoadGenesisData()
			systemService := system.NewService(cfg, gd)
			return systemService, err
		})
	m.EXPECT().createNetworkService(dotConfig, gomock.AssignableToTypeOf(&state.Service{}),
		gomock.AssignableToTypeOf(&telemetry.Mailer{})).Return(testNetworkService, nil)

	got, err := newNode(dotConfig, ks, m, mockServiceRegistry)
	assert.NoError(t, err)

	expected := &Node{
		Name: "TestNode",
	}

	assert.Equal(t, expected.Name, got.Name)
}

//go:generate mockgen -destination=mock_block_state_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/network BlockState

func setConfigTestDefaults(t *testing.T, cfg *network.Config) {
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
		blockstate := NewMockBlockState(ctrl)

		cfg.BlockState = blockstate
	}

	cfg.SlotDuration = time.Second

	if cfg.Telemetry == nil {
		telemetryMock := NewMockClient(ctrl)
		telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()
		cfg.Telemetry = telemetryMock
	}
}

func TestNodeInitialized(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	cfg.Init.Genesis = genFile

	nodeInstance := nodeBuilder{}
	err := nodeInstance.initNode(cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		basepath string
		want     bool
	}{
		{
			name:     "blank base path",
			basepath: "",
			want:     false,
		},
		{
			name:     "working example",
			basepath: cfg.Global.BasePath,
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NodeInitialized(tt.basepath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func initKeystore(t *testing.T, cfg *Config) (*keystore.GlobalKeystore, error) {
	ks := keystore.NewGlobalKeystore()
	// load built-in test keys if specified by `cfg.Account.Key`
	err := keystore.LoadKeystore(cfg.Account.Key, ks.Acco)
	require.NoError(t, err)

	err = keystore.LoadKeystore(cfg.Account.Key, ks.Babe)
	require.NoError(t, err)

	err = keystore.LoadKeystore(cfg.Account.Key, ks.Gran)
	require.NoError(t, err)

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
				Name:            tt.fields.Name,
				ServiceRegistry: tt.fields.Services,
				started:         tt.fields.started,
				metricsServer:   tt.fields.metricsServer,
			}
			go func() {
				<-n.started
				n.Stop()
			}()

			err := n.Start()
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_nodeBuilder_loadRuntime(t *testing.T) {
	cfg := NewTestConfig(t)
	type args struct {
		cfg *Config
		ns  *runtime.NodeStorage
		ks  *keystore.GlobalKeystore
		net *network.Service
	}
	tests := []struct {
		name string
		args args
		err  error
	}{
		{
			name: "base case",
			args: args{
				cfg: cfg,
				ns:  &runtime.NodeStorage{},
				ks:  nil,
				net: nil,
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateSrvc := newStateService(t, ctrl)
			no := nodeBuilder{}
			err := no.loadRuntime(tt.args.cfg, tt.args.ns, stateSrvc, tt.args.ks, tt.args.net)
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
