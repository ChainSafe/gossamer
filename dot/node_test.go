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
	"fmt"
	"reflect"
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
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := InitNode(tt.args.cfg); (err != nil) != tt.wantErr {
				t.Errorf("InitNode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadGlobalNodeName(t *testing.T) {
	type args struct {
		basepath string
	}
	tests := []struct {
		name         string
		args         args
		wantNodename string
		wantErr      bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodename, err := LoadGlobalNodeName(tt.args.basepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadGlobalNodeName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotNodename != tt.wantNodename {
				t.Errorf("LoadGlobalNodeName() gotNodename = %v, want %v", gotNodename, tt.wantNodename)
			}
		})
	}
}

func TestNewNode(t *testing.T) {
	type args struct {
		cfg      *Config
		ks       *keystore.GlobalKeystore
		stopFunc func()
	}
	tests := []struct {
		name    string
		args    args
		want    *Node
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNode(tt.args.cfg, tt.args.ks, tt.args.stopFunc)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNode() got = %v, want %v", got, tt.want)
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
