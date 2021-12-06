// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitNode(t *testing.T) {
	cfg := NewTestConfig(t)
	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

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
			ni := nodeInterface{}
			err := ni.initNode(tt.args.cfg)

			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadGlobalNodeName(t *testing.T) {
	t.Parallel()

	// initialise database using data directory
	basePath := utils.NewTestBasePath(t, "tmpBase")
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
			args: args{basepath: "test_data"},
			err:  errors.New("Key not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodename, err := LoadGlobalNodeName(tt.args.basepath)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantNodename, gotNodename)
		})
	}
}

func TestNewNodeB(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)
	defer utils.RemoveTestDir(t)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	type args struct {
		cfg      *Config
		stopFunc func()
	}
	tests := []struct {
		name string
		args args
		want *Node
		err  error
	}{
		{
			name: "missing account key",
			args: args{
				cfg: &Config{
					Global: GlobalConfig{BasePath: cfg.Global.BasePath},
					Init:   InitConfig{Genesis: genFile.Name()},
					Core:   CoreConfig{Roles: types.AuthorityRole},
				},
			},
			err: errors.New("no keys provided for authority node"),
		},
		// TODO this is commented out because in holds a lock on badger db, causing next test to foil
		//{
		//	name: "missing wasm config",
		//	args: args{
		//		cfg: &Config{
		//			Global: GlobalConfig{BasePath: cfg.Global.BasePath},
		//			Init:   InitConfig{Genesis: genFile.Name()},
		//			Account: AccountConfig{Key: "alice"},
		//		},
		//	},
		//	err:  errors.New("failed to get runtime instance"),
		//},
		{
			name: "minimal config",
			args: args{
				cfg: &Config{
					Global:  GlobalConfig{BasePath: cfg.Global.BasePath},
					Init:    InitConfig{Genesis: genFile.Name()},
					Account: AccountConfig{Key: "alice"},
					Core:    CoreConfig{WasmInterpreter: wasmer.Name},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNodeB(tt.args.cfg, tt.args.stopFunc)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
				utils.RemoveTestDir(t)
			} else {
				assert.NoError(t, err)
			}

			if tt.want != nil {
				assert.Equal(t, tt.want.Name, got.Name)
			}
		})
	}
}

func TestNewNodeC(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)
	defer utils.RemoveTestDir(t)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	type args struct {
		cfg *Config
		//stopFunc func()
	}
	tests := []struct {
		name string
		args args
		want *Node
		err  error
	}{
		{
			name: "missing account key",
			args: args{
				cfg: &Config{
					Global: GlobalConfig{BasePath: cfg.Global.BasePath},
					Init:   InitConfig{Genesis: genFile.Name()},
					Core:   CoreConfig{Roles: types.AuthorityRole},
				},
			},
			err: errors.New("no keys provided for authority node"),
		},
		{
			name: "minimal config",
			args: args{
				cfg: &Config{
					Global:  GlobalConfig{BasePath: cfg.Global.BasePath},
					Init:    InitConfig{Genesis: genFile.Name()},
					Account: AccountConfig{Key: "alice"},
					Core:    CoreConfig{WasmInterpreter: wasmer.Name},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNodeC(tt.args.cfg)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
				utils.RemoveTestDir(t)
			} else {
				assert.NoError(t, err)
			}

			if tt.want != nil {
				assert.Equal(t, tt.want.Name, got.Name)
			}
		})
	}
}

func TestNewNodeMock(t *testing.T) {
	ctrl := gomock.NewController(t)

	m := NewMocknewNodeIface(ctrl)
	m.EXPECT().nodeInitialised(gomock.Any()).Return(true).AnyTimes()
	m.EXPECT().initKeystore(gomock.Any()).DoAndReturn(func(config *Config) (*keystore.GlobalKeystore, error) {
		if len(config.Account.Key) == 0 {
			return nil, errors.New("no keys provided for authority node")
		}
		return &keystore.GlobalKeystore{}, nil
	}).AnyTimes()
	m.EXPECT().createStateService(gomock.Any()).Return(&state.Service{}, nil).AnyTimes()
	m.EXPECT().createRuntimeStorage(gomock.Any()).AnyTimes()
	m.EXPECT().loadRuntime(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().createBlockVerifier(gomock.Any()).AnyTimes()
	m.EXPECT().createDigestHandler(gomock.Any()).AnyTimes()
	m.EXPECT().createCoreService(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().createGRANDPAService(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().newSyncService(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any()).AnyTimes()
	m.EXPECT().createBABEService(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().createSystemService(gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().initialiseTelemetry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	m.EXPECT().createNetworkService(gomock.Any(), gomock.Any()).AnyTimes()

	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)
	defer utils.RemoveTestDir(t)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	type args struct {
		cfg *Config
		//stopFunc func()
	}
	tests := []struct {
		name string
		args args
		want *Node
		err  error
	}{
		{
			name: "missing account key",
			args: args{
				cfg: &Config{
					Global: GlobalConfig{BasePath: cfg.Global.BasePath},
					Init:   InitConfig{Genesis: genFile.Name()},
					Core:   CoreConfig{Roles: types.AuthorityRole},
				},
			},
			err: errors.New("no keys provided for authority node"),
		},
		{
			name: "minimal config",
			args: args{
				cfg: &Config{
					Global:  GlobalConfig{BasePath: cfg.Global.BasePath},
					Init:    InitConfig{Genesis: genFile.Name()},
					Account: AccountConfig{Key: "alice"},
					Core:    CoreConfig{WasmInterpreter: wasmer.Name},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newNodeC(tt.args.cfg, m)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
				utils.RemoveTestDir(t)
			} else {
				assert.NoError(t, err)
			}

			if tt.want != nil {
				assert.Equal(t, tt.want.Name, got.Name)
			}
		})
	}
}

func TestNewNode(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()
	cfg.Core.GrandpaAuthority = false
	cfg.Core.BABELead = true

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	err = keystore.LoadKeystore("alice", ks.Gran)
	require.NoError(t, err)
	err = keystore.LoadKeystore("alice", ks.Babe)
	require.NoError(t, err)

	cfg.Core.Roles = types.FullNodeRole

	type args struct {
		cfg *Config
		ks  *keystore.GlobalKeystore
	}
	tests := []struct {
		name string
		args args
		want *Node
		err  error
	}{
		{
			name: "working example",
			args: args{
				cfg: cfg,
				ks:  ks,
			},
			want: &Node{Name: "Gossamer"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNode(tt.args.cfg, tt.args.ks)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.want != nil {
				assert.Equal(t, tt.want.Name, got.Name)
			}
		})
	}
}

func TestNodeInitialized(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
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
