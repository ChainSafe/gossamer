// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	gsync "github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/runtime"
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

func TestNewNodeC(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)
	defer utils.RemoveTestDir(t)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

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
					Init:    InitConfig{Genesis: genFile.Name()},
					Account: AccountConfig{Key: "alice"},
					Core: CoreConfig{
						Roles:           types.FullNodeRole,
						WasmInterpreter: wasmer.Name,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks, err := InitKeystore(tt.args.cfg)
			assert.NoError(t, err)
			got, err := NewNode(tt.args.cfg, ks)
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
	defer ctrl.Finish()

	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)
	defer utils.RemoveTestDir(t)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

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
					Init:    InitConfig{Genesis: genFile.Name()},
					Account: AccountConfig{Key: "alice"},
					Core: CoreConfig{
						Roles:           types.FullNodeRole,
						WasmInterpreter: wasmer.Name,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks, err := InitKeystore(tt.args.cfg)
			assert.NoError(t, err)
			m := NewMocknewNodeIface(ctrl)
			m.EXPECT().nodeInitialised(tt.args.cfg.Global.BasePath).Return(true)
			m.EXPECT().createStateService(tt.args.cfg).Return(&state.Service{}, nil)
			m.EXPECT().createRuntimeStorage(&state.Service{}).Return(&runtime.NodeStorage{}, nil)
			m.EXPECT().loadRuntime(tt.args.cfg, &runtime.NodeStorage{}, &state.Service{}, ks, &network.Service{}).Return(nil)
			m.EXPECT().createBlockVerifier(&state.Service{}).Return(&babe.VerificationManager{}, nil)
			m.EXPECT().createDigestHandler(&state.Service{}).Return(&digest.Handler{}, nil)
			m.EXPECT().createCoreService(tt.args.cfg, ks, &state.Service{}, &network.Service{},
				&digest.Handler{}).Return(&core.Service{}, nil)
			m.EXPECT().createGRANDPAService(tt.args.cfg, &state.Service{}, &digest.Handler{}, ks.Gran,
				&network.Service{}).Return(&grandpa.Service{}, nil)
			m.EXPECT().newSyncService(tt.args.cfg, &state.Service{}, &grandpa.Service{}, &babe.VerificationManager{},
				&core.Service{}, &network.Service{}).Return(&gsync.Service{}, nil)
			m.EXPECT().createBABEService(tt.args.cfg, &state.Service{}, ks.Babe,
				&core.Service{}).Return(&babe.Service{}, nil)
			m.EXPECT().createSystemService(&tt.args.cfg.System, &state.Service{}).Return(&system.Service{}, nil)
			netA := &network.Service{}
			netA.SetTransactionHandler(&core.Service{})
			netA.SetSyncer(&gsync.Service{})
			m.EXPECT().initialiseTelemetry(tt.args.cfg, &state.Service{}, netA,
				&system.Service{})
			m.EXPECT().createNetworkService(tt.args.cfg, &state.Service{}).Return(&network.Service{}, nil)

			got, err := newNode(tt.args.cfg, ks, m)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
				utils.RemoveTestDir(t)
			} else {
				assert.NoError(t, err)
			}

			if tt.want != nil {
				assert.Equal(t, tt.want, got)
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
