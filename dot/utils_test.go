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
	"os"
	"reflect"
	"testing"

	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/lib/genesis"
)

func TestCreateJSONRawFile(t *testing.T) {
	type args struct {
		bs *BuildSpec
		fp string
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateJSONRawFile(tt.args.bs, tt.args.fp); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateJSONRawFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExportConfig(t *testing.T) {
	type args struct {
		cfg *Config
		fp  string
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExportConfig(tt.args.cfg, tt.args.fp); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExportConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExportTomlConfig(t *testing.T) {
	type args struct {
		cfg *ctoml.Config
		fp  string
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExportTomlConfig(tt.args.cfg, tt.args.fp); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExportTomlConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTestConfig(t *testing.T) {
	type args struct {
		t *testing.T
	}
	tests := []struct {
		name string
		args args
		want *Config
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTestConfig(tt.args.t); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTestConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTestConfigWithFile(t *testing.T) {
	type args struct {
		t *testing.T
	}
	tests := []struct {
		name  string
		args  args
		want  *Config
		want1 *os.File
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := NewTestConfigWithFile(tt.args.t)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTestConfigWithFile() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("NewTestConfigWithFile() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestNewTestGenesis(t *testing.T) {
	type args struct {
		t *testing.T
	}
	tests := []struct {
		name string
		args args
		want *genesis.Genesis
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTestGenesis(tt.args.t); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTestGenesis() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTestGenesisAndRuntime(t *testing.T) {
	type args struct {
		t *testing.T
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTestGenesisAndRuntime(tt.args.t); got != tt.want {
				t.Errorf("NewTestGenesisAndRuntime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTestGenesisFile(t *testing.T) {
	type args struct {
		t   *testing.T
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTestGenesisFile(tt.args.t, tt.args.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTestGenesisFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTestGenesisRawFile(t *testing.T) {
	type args struct {
		t   *testing.T
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTestGenesisRawFile(tt.args.t, tt.args.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTestGenesisRawFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRandomNodeName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RandomNodeName(); got != tt.want {
				t.Errorf("RandomNodeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteConfig(t *testing.T) {
	type args struct {
		data []byte
		fp   string
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WriteConfig(tt.args.data, tt.args.fp); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WriteConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_setupLogger(t *testing.T) {
	type args struct {
		cfg *Config
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
