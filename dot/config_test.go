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
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/assert"
)

func TestConfig_String(t *testing.T) {
	t.Parallel()

	type fields struct {
		Global  GlobalConfig
		Log     LogConfig
		Init    InitConfig
		Account AccountConfig
		Core    CoreConfig
		Network NetworkConfig
		RPC     RPCConfig
		System  types.SystemInfo
		State   StateConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "default values",
			fields: fields{},
			want:   "{\n\t\"Global\": {\n\t\t\"Name\": \"\",\n\t\t\"ID\": \"\",\n\t\t\"BasePath\": \"\",\n\t\t\"LogLvl\": 0,\n\t\t\"PublishMetrics\": false,\n\t\t\"MetricsPort\": 0,\n\t\t\"NoTelemetry\": false,\n\t\t\"TelemetryURLs\": null,\n\t\t\"RetainBlocks\": 0,\n\t\t\"Pruning\": \"\"\n\t},\n\t\"Log\": {\n\t\t\"CoreLvl\": 0,\n\t\t\"SyncLvl\": 0,\n\t\t\"NetworkLvl\": 0,\n\t\t\"RPCLvl\": 0,\n\t\t\"StateLvl\": 0,\n\t\t\"RuntimeLvl\": 0,\n\t\t\"BlockProducerLvl\": 0,\n\t\t\"FinalityGadgetLvl\": 0\n\t},\n\t\"Init\": {\n\t\t\"Genesis\": \"\"\n\t},\n\t\"Account\": {\n\t\t\"Key\": \"\",\n\t\t\"Unlock\": \"\"\n\t},\n\t\"Core\": {\n\t\t\"Roles\": 0,\n\t\t\"BabeAuthority\": false,\n\t\t\"BABELead\": false,\n\t\t\"GrandpaAuthority\": false,\n\t\t\"WasmInterpreter\": \"\",\n\t\t\"GrandpaInterval\": 0\n\t},\n\t\"Network\": {\n\t\t\"Port\": 0,\n\t\t\"Bootnodes\": null,\n\t\t\"ProtocolID\": \"\",\n\t\t\"NoBootstrap\": false,\n\t\t\"NoMDNS\": false,\n\t\t\"MinPeers\": 0,\n\t\t\"MaxPeers\": 0,\n\t\t\"PersistentPeers\": null,\n\t\t\"DiscoveryInterval\": 0\n\t},\n\t\"RPC\": {\n\t\t\"Enabled\": false,\n\t\t\"External\": false,\n\t\t\"Unsafe\": false,\n\t\t\"UnsafeExternal\": false,\n\t\t\"Port\": 0,\n\t\t\"Host\": \"\",\n\t\t\"Modules\": null,\n\t\t\"WSPort\": 0,\n\t\t\"WS\": false,\n\t\t\"WSExternal\": false,\n\t\t\"WSUnsafe\": false,\n\t\t\"WSUnsafeExternal\": false\n\t},\n\t\"System\": {\n\t\t\"SystemName\": \"\",\n\t\t\"SystemVersion\": \"\"\n\t},\n\t\"State\": {\n\t\t\"Rewind\": 0\n\t}\n}",
		},
		{
			name: "global config",
			fields: fields{
				Global: GlobalConfig{
					Name:           "name",
					ID:             "id",
					BasePath:       "basepath",
					LogLvl:         2,
					PublishMetrics: true,
					MetricsPort:    3,
					NoTelemetry:    true,
					TelemetryURLs: []genesis.TelemetryEndpoint{{
						Endpoint:  "endpoint",
						Verbosity: 4,
					}},
					RetainBlocks: 5,
					Pruning:      "a",
				},
			},
			want: "{\n\t\"Global\": {\n\t\t\"Name\": \"name\",\n\t\t\"ID\": \"id\",\n\t\t\"BasePath\": \"basepath\",\n\t\t\"LogLvl\": 2,\n\t\t\"PublishMetrics\": true,\n\t\t\"MetricsPort\": 3,\n\t\t\"NoTelemetry\": true,\n\t\t\"TelemetryURLs\": [\n\t\t\t{\n\t\t\t\t\"Endpoint\": \"endpoint\",\n\t\t\t\t\"Verbosity\": 4\n\t\t\t}\n\t\t],\n\t\t\"RetainBlocks\": 5,\n\t\t\"Pruning\": \"a\"\n\t},\n\t\"Log\": {\n\t\t\"CoreLvl\": 0,\n\t\t\"SyncLvl\": 0,\n\t\t\"NetworkLvl\": 0,\n\t\t\"RPCLvl\": 0,\n\t\t\"StateLvl\": 0,\n\t\t\"RuntimeLvl\": 0,\n\t\t\"BlockProducerLvl\": 0,\n\t\t\"FinalityGadgetLvl\": 0\n\t},\n\t\"Init\": {\n\t\t\"Genesis\": \"\"\n\t},\n\t\"Account\": {\n\t\t\"Key\": \"\",\n\t\t\"Unlock\": \"\"\n\t},\n\t\"Core\": {\n\t\t\"Roles\": 0,\n\t\t\"BabeAuthority\": false,\n\t\t\"BABELead\": false,\n\t\t\"GrandpaAuthority\": false,\n\t\t\"WasmInterpreter\": \"\",\n\t\t\"GrandpaInterval\": 0\n\t},\n\t\"Network\": {\n\t\t\"Port\": 0,\n\t\t\"Bootnodes\": null,\n\t\t\"ProtocolID\": \"\",\n\t\t\"NoBootstrap\": false,\n\t\t\"NoMDNS\": false,\n\t\t\"MinPeers\": 0,\n\t\t\"MaxPeers\": 0,\n\t\t\"PersistentPeers\": null,\n\t\t\"DiscoveryInterval\": 0\n\t},\n\t\"RPC\": {\n\t\t\"Enabled\": false,\n\t\t\"External\": false,\n\t\t\"Unsafe\": false,\n\t\t\"UnsafeExternal\": false,\n\t\t\"Port\": 0,\n\t\t\"Host\": \"\",\n\t\t\"Modules\": null,\n\t\t\"WSPort\": 0,\n\t\t\"WS\": false,\n\t\t\"WSExternal\": false,\n\t\t\"WSUnsafe\": false,\n\t\t\"WSUnsafeExternal\": false\n\t},\n\t\"System\": {\n\t\t\"SystemName\": \"\",\n\t\t\"SystemVersion\": \"\"\n\t},\n\t\"State\": {\n\t\t\"Rewind\": 0\n\t}\n}",
		},
		{
			name: "log config",
			fields: fields{
				Log: LogConfig{
					CoreLvl:           1,
					SyncLvl:           2,
					NetworkLvl:        3,
					RPCLvl:            4,
					StateLvl:          5,
					RuntimeLvl:        6,
					BlockProducerLvl:  7,
					FinalityGadgetLvl: 8,
				},
			},
			want: "{\n\t\"Global\": {\n\t\t\"Name\": \"\",\n\t\t\"ID\": \"\",\n\t\t\"BasePath\": \"\",\n\t\t\"LogLvl\": 0,\n\t\t\"PublishMetrics\": false,\n\t\t\"MetricsPort\": 0,\n\t\t\"NoTelemetry\": false,\n\t\t\"TelemetryURLs\": null,\n\t\t\"RetainBlocks\": 0,\n\t\t\"Pruning\": \"\"\n\t},\n\t\"Log\": {\n\t\t\"CoreLvl\": 1,\n\t\t\"SyncLvl\": 2,\n\t\t\"NetworkLvl\": 3,\n\t\t\"RPCLvl\": 4,\n\t\t\"StateLvl\": 5,\n\t\t\"RuntimeLvl\": 6,\n\t\t\"BlockProducerLvl\": 7,\n\t\t\"FinalityGadgetLvl\": 8\n\t},\n\t\"Init\": {\n\t\t\"Genesis\": \"\"\n\t},\n\t\"Account\": {\n\t\t\"Key\": \"\",\n\t\t\"Unlock\": \"\"\n\t},\n\t\"Core\": {\n\t\t\"Roles\": 0,\n\t\t\"BabeAuthority\": false,\n\t\t\"BABELead\": false,\n\t\t\"GrandpaAuthority\": false,\n\t\t\"WasmInterpreter\": \"\",\n\t\t\"GrandpaInterval\": 0\n\t},\n\t\"Network\": {\n\t\t\"Port\": 0,\n\t\t\"Bootnodes\": null,\n\t\t\"ProtocolID\": \"\",\n\t\t\"NoBootstrap\": false,\n\t\t\"NoMDNS\": false,\n\t\t\"MinPeers\": 0,\n\t\t\"MaxPeers\": 0,\n\t\t\"PersistentPeers\": null,\n\t\t\"DiscoveryInterval\": 0\n\t},\n\t\"RPC\": {\n\t\t\"Enabled\": false,\n\t\t\"External\": false,\n\t\t\"Unsafe\": false,\n\t\t\"UnsafeExternal\": false,\n\t\t\"Port\": 0,\n\t\t\"Host\": \"\",\n\t\t\"Modules\": null,\n\t\t\"WSPort\": 0,\n\t\t\"WS\": false,\n\t\t\"WSExternal\": false,\n\t\t\"WSUnsafe\": false,\n\t\t\"WSUnsafeExternal\": false\n\t},\n\t\"System\": {\n\t\t\"SystemName\": \"\",\n\t\t\"SystemVersion\": \"\"\n\t},\n\t\"State\": {\n\t\t\"Rewind\": 0\n\t}\n}",
		},
		{
			name: "init config",
			fields: fields{
				Init: InitConfig{Genesis: "genesis"},
			},
			want: "{\n\t\"Global\": {\n\t\t\"Name\": \"\",\n\t\t\"ID\": \"\",\n\t\t\"BasePath\": \"\",\n\t\t\"LogLvl\": 0,\n\t\t\"PublishMetrics\": false,\n\t\t\"MetricsPort\": 0,\n\t\t\"NoTelemetry\": false,\n\t\t\"TelemetryURLs\": null,\n\t\t\"RetainBlocks\": 0,\n\t\t\"Pruning\": \"\"\n\t},\n\t\"Log\": {\n\t\t\"CoreLvl\": 0,\n\t\t\"SyncLvl\": 0,\n\t\t\"NetworkLvl\": 0,\n\t\t\"RPCLvl\": 0,\n\t\t\"StateLvl\": 0,\n\t\t\"RuntimeLvl\": 0,\n\t\t\"BlockProducerLvl\": 0,\n\t\t\"FinalityGadgetLvl\": 0\n\t},\n\t\"Init\": {\n\t\t\"Genesis\": \"genesis\"\n\t},\n\t\"Account\": {\n\t\t\"Key\": \"\",\n\t\t\"Unlock\": \"\"\n\t},\n\t\"Core\": {\n\t\t\"Roles\": 0,\n\t\t\"BabeAuthority\": false,\n\t\t\"BABELead\": false,\n\t\t\"GrandpaAuthority\": false,\n\t\t\"WasmInterpreter\": \"\",\n\t\t\"GrandpaInterval\": 0\n\t},\n\t\"Network\": {\n\t\t\"Port\": 0,\n\t\t\"Bootnodes\": null,\n\t\t\"ProtocolID\": \"\",\n\t\t\"NoBootstrap\": false,\n\t\t\"NoMDNS\": false,\n\t\t\"MinPeers\": 0,\n\t\t\"MaxPeers\": 0,\n\t\t\"PersistentPeers\": null,\n\t\t\"DiscoveryInterval\": 0\n\t},\n\t\"RPC\": {\n\t\t\"Enabled\": false,\n\t\t\"External\": false,\n\t\t\"Unsafe\": false,\n\t\t\"UnsafeExternal\": false,\n\t\t\"Port\": 0,\n\t\t\"Host\": \"\",\n\t\t\"Modules\": null,\n\t\t\"WSPort\": 0,\n\t\t\"WS\": false,\n\t\t\"WSExternal\": false,\n\t\t\"WSUnsafe\": false,\n\t\t\"WSUnsafeExternal\": false\n\t},\n\t\"System\": {\n\t\t\"SystemName\": \"\",\n\t\t\"SystemVersion\": \"\"\n\t},\n\t\"State\": {\n\t\t\"Rewind\": 0\n\t}\n}",
		},
		{
			name: "account config",
			fields: fields{
				Account: AccountConfig{
					Key:    "aKey",
					Unlock: "unlock",
				},
			},
			want: "{\n\t\"Global\": {\n\t\t\"Name\": \"\",\n\t\t\"ID\": \"\",\n\t\t\"BasePath\": \"\",\n\t\t\"LogLvl\": 0,\n\t\t\"PublishMetrics\": false,\n\t\t\"MetricsPort\": 0,\n\t\t\"NoTelemetry\": false,\n\t\t\"TelemetryURLs\": null,\n\t\t\"RetainBlocks\": 0,\n\t\t\"Pruning\": \"\"\n\t},\n\t\"Log\": {\n\t\t\"CoreLvl\": 0,\n\t\t\"SyncLvl\": 0,\n\t\t\"NetworkLvl\": 0,\n\t\t\"RPCLvl\": 0,\n\t\t\"StateLvl\": 0,\n\t\t\"RuntimeLvl\": 0,\n\t\t\"BlockProducerLvl\": 0,\n\t\t\"FinalityGadgetLvl\": 0\n\t},\n\t\"Init\": {\n\t\t\"Genesis\": \"\"\n\t},\n\t\"Account\": {\n\t\t\"Key\": \"aKey\",\n\t\t\"Unlock\": \"unlock\"\n\t},\n\t\"Core\": {\n\t\t\"Roles\": 0,\n\t\t\"BabeAuthority\": false,\n\t\t\"BABELead\": false,\n\t\t\"GrandpaAuthority\": false,\n\t\t\"WasmInterpreter\": \"\",\n\t\t\"GrandpaInterval\": 0\n\t},\n\t\"Network\": {\n\t\t\"Port\": 0,\n\t\t\"Bootnodes\": null,\n\t\t\"ProtocolID\": \"\",\n\t\t\"NoBootstrap\": false,\n\t\t\"NoMDNS\": false,\n\t\t\"MinPeers\": 0,\n\t\t\"MaxPeers\": 0,\n\t\t\"PersistentPeers\": null,\n\t\t\"DiscoveryInterval\": 0\n\t},\n\t\"RPC\": {\n\t\t\"Enabled\": false,\n\t\t\"External\": false,\n\t\t\"Unsafe\": false,\n\t\t\"UnsafeExternal\": false,\n\t\t\"Port\": 0,\n\t\t\"Host\": \"\",\n\t\t\"Modules\": null,\n\t\t\"WSPort\": 0,\n\t\t\"WS\": false,\n\t\t\"WSExternal\": false,\n\t\t\"WSUnsafe\": false,\n\t\t\"WSUnsafeExternal\": false\n\t},\n\t\"System\": {\n\t\t\"SystemName\": \"\",\n\t\t\"SystemVersion\": \"\"\n\t},\n\t\"State\": {\n\t\t\"Rewind\": 0\n\t}\n}",
		},
		{
			name: "core config",
			fields: fields{
				Core: CoreConfig{
					Roles:            1,
					BabeAuthority:    true,
					BABELead:         true,
					GrandpaAuthority: true,
					WasmInterpreter:  "wasm",
					GrandpaInterval:  2,
				},
			},
			want: "{\n\t\"Global\": {\n\t\t\"Name\": \"\",\n\t\t\"ID\": \"\",\n\t\t\"BasePath\": \"\",\n\t\t\"LogLvl\": 0,\n\t\t\"PublishMetrics\": false,\n\t\t\"MetricsPort\": 0,\n\t\t\"NoTelemetry\": false,\n\t\t\"TelemetryURLs\": null,\n\t\t\"RetainBlocks\": 0,\n\t\t\"Pruning\": \"\"\n\t},\n\t\"Log\": {\n\t\t\"CoreLvl\": 0,\n\t\t\"SyncLvl\": 0,\n\t\t\"NetworkLvl\": 0,\n\t\t\"RPCLvl\": 0,\n\t\t\"StateLvl\": 0,\n\t\t\"RuntimeLvl\": 0,\n\t\t\"BlockProducerLvl\": 0,\n\t\t\"FinalityGadgetLvl\": 0\n\t},\n\t\"Init\": {\n\t\t\"Genesis\": \"\"\n\t},\n\t\"Account\": {\n\t\t\"Key\": \"\",\n\t\t\"Unlock\": \"\"\n\t},\n\t\"Core\": {\n\t\t\"Roles\": 1,\n\t\t\"BabeAuthority\": true,\n\t\t\"BABELead\": true,\n\t\t\"GrandpaAuthority\": true,\n\t\t\"WasmInterpreter\": \"wasm\",\n\t\t\"GrandpaInterval\": 2\n\t},\n\t\"Network\": {\n\t\t\"Port\": 0,\n\t\t\"Bootnodes\": null,\n\t\t\"ProtocolID\": \"\",\n\t\t\"NoBootstrap\": false,\n\t\t\"NoMDNS\": false,\n\t\t\"MinPeers\": 0,\n\t\t\"MaxPeers\": 0,\n\t\t\"PersistentPeers\": null,\n\t\t\"DiscoveryInterval\": 0\n\t},\n\t\"RPC\": {\n\t\t\"Enabled\": false,\n\t\t\"External\": false,\n\t\t\"Unsafe\": false,\n\t\t\"UnsafeExternal\": false,\n\t\t\"Port\": 0,\n\t\t\"Host\": \"\",\n\t\t\"Modules\": null,\n\t\t\"WSPort\": 0,\n\t\t\"WS\": false,\n\t\t\"WSExternal\": false,\n\t\t\"WSUnsafe\": false,\n\t\t\"WSUnsafeExternal\": false\n\t},\n\t\"System\": {\n\t\t\"SystemName\": \"\",\n\t\t\"SystemVersion\": \"\"\n\t},\n\t\"State\": {\n\t\t\"Rewind\": 0\n\t}\n}",
		},
		{
			name: "network config",
			fields: fields{
				Network: NetworkConfig{
					Port:              1,
					Bootnodes:         []string{"node1", "node2"},
					ProtocolID:        "pID",
					NoBootstrap:       true,
					NoMDNS:            true,
					MinPeers:          2,
					MaxPeers:          3,
					PersistentPeers:   []string{"peer1", "peer2"},
					DiscoveryInterval: 4,
				},
			},
			want: "{\n\t\"Global\": {\n\t\t\"Name\": \"\",\n\t\t\"ID\": \"\",\n\t\t\"BasePath\": \"\",\n\t\t\"LogLvl\": 0,\n\t\t\"PublishMetrics\": false,\n\t\t\"MetricsPort\": 0,\n\t\t\"NoTelemetry\": false,\n\t\t\"TelemetryURLs\": null,\n\t\t\"RetainBlocks\": 0,\n\t\t\"Pruning\": \"\"\n\t},\n\t\"Log\": {\n\t\t\"CoreLvl\": 0,\n\t\t\"SyncLvl\": 0,\n\t\t\"NetworkLvl\": 0,\n\t\t\"RPCLvl\": 0,\n\t\t\"StateLvl\": 0,\n\t\t\"RuntimeLvl\": 0,\n\t\t\"BlockProducerLvl\": 0,\n\t\t\"FinalityGadgetLvl\": 0\n\t},\n\t\"Init\": {\n\t\t\"Genesis\": \"\"\n\t},\n\t\"Account\": {\n\t\t\"Key\": \"\",\n\t\t\"Unlock\": \"\"\n\t},\n\t\"Core\": {\n\t\t\"Roles\": 0,\n\t\t\"BabeAuthority\": false,\n\t\t\"BABELead\": false,\n\t\t\"GrandpaAuthority\": false,\n\t\t\"WasmInterpreter\": \"\",\n\t\t\"GrandpaInterval\": 0\n\t},\n\t\"Network\": {\n\t\t\"Port\": 1,\n\t\t\"Bootnodes\": [\n\t\t\t\"node1\",\n\t\t\t\"node2\"\n\t\t],\n\t\t\"ProtocolID\": \"pID\",\n\t\t\"NoBootstrap\": true,\n\t\t\"NoMDNS\": true,\n\t\t\"MinPeers\": 2,\n\t\t\"MaxPeers\": 3,\n\t\t\"PersistentPeers\": [\n\t\t\t\"peer1\",\n\t\t\t\"peer2\"\n\t\t],\n\t\t\"DiscoveryInterval\": 4\n\t},\n\t\"RPC\": {\n\t\t\"Enabled\": false,\n\t\t\"External\": false,\n\t\t\"Unsafe\": false,\n\t\t\"UnsafeExternal\": false,\n\t\t\"Port\": 0,\n\t\t\"Host\": \"\",\n\t\t\"Modules\": null,\n\t\t\"WSPort\": 0,\n\t\t\"WS\": false,\n\t\t\"WSExternal\": false,\n\t\t\"WSUnsafe\": false,\n\t\t\"WSUnsafeExternal\": false\n\t},\n\t\"System\": {\n\t\t\"SystemName\": \"\",\n\t\t\"SystemVersion\": \"\"\n\t},\n\t\"State\": {\n\t\t\"Rewind\": 0\n\t}\n}",
		},
		{
			name: "rpc config",
			fields: fields{
				RPC: RPCConfig{
					Enabled:          true,
					External:         true,
					Unsafe:           true,
					UnsafeExternal:   true,
					Port:             1,
					Host:             "host",
					Modules:          []string{"mod1", "mod2"},
					WSPort:           2,
					WS:               true,
					WSExternal:       true,
					WSUnsafe:         true,
					WSUnsafeExternal: true,
				},
			},
			want: "{\n\t\"Global\": {\n\t\t\"Name\": \"\",\n\t\t\"ID\": \"\",\n\t\t\"BasePath\": \"\",\n\t\t\"LogLvl\": 0,\n\t\t\"PublishMetrics\": false,\n\t\t\"MetricsPort\": 0,\n\t\t\"NoTelemetry\": false,\n\t\t\"TelemetryURLs\": null,\n\t\t\"RetainBlocks\": 0,\n\t\t\"Pruning\": \"\"\n\t},\n\t\"Log\": {\n\t\t\"CoreLvl\": 0,\n\t\t\"SyncLvl\": 0,\n\t\t\"NetworkLvl\": 0,\n\t\t\"RPCLvl\": 0,\n\t\t\"StateLvl\": 0,\n\t\t\"RuntimeLvl\": 0,\n\t\t\"BlockProducerLvl\": 0,\n\t\t\"FinalityGadgetLvl\": 0\n\t},\n\t\"Init\": {\n\t\t\"Genesis\": \"\"\n\t},\n\t\"Account\": {\n\t\t\"Key\": \"\",\n\t\t\"Unlock\": \"\"\n\t},\n\t\"Core\": {\n\t\t\"Roles\": 0,\n\t\t\"BabeAuthority\": false,\n\t\t\"BABELead\": false,\n\t\t\"GrandpaAuthority\": false,\n\t\t\"WasmInterpreter\": \"\",\n\t\t\"GrandpaInterval\": 0\n\t},\n\t\"Network\": {\n\t\t\"Port\": 0,\n\t\t\"Bootnodes\": null,\n\t\t\"ProtocolID\": \"\",\n\t\t\"NoBootstrap\": false,\n\t\t\"NoMDNS\": false,\n\t\t\"MinPeers\": 0,\n\t\t\"MaxPeers\": 0,\n\t\t\"PersistentPeers\": null,\n\t\t\"DiscoveryInterval\": 0\n\t},\n\t\"RPC\": {\n\t\t\"Enabled\": true,\n\t\t\"External\": true,\n\t\t\"Unsafe\": true,\n\t\t\"UnsafeExternal\": true,\n\t\t\"Port\": 1,\n\t\t\"Host\": \"host\",\n\t\t\"Modules\": [\n\t\t\t\"mod1\",\n\t\t\t\"mod2\"\n\t\t],\n\t\t\"WSPort\": 2,\n\t\t\"WS\": true,\n\t\t\"WSExternal\": true,\n\t\t\"WSUnsafe\": true,\n\t\t\"WSUnsafeExternal\": true\n\t},\n\t\"System\": {\n\t\t\"SystemName\": \"\",\n\t\t\"SystemVersion\": \"\"\n\t},\n\t\"State\": {\n\t\t\"Rewind\": 0\n\t}\n}",
		},
		{
			name: "system info",
			fields: fields{
				System: types.SystemInfo{
					SystemName:    "name",
					SystemVersion: "version",
				},
			},
			want: "{\n\t\"Global\": {\n\t\t\"Name\": \"\",\n\t\t\"ID\": \"\",\n\t\t\"BasePath\": \"\",\n\t\t\"LogLvl\": 0,\n\t\t\"PublishMetrics\": false,\n\t\t\"MetricsPort\": 0,\n\t\t\"NoTelemetry\": false,\n\t\t\"TelemetryURLs\": null,\n\t\t\"RetainBlocks\": 0,\n\t\t\"Pruning\": \"\"\n\t},\n\t\"Log\": {\n\t\t\"CoreLvl\": 0,\n\t\t\"SyncLvl\": 0,\n\t\t\"NetworkLvl\": 0,\n\t\t\"RPCLvl\": 0,\n\t\t\"StateLvl\": 0,\n\t\t\"RuntimeLvl\": 0,\n\t\t\"BlockProducerLvl\": 0,\n\t\t\"FinalityGadgetLvl\": 0\n\t},\n\t\"Init\": {\n\t\t\"Genesis\": \"\"\n\t},\n\t\"Account\": {\n\t\t\"Key\": \"\",\n\t\t\"Unlock\": \"\"\n\t},\n\t\"Core\": {\n\t\t\"Roles\": 0,\n\t\t\"BabeAuthority\": false,\n\t\t\"BABELead\": false,\n\t\t\"GrandpaAuthority\": false,\n\t\t\"WasmInterpreter\": \"\",\n\t\t\"GrandpaInterval\": 0\n\t},\n\t\"Network\": {\n\t\t\"Port\": 0,\n\t\t\"Bootnodes\": null,\n\t\t\"ProtocolID\": \"\",\n\t\t\"NoBootstrap\": false,\n\t\t\"NoMDNS\": false,\n\t\t\"MinPeers\": 0,\n\t\t\"MaxPeers\": 0,\n\t\t\"PersistentPeers\": null,\n\t\t\"DiscoveryInterval\": 0\n\t},\n\t\"RPC\": {\n\t\t\"Enabled\": false,\n\t\t\"External\": false,\n\t\t\"Unsafe\": false,\n\t\t\"UnsafeExternal\": false,\n\t\t\"Port\": 0,\n\t\t\"Host\": \"\",\n\t\t\"Modules\": null,\n\t\t\"WSPort\": 0,\n\t\t\"WS\": false,\n\t\t\"WSExternal\": false,\n\t\t\"WSUnsafe\": false,\n\t\t\"WSUnsafeExternal\": false\n\t},\n\t\"System\": {\n\t\t\"SystemName\": \"name\",\n\t\t\"SystemVersion\": \"version\"\n\t},\n\t\"State\": {\n\t\t\"Rewind\": 0\n\t}\n}",
		},
		{
			name: "state config",
			fields: fields{
				State: StateConfig{Rewind: 2},
			},
			want: "{\n\t\"Global\": {\n\t\t\"Name\": \"\",\n\t\t\"ID\": \"\",\n\t\t\"BasePath\": \"\",\n\t\t\"LogLvl\": 0,\n\t\t\"PublishMetrics\": false,\n\t\t\"MetricsPort\": 0,\n\t\t\"NoTelemetry\": false,\n\t\t\"TelemetryURLs\": null,\n\t\t\"RetainBlocks\": 0,\n\t\t\"Pruning\": \"\"\n\t},\n\t\"Log\": {\n\t\t\"CoreLvl\": 0,\n\t\t\"SyncLvl\": 0,\n\t\t\"NetworkLvl\": 0,\n\t\t\"RPCLvl\": 0,\n\t\t\"StateLvl\": 0,\n\t\t\"RuntimeLvl\": 0,\n\t\t\"BlockProducerLvl\": 0,\n\t\t\"FinalityGadgetLvl\": 0\n\t},\n\t\"Init\": {\n\t\t\"Genesis\": \"\"\n\t},\n\t\"Account\": {\n\t\t\"Key\": \"\",\n\t\t\"Unlock\": \"\"\n\t},\n\t\"Core\": {\n\t\t\"Roles\": 0,\n\t\t\"BabeAuthority\": false,\n\t\t\"BABELead\": false,\n\t\t\"GrandpaAuthority\": false,\n\t\t\"WasmInterpreter\": \"\",\n\t\t\"GrandpaInterval\": 0\n\t},\n\t\"Network\": {\n\t\t\"Port\": 0,\n\t\t\"Bootnodes\": null,\n\t\t\"ProtocolID\": \"\",\n\t\t\"NoBootstrap\": false,\n\t\t\"NoMDNS\": false,\n\t\t\"MinPeers\": 0,\n\t\t\"MaxPeers\": 0,\n\t\t\"PersistentPeers\": null,\n\t\t\"DiscoveryInterval\": 0\n\t},\n\t\"RPC\": {\n\t\t\"Enabled\": false,\n\t\t\"External\": false,\n\t\t\"Unsafe\": false,\n\t\t\"UnsafeExternal\": false,\n\t\t\"Port\": 0,\n\t\t\"Host\": \"\",\n\t\t\"Modules\": null,\n\t\t\"WSPort\": 0,\n\t\t\"WS\": false,\n\t\t\"WSExternal\": false,\n\t\t\"WSUnsafe\": false,\n\t\t\"WSUnsafeExternal\": false\n\t},\n\t\"System\": {\n\t\t\"SystemName\": \"\",\n\t\t\"SystemVersion\": \"\"\n\t},\n\t\"State\": {\n\t\t\"Rewind\": 2\n\t}\n}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Global:  tt.fields.Global,
				Log:     tt.fields.Log,
				Init:    tt.fields.Init,
				Account: tt.fields.Account,
				Core:    tt.fields.Core,
				Network: tt.fields.Network,
				RPC:     tt.fields.RPC,
				System:  tt.fields.System,
				State:   tt.fields.State,
			}
			got := c.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDevConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "dev default",
			want: &Config{
				Global: GlobalConfig{
					ID: "dev",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DevConfig()
			assert.Equal(t, tt.want.Global.ID, got.Global.ID)
		})
	}
}

func TestGssmrConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "gossamer default",
			want: &Config{
				Global: GlobalConfig{
					ID: "gssmr",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GssmrConfig()
			assert.Equal(t, tt.want.Global.ID, got.Global.ID)
		})
	}
}

func TestKusamaConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "kusama default",
			want: &Config{
				Global: GlobalConfig{
					ID: "ksmcc3",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KusamaConfig()
			assert.Equal(t, tt.want.Global.ID, got.Global.ID)
		})
	}
}

func TestPolkadotConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "polkadot default",
			want: &Config{
				Global: GlobalConfig{
					ID: "polkadot",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PolkadotConfig()
			assert.Equal(t, tt.want.Global.ID, got.Global.ID)
		})
	}
}

func TestRPCConfig_isRPCEnabled(t *testing.T) {
	t.Parallel()

	type fields struct {
		Enabled          bool
		External         bool
		Unsafe           bool
		UnsafeExternal   bool
		Port             uint32
		Host             string
		Modules          []string
		WSPort           uint32
		WS               bool
		WSExternal       bool
		WSUnsafe         bool
		WSUnsafeExternal bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "default",
			want: false,
		},
		{
			name:   "enabled true",
			fields: fields{Enabled: true},
			want:   true,
		},
		{
			name:   "external true",
			fields: fields{External: true},
			want:   true,
		},
		{
			name:   "unsafe true",
			fields: fields{Unsafe: true},
			want:   true,
		},
		{
			name:   "unsafe external true",
			fields: fields{UnsafeExternal: true},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RPCConfig{
				Enabled:          tt.fields.Enabled,
				External:         tt.fields.External,
				Unsafe:           tt.fields.Unsafe,
				UnsafeExternal:   tt.fields.UnsafeExternal,
				Port:             tt.fields.Port,
				Host:             tt.fields.Host,
				Modules:          tt.fields.Modules,
				WSPort:           tt.fields.WSPort,
				WS:               tt.fields.WS,
				WSExternal:       tt.fields.WSExternal,
				WSUnsafe:         tt.fields.WSUnsafe,
				WSUnsafeExternal: tt.fields.WSUnsafeExternal,
			}
			got := r.isRPCEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRPCConfig_isWSEnabled(t *testing.T) {
	t.Parallel()

	type fields struct {
		Enabled          bool
		External         bool
		Unsafe           bool
		UnsafeExternal   bool
		Port             uint32
		Host             string
		Modules          []string
		WSPort           uint32
		WS               bool
		WSExternal       bool
		WSUnsafe         bool
		WSUnsafeExternal bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "default",
			want: false,
		},
		{
			name:   "ws true",
			fields: fields{WS: true},
			want:   true,
		},
		{
			name:   "ws external true",
			fields: fields{WSExternal: true},
			want:   true,
		},
		{
			name:   "ws unsafe true",
			fields: fields{WSUnsafe: true},
			want:   true,
		},
		{
			name:   "ws unsafe external true",
			fields: fields{WSUnsafeExternal: true},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RPCConfig{
				Enabled:          tt.fields.Enabled,
				External:         tt.fields.External,
				Unsafe:           tt.fields.Unsafe,
				UnsafeExternal:   tt.fields.UnsafeExternal,
				Port:             tt.fields.Port,
				Host:             tt.fields.Host,
				Modules:          tt.fields.Modules,
				WSPort:           tt.fields.WSPort,
				WS:               tt.fields.WS,
				WSExternal:       tt.fields.WSExternal,
				WSUnsafe:         tt.fields.WSUnsafe,
				WSUnsafeExternal: tt.fields.WSUnsafeExternal,
			}
			got := r.isWSEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_networkServiceEnabled(t *testing.T) {
	t.Parallel()

	type args struct {
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "dev config",
			args: args{cfg: DevConfig()},
			want: true,
		},
		{
			name: "empty config",
			args: args{cfg: &Config{}},
			want: false,
		},
		{
			name: "core roles 0",
			args: args{cfg: &Config{
				Core: CoreConfig{
					Roles: 0,
				},
			}},
			want: false,
		},
		{
			name: "core roles 1",
			args: args{cfg: &Config{
				Core: CoreConfig{
					Roles: 1,
				},
			}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := networkServiceEnabled(tt.args.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}
