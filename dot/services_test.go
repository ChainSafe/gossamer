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
	"reflect"
	"testing"

	"github.com/ChainSafe/chaindb"
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
)

func Test_createBABEService(t *testing.T) {
	type args struct {
		cfg *Config
		st  *state.Service
		ks  keystore.Keystore
		cs  *core.Service
	}
	tests := []struct {
		name    string
		args    args
		want    *babe.Service
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createBABEService(tt.args.cfg, tt.args.st, tt.args.ks, tt.args.cs)
			if (err != nil) != tt.wantErr {
				t.Errorf("createBABEService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createBABEService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createBlockVerifier(t *testing.T) {
	type args struct {
		st *state.Service
	}
	tests := []struct {
		name    string
		args    args
		want    *babe.VerificationManager
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createBlockVerifier(tt.args.st)
			if (err != nil) != tt.wantErr {
				t.Errorf("createBlockVerifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createBlockVerifier() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createCoreService(t *testing.T) {
	type args struct {
		cfg *Config
		ks  *keystore.GlobalKeystore
		st  *state.Service
		net *network.Service
		dh  *digest.Handler
	}
	tests := []struct {
		name    string
		args    args
		want    *core.Service
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createCoreService(tt.args.cfg, tt.args.ks, tt.args.st, tt.args.net, tt.args.dh)
			if (err != nil) != tt.wantErr {
				t.Errorf("createCoreService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createCoreService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createDigestHandler(t *testing.T) {
	type args struct {
		st *state.Service
	}
	tests := []struct {
		name    string
		args    args
		want    *digest.Handler
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createDigestHandler(tt.args.st)
			if (err != nil) != tt.wantErr {
				t.Errorf("createDigestHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createDigestHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createGRANDPAService(t *testing.T) {
	type args struct {
		cfg *Config
		st  *state.Service
		dh  *digest.Handler
		ks  keystore.Keystore
		net *network.Service
	}
	tests := []struct {
		name    string
		args    args
		want    *grandpa.Service
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createGRANDPAService(tt.args.cfg, tt.args.st, tt.args.dh, tt.args.ks, tt.args.net)
			if (err != nil) != tt.wantErr {
				t.Errorf("createGRANDPAService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createGRANDPAService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createNetworkService(t *testing.T) {
	type args struct {
		cfg       *Config
		stateSrvc *state.Service
	}
	tests := []struct {
		name    string
		args    args
		want    *network.Service
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createNetworkService(tt.args.cfg, tt.args.stateSrvc)
			if (err != nil) != tt.wantErr {
				t.Errorf("createNetworkService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createNetworkService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createRPCService(t *testing.T) {
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
		name    string
		args    args
		want    *rpc.HTTPServer
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createRPCService(tt.args.cfg, tt.args.ns, tt.args.stateSrvc, tt.args.coreSrvc, tt.args.networkSrvc, tt.args.bp, tt.args.sysSrvc, tt.args.finSrvc)
			if (err != nil) != tt.wantErr {
				t.Errorf("createRPCService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createRPCService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createRuntime(t *testing.T) {
	type args struct {
		cfg  *Config
		ns   runtime.NodeStorage
		st   *state.Service
		ks   *keystore.GlobalKeystore
		net  *network.Service
		code []byte
	}
	tests := []struct {
		name    string
		args    args
		want    runtime.Instance
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createRuntime(tt.args.cfg, tt.args.ns, tt.args.st, tt.args.ks, tt.args.net, tt.args.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("createRuntime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createRuntime() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createRuntimeStorage(t *testing.T) {
	type args struct {
		st *state.Service
	}
	tests := []struct {
		name    string
		args    args
		want    *runtime.NodeStorage
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createRuntimeStorage(tt.args.st)
			if (err != nil) != tt.wantErr {
				t.Errorf("createRuntimeStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createRuntimeStorage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createStateService(t *testing.T) {
	type args struct {
		cfg *Config
	}
	tests := []struct {
		name    string
		args    args
		want    *state.Service
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createStateService(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("createStateService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createStateService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createSystemService(t *testing.T) {
	type args struct {
		cfg       *types.SystemInfo
		stateSrvc *state.Service
	}
	tests := []struct {
		name    string
		args    args
		want    *system.Service
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createSystemService(tt.args.cfg, tt.args.stateSrvc)
			if (err != nil) != tt.wantErr {
				t.Errorf("createSystemService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createSystemService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newInMemoryDB(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    chaindb.Database
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newInMemoryDB(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("newInMemoryDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newInMemoryDB() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newSyncService(t *testing.T) {
	type args struct {
		cfg      *Config
		st       *state.Service
		fg       sync.FinalityGadget
		verifier *babe.VerificationManager
		cs       *core.Service
		net      *network.Service
	}
	tests := []struct {
		name    string
		args    args
		want    *sync.Service
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newSyncService(tt.args.cfg, tt.args.st, tt.args.fg, tt.args.verifier, tt.args.cs, tt.args.net)
			if (err != nil) != tt.wantErr {
				t.Errorf("newSyncService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newSyncService() got = %v, want %v", got, tt.want)
			}
		})
	}
}
