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

/****** these are for production settings ***********/

// EnsureRoot creates the root, config, and data directories if they don't exist,
// and panics if it fails.
//func EnsureRoot(rootDir string) {
//	if err := os.EnsureDir(rootDir, DefaultDirPerm); err != nil {
//		panic(err.Error())
//	}
//	if err := os.EnsureDir(filepath.Join(rootDir, defaultConfigDir), DefaultDirPerm); err != nil {
//		panic(err.Error())
//	}
//
//	configFilePath := filepath.Join(rootDir, defaultConfigFilePath)
//
//	// Write default config file if missing.
//	if !os.FileExists(configFilePath) {
//		writeDefaultConfigFile(configFilePath)
//	}
//}

//func writeDefaultConfigFile(configFilePath string) {
//	WriteConfigFile(configFilePath, DefaultWestendDevConfig())
//}

// WriteConfigFile renders config using the template and writes it to configFilePath.
func WriteConfigFile(configFilePath string, config *Config) error {
	var buffer bytes.Buffer

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

name = "{{ .BaseConfig.Name }}"

id = "{{ .BaseConfig.ID }}"

base-path = "{{ .BaseConfig.BasePath }}"

genesis = "{{ .BaseConfig.Genesis }}"

log-level = "{{ .BaseConfig.LogLevel }}"

metrics-address = "{{ .BaseConfig.MetricsAddress }}"

retain-blocks = {{ .BaseConfig.RetainBlocks }}

pruning = "{{ .BaseConfig.Pruning }}"

no-telemetry = {{ .BaseConfig.NoTelemetry }}

{{range .BaseConfig.TelemetryURLs}} 
[[telemetry-urls]]
endpoint = "{{ .Endpoint }}"
verbosity = {{ .Verbosity }}
{{end}}

publish-metrics = {{ .BaseConfig.PublishMetrics }}

#######################################################################
###                 Advanced Configuration Options                  ###
#######################################################################

#######################################################
###              Log Configuration Options          ###
#######################################################
[log]

# One of: crit, error, warn, info, debug, trace

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

# Account key
key = "{{ .Account.Key }}"

# Account unlock
unlock = "{{ .Account.Unlock }}"

#######################################################
###          Network Configuration Options          ###
#######################################################
[network]

port = {{ .Network.Port }}

bootnodes = [{{ StringsJoin .Network.Bootnodes ", " }}]

protocol-id = "{{ .Network.ProtocolID }}"

no-bootstrap = {{ .Network.NoBootstrap }}

no-mdns = {{ .Network.NoMDNS }}

min-peers = {{ .Network.MinPeers }}

max-peers = {{ .Network.MaxPeers }}

persistent-peers = [{{ StringsJoin .Network.PersistentPeers ", " }}]

discovery-interval = "{{ .Network.DiscoveryInterval }}"

public-ip = "{{ .Network.PublicIP }}"

public-dns = "{{ .Network.PublicDNS }}"

node-key = "{{ .Network.NodeKey }}"

listen-address = "{{ .Network.ListenAddress }}"

#######################################################
###             Core Configuration Options          ###
#######################################################
[core]

role = {{ .Core.Role }}

babe-authority = {{ .Core.BabeAuthority }}

grandpa-authority = {{ .Core.GrandpaAuthority }}

slot-duration = {{ .Core.SlotDuration }}

epoch-length = {{ .Core.EpochLength }}

wasm-interpreter = "{{ .Core.WasmInterpreter }}"

grandpa-interval = "{{ .Core.GrandpaInterval }}"

babe-lead = {{ .Core.BABELead }}

#######################################################
###            State Configuration Options          ###
#######################################################
[state]

rewind = {{ .State.Rewind }}

#######################################################
###              RPC Configuration Options          ###
#######################################################
[rpc]

enabled = {{ .RPC.Enabled }}

unsafe = {{ .RPC.Unsafe }}

unsafe-external = {{ .RPC.UnsafeExternal }}

external = {{ .RPC.External }}

port = {{ .RPC.Port }}

host = "{{ .RPC.Host }}"

modules = [{{ range .RPC.Modules }}"{{ . }}", {{ end }}]

ws = {{ .RPC.WS }}

ws-port = {{ .RPC.WSPort }}

ws-external = {{ .RPC.WSExternal }}

ws-unsafe = {{ .RPC.WSUnsafe }}

ws-unsafe-external = {{ .RPC.WSUnsafeExternal }}

#######################################################
###            PPROF Configuration Options          ###
#######################################################
[pprof]

enabled = {{ .Pprof.Enabled }}

listening-address = "{{ .Pprof.ListeningAddress }}"

block-profile-rate = {{ .Pprof.BlockProfileRate }}

mutex-profile-rate = {{ .Pprof.MutexProfileRate }}
`
