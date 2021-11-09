// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package dot

import (
	"errors"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/services"
)

func TestInitNode(t *testing.T) {
	type args struct {
		cfg *Config
	}
	tests := []struct {
		name    string
		args    args
		err error
	}{
		{
			name:    "no arguments",
			args:    args{cfg: GssmrConfig()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// todo (ed) deal with file path to test this
			fmt.Printf("gen %v\n", tt.args.cfg.Init.Genesis)
			err := InitNode(tt.args.cfg)

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
		err error
	}{
		{
			name:         "working example",
			args: args{basepath: basePath},
			wantNodename: "nodeName",
		},
		{
			name:         "no arguments",
			err: errors.New("Key not found"),
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

func TestNewNode(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	err = keystore.LoadKeystore("alice", ks.Gran)
	require.NoError(t, err)
	err = keystore.LoadKeystore("alice", ks.Babe)
	require.NoError(t, err)

	cfg.Core.Roles = types.FullNodeRole


	type args struct {
		cfg      *Config
		ks       *keystore.GlobalKeystore
		stopFunc func()
	}
	tests := []struct {
		name    string
		args    args
		want    *Node
		err error
	}{
		{
			name:    "missing keystore",
			args:    args{
				cfg: cfg,
			},
			err: errors.New("failed to create core service: cannot have nil keystore"),
		},
		// todo (ed) this second test fails with; failed to create state service: failed to start state service: Cannot acquire directory lock on "/home/emack/projects/ChainSafe/gossamer/dot/test_data/TestNewNode/db".  Another process is using this Badger database.: resource temporarily unavailable
		{
			name:    "working example",
			args:    args{
				cfg:      cfg,
				ks:       ks,
			},
			want:    &Node{Name: "Gossamer"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNode(tt.args.cfg, tt.args.ks, tt.args.stopFunc)
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
	type args struct {
		basepath string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NodeInitialized(tt.args.basepath); got != tt.want {
				t.Errorf("NodeInitialized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNode_Start(t *testing.T) {
	type fields struct {
		Name     string
		Services *services.ServiceRegistry
		StopFunc func()
		wg       sync.WaitGroup
		started  chan struct{}
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				Name:     tt.fields.Name,
				Services: tt.fields.Services,
				StopFunc: tt.fields.StopFunc,
				wg:       tt.fields.wg,
				started:  tt.fields.started,
			}
			if err := n.Start(); (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNode_Stop(t *testing.T) {
	type fields struct {
		Name     string
		Services *services.ServiceRegistry
		StopFunc func()
		wg       sync.WaitGroup
		started  chan struct{}
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				Name:     tt.fields.Name,
				Services: tt.fields.Services,
				StopFunc: tt.fields.StopFunc,
				wg:       tt.fields.wg,
				started:  tt.fields.started,
			}
			fmt.Printf("node %v\n", n)
		})
	}
}

func Test_loadRuntime(t *testing.T) {
	type args struct {
		cfg       *Config
		ns        *runtime.NodeStorage
		stateSrvc *state.Service
		ks        *keystore.GlobalKeystore
		net       *network.Service
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := loadRuntime(tt.args.cfg, tt.args.ns, tt.args.stateSrvc, tt.args.ks, tt.args.net); (err != nil) != tt.wantErr {
				t.Errorf("loadRuntime() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_storeGlobalNodeName(t *testing.T) {
	type args struct {
		name     string
		basepath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := storeGlobalNodeName(tt.args.name, tt.args.basepath); (err != nil) != tt.wantErr {
				t.Errorf("storeGlobalNodeName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
