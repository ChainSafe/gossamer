// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	// DefaultDirPerm is the default directory permission for gossamer files
	DefaultDirPerm = 0o700
	// defaultConfigDir is the default directory for gossamer config files
	defaultConfigDir = "config"
	// defaultConfigFileName is the default name of the config file
	defaultConfigFileName = "config.toml"
)

var (
	defaultConfigFilePath = filepath.Join(defaultConfigDir, defaultConfigFileName)
)

var configTemplate *template.Template

func init() {
	var err error
	tmpl := template.New("configFileTemplate").Funcs(template.FuncMap{
		"StringsJoin": strings.Join,
	})
	if configTemplate, err = tmpl.Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

// WriteConfigFile writes the config to the base path.
func WriteConfigFile(basePath string, config *Config) error {
	var buffer bytes.Buffer
	configFilePath := filepath.Join(basePath, defaultConfigFilePath)
	if err := configTemplate.Execute(&buffer, config); err != nil {
		return fmt.Errorf("failed to render config template: %w", err)
	}

	return os.WriteFile(configFilePath, buffer.Bytes(), 0o600)
}

// Note: any changes to the comments/variables/mapstructure
// must be reflected in the appropriate struct in config/config.go
const defaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

# NOTE: Any path below can be absolute (e.g. "/var/gossamer/data") or
# relative to the home directory (e.g. "data"). The home directory is
# "$HOME/.gossamer" by default, but could be changed via $GSSMRHOME env variable
# or --home cmd flag.

#######################################################################
###                   Main Base Config Options                      ###
#######################################################################

# Name of the node
# Defaults to "Gossamer"
name = "{{ .BaseConfig.Name }}"

# Identifier of the node
# Defaults to a random value
id = "{{ .BaseConfig.ID }}"

# Path to the working directory of the node
# Defaults to "$HOME/.gossamer/<CHAIN>"
base-path = "{{ .BaseConfig.BasePath }}"

# Path to the chain-spec raw JSON file
chain-spec = "{{ .BaseConfig.ChainSpec }}"

# Global log level
# One of: crit, error, warn, info, debug, trace
# Defaults to "info"
log-level = "{{ .BaseConfig.LogLevel }}"

# Listen address for the prometheus server
# Defaults to "localhost:9876"
prometheus-port = {{ .BaseConfig.PrometheusPort }}

# Retain number of block from latest block while pruning
# Defaults to 512
retain-blocks = {{ .BaseConfig.RetainBlocks }}

# State trie online pruning mode
# Defaults to "archive"
pruning = "{{ .BaseConfig.Pruning }}"

# Disable connecting to the Substrate telemetry server
# Defaults to false
no-telemetry = {{ .BaseConfig.NoTelemetry }}

# List of telemetry server URLs to connect to
# Format for each entry:
# [[telemetry-urls]]
# endpoint = "wss://telemetry.polkadot.io/submit/"
# verbosity = 0
{{range .BaseConfig.TelemetryURLs}} 
[[telemetry-urls]]
endpoint = "{{ .Endpoint }}"
verbosity = {{ .Verbosity }}
{{end}}

# Publish metrics to prometheus
# Defaults to false
prometheus-external = {{ .BaseConfig.PrometheusExternal }}


#######################################################################
###                 Advanced Configuration Options                  ###
#######################################################################

#######################################################
###              Log Configuration Options          ###
#######################################################
[log]

# One of: crit, error, warn, info, debug, trace
# Defaults to "info"

# Core module log level
core = "{{ .Log.Core }}"

# Digest module log level
digest = "{{ .Log.Digest }}"

# Sync module log level
sync = "{{ .Log.Sync }}"

# Network module log level
network = "{{ .Log.Network }}"

# RPC module log level
rpc = "{{ .Log.RPC }}"

# State module log level
state = "{{ .Log.State }}"

# Runtime module log level
runtime = "{{ .Log.Runtime }}"

# BABE module log level
babe = "{{ .Log.Babe }}"

# GRANDPA module log level
grandpa = "{{ .Log.Grandpa }}"

# WASM module log level
wasmer = "{{ .Log.Wasmer }}"


#######################################################
###          Account Configuration Options          ###
#######################################################
[account]

# Keyring to use for the node
key = "{{ .Account.Key }}"

# Unlock an account. eg. --unlock=0 to unlock account 0
unlock = "{{ .Account.Unlock }}"

#######################################################
###          Network Configuration Options          ###
#######################################################
[network]

# Network port to use
# Defaults to 7001
port = {{ .Network.Port }}

# Comma separated node URLs for network discovery bootstrap
bootnodes = "{{ StringsJoin .Network.Bootnodes "," }}"

# Protocol ID to use
protocol-id = "{{ .Network.ProtocolID }}"

# Disables network bootstrapping (mDNS still enabled)
# Defaults to false
no-bootstrap = {{ .Network.NoBootstrap }}

# Disables network mDNS discovery
# Defaults to false
no-mdns = {{ .Network.NoMDNS }}

# Minimum number of peers to connect to
# Defaults to 25
min-peers = {{ .Network.MinPeers }}

# Maximum number of peers to connect to
# Defaults to 50
max-peers = {{ .Network.MaxPeers }}

# Comma separated list of peers to always keep connected to
persistent-peers = "{{ StringsJoin .Network.PersistentPeers ", " }}"

# Interval to perform peer discovery in duration
# Format: "10s", "1m", "1h"
discovery-interval = "{{ .Network.DiscoveryInterval }}"

# Overrides the public IP address used for peer to peer networking"
public-ip = "{{ .Network.PublicIP }}"

# Overrides the public DNS used for peer to peer networking"
public-dns = "{{ .Network.PublicDNS }}"

# Overrides the secret Ed25519 key to use for libp2p networking
node-key = "{{ .Network.NodeKey }}"

# Multiaddress to listen on
listen-addr = "{{ .Network.ListenAddress }}"

#######################################################
###             Core Configuration Options          ###
#######################################################
[core]

# Role of the gossamer node
# Represented as an integer
# One of: 1 (Full), 2 (Light), 4 (Authority)
role = {{ .Core.Role }}

# Enable BABE authoring
# Defaults to true
babe-authority = {{ .Core.BabeAuthority }}

# Enable GRANDPA authoring
# Defaults to true
grandpa-authority = {{ .Core.GrandpaAuthority }}

# WASM interpreter
# Defaults to "wasmer"
wasm-interpreter = "{{ .Core.WasmInterpreter }}"

# Grandpa interval
grandpa-interval = "{{ .Core.GrandpaInterval }}"

#######################################################
###            State Configuration Options          ###
#######################################################
[state]
# Rewind head of chain to the given block number
# Defaults to 0
rewind = {{ .State.Rewind }}

#######################################################
###              RPC Configuration Options          ###
#######################################################
[rpc]

# Enable external HTTP-RPC connections
# Defaults to false
rpc-external = {{ .RPC.RPCExternal }}

# Enable unsafe RPC methods
# Defaults to false
unsafe-rpc = {{ .RPC.UnsafeRPC }}

# Enable external HTTP-RPC connections to unsafe procedures
# Defaults to false
unsafe-rpc-external = {{ .RPC.UnsafeRPCExternal }}

# HTTP-RPC server listening port
# Defaults to 8545
port = {{ .RPC.Port }}

# HTTP-RPC server listening hostname
# Defaults to "localhost"
host = "{{ .RPC.Host }}"

# API modules to enable via HTTP-RPC, comma separated list
# Defaults to "system, author, chain, state, rpc, grandpa, offchain, childstate, syncstate, payment"
modules = [{{ range .RPC.Modules }}"{{ . }}", {{ end }}]

# Websockets server listening port
# Defaults to 8546
ws-port = {{ .RPC.WSPort }}

# Enable external websocket connections
# Defaults to false
ws-external = {{ .RPC.WSExternal }}

# Enable external websocket connections to unsafe procedures
# Defaults to false
unsafe-ws-external = {{ .RPC.UnsafeWSExternal }}

#######################################################
###            PPROF Configuration Options          ###
#######################################################
[pprof]

# Enable the pprof server
# Defaults to false
enabled = {{ .Pprof.Enabled }}

# Pprof server listening address
# Defaults to "localhost:6060"
listening-address = "{{ .Pprof.ListeningAddress }}"

# The frequency at which the Go runtime samples the state of goroutines to generate block profile information.
# Defaults to 0
block-profile-rate = {{ .Pprof.BlockProfileRate }}

# The frequency at which the Go runtime samples the state of mutexes to generate mutex profile information.
# Defaults to 0
mutex-profile-rate = {{ .Pprof.MutexProfileRate }}

#######################################################
###            Checkpoint Configuration Options     ###
#######################################################

[checkpoint]

# Enable database checkpoints
# Defaults to false
# Default path: 'base-path'/snapshot
# Default frequecy: 1 million blocks
enabled = {{ .Checkpoint.Enabled }}
path = {{ .Checkpoint.Path }}
frequency = {{ .Checkpoint.Frequency }}
`
