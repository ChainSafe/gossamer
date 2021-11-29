// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_createBABEService(t *testing.T) {
	t.Parallel()

	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Babe.Insert(kr.Alice())

	ns, err := createRuntimeStorage(stateSrvc)
	require.NoError(t, err)
	err = loadRuntime(cfg, ns, stateSrvc, ks, &network.Service{})
	require.NoError(t, err)

	dh, err := createDigestHandler(stateSrvc)
	require.NoError(t, err)

	coreSrvc, err := createCoreService(cfg, ks, stateSrvc, &network.Service{}, dh)
	require.NoError(t, err)

	type args struct {
		cfg *Config
		st  *state.Service
		ks  keystore.Keystore
		cs  *core.Service
	}
	tests := []struct {
		name string
		args args
		want *babe.Service
		err  error
	}{
		{
			name: "invalid keystore type test",
			args: args{
				cfg: cfg,
				st:  stateSrvc,
				ks:  ks.Gran,
				cs:  coreSrvc,
			},
			err: errors.New("invalid keystore type"),
		},
		{
			name: "working example",
			args: args{
				cfg: cfg,
				st:  stateSrvc,
				ks:  ks.Babe,
				cs:  coreSrvc,
			},
			want: &babe.Service{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createBABEService(tt.args.cfg, tt.args.st, tt.args.ks, tt.args.cs)
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

func Test_createBlockVerifier(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	type args struct {
		st *state.Service
	}
	tests := []struct {
		name string
		args args
		want *babe.VerificationManager
		err  error
	}{
		{
			name: "nil BlockState test",
			args: args{st: &state.Service{}},
			err:  errors.New("cannot have nil BlockState"),
		},
		{
			name: "working example",
			args: args{st: stateSrvc},
			want: &babe.VerificationManager{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createBlockVerifier(tt.args.st)
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

func Test_createCoreService(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	require.NotNil(t, ks)
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())

	type args struct {
		cfg *Config
		ks  *keystore.GlobalKeystore
		st  *state.Service
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
			name: "missing keystore test",
			args: args{
				cfg: cfg,
				st:  stateSrvc,
			},
			err: errors.New("cannot have nil keystore"),
		},
		{
			name: "working example",
			args: args{
				cfg: cfg,
				ks:  ks,
				st:  stateSrvc,
			},
			want: &core.Service{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createCoreService(tt.args.cfg, tt.args.ks, tt.args.st, tt.args.net, tt.args.dh)
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

func Test_createDigestHandler(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	type args struct {
		st *state.Service
	}
	tests := []struct {
		name string
		args args
		want *digest.Handler
		err  error
	}{
		{
			name: "working example",
			args: args{st: stateSrvc},
			want: &digest.Handler{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createDigestHandler(tt.args.st)
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

func Test_createGRANDPAService(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Core.Roles = types.AuthorityRole
	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	ks.Gran.Insert(kr.Alice())

	ns, err := createRuntimeStorage(stateSrvc)
	require.NoError(t, err)

	err = loadRuntime(cfg, ns, stateSrvc, ks, &network.Service{})
	require.NoError(t, err)

	dh, err := createDigestHandler(stateSrvc)
	require.NoError(t, err)

	type args struct {
		cfg *Config
		st  *state.Service
		dh  *digest.Handler
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
			name: "invalid key type test",
			args: args{
				cfg: cfg,
				st:  stateSrvc,
				dh:  dh,
				ks:  ks.Babe,
			},
			err: errors.New("invalid keystore type"),
		},
		{
			name: "working example",
			args: args{
				cfg: cfg,
				st:  stateSrvc,
				dh:  dh,
				ks:  ks.Gran,
			},
			want: &grandpa.Service{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createGRANDPAService(tt.args.cfg, tt.args.st, tt.args.dh, tt.args.ks, tt.args.net)
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

func Test_createNetworkService(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	type args struct {
		cfg       *Config
		stateSrvc *state.Service
	}
	tests := []struct {
		name string
		args args
		want *network.Service
		err  error
	}{
		{
			name: "working example",
			args: args{
				cfg:       cfg,
				stateSrvc: stateSrvc,
			},
			want: &network.Service{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createNetworkService(tt.args.cfg, tt.args.stateSrvc)
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

func Test_createRPCService(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	type args struct {
		cfg         *Config
		ns          *runtime.NodeStorage
		stateSrvc   *state.Service
		coreSrvc    *core.Service
		networkSrvc *network.Service
		bp          modules.BlockProducerAPI
		sysSrvc     *system.Service
		finSrvc     *grandpa.Service
	}
	tests := []struct {
		name string
		args args
		want *rpc.HTTPServer
		err  error
	}{
		{
			name: "working example",
			args: args{
				cfg:       cfg,
				stateSrvc: stateSrvc,
			},
			want: &rpc.HTTPServer{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createRPCService(tt.args.cfg, tt.args.ns, tt.args.stateSrvc, tt.args.coreSrvc, tt.args.networkSrvc, tt.args.bp, tt.args.sysSrvc, tt.args.finSrvc)
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

func Test_createRuntime(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	_ = wasmer.NewTestInstance(t, runtime.NODE_RUNTIME)
	runtimeFilePath := runtime.GetAbsolutePath(runtime.NODE_RUNTIME_FP)

	runtimeData, err := ioutil.ReadFile(filepath.Clean(runtimeFilePath))
	require.Nil(t, err)

	type args struct {
		cfg  *Config
		ns   runtime.NodeStorage
		st   *state.Service
		ks   *keystore.GlobalKeystore
		net  *network.Service
		code []byte
	}
	tests := []struct {
		name string
		args args
		want bool
		err  error
	}{
		{
			name: "empty code test",
			args: args{
				cfg: cfg,
				st:  stateSrvc,
			},
			err: errors.New("failed to create runtime executor: code is empty"),
		},
		{
			name: "bad code test",
			args: args{
				cfg:  cfg,
				st:   stateSrvc,
				code: []byte(`fake code`),
			},
			err: errors.New("failed to create runtime executor: Failed to instantiate the module:\n    compile error: Validation error \"Bad magic number\""),
		},
		{
			name: "working example",
			args: args{
				cfg:  cfg,
				st:   stateSrvc,
				code: runtimeData,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createRuntime(tt.args.cfg, tt.args.ns, tt.args.st, tt.args.ks, tt.args.net, tt.args.code)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.want {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func Test_createRuntimeStorage(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	type args struct {
		st *state.Service
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
			got, err := createRuntimeStorage(tt.args.st)
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

func Test_createStateService(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	cfg2 := NewTestConfig(t)
	cfg2.Global.BasePath = "test_data"

	type args struct {
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want *state.Service
		err  error
	}{
		{
			name: "working example",
			args: args{cfg: cfg},
			want: &state.Service{},
		},
		{
			name: "broken config test",
			args: args{cfg: cfg2},
			err:  errors.New("failed to start state service: failed to create block state: cannot get block 0: Key not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ni.createStateService(tt.args.cfg)
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

func Test_createSystemService(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	type args struct {
		cfg       *types.SystemInfo
		stateSrvc *state.Service
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
			got, err := createSystemService(tt.args.cfg, tt.args.stateSrvc)
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
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.want {
				assert.NotNil(t, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func Test_newSyncService(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisFile(t, cfg)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	ni := nodeInterface{}
	err := ni.initNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := ni.createStateService(cfg)
	require.NoError(t, err)

	type args struct {
		cfg      *Config
		st       *state.Service
		fg       sync.FinalityGadget
		verifier *babe.VerificationManager
		cs       *core.Service
		net      *network.Service
	}
	tests := []struct {
		name string
		args args
		want *sync.Service
		err  error
	}{
		{
			name: "missing FinalityGadget test",
			args: args{
				cfg: cfg,
				st:  stateSrvc,
			},
			err: errors.New("cannot have nil FinalityGadget"),
		},
		{
			name: "working example",
			args: args{
				cfg: cfg,
				st:  stateSrvc,
				fg:  &grandpa.Service{},
			},
			want: &sync.Service{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newSyncService(tt.args.cfg, tt.args.st, tt.args.fg, tt.args.verifier, tt.args.cs, tt.args.net)
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
