---
layout: default
title: Configuration
permalink: /usage/configuration/
---

# Configuration

Gossamer consumes a `.toml` file containing predefined settings for the node from setting the chain-spec file, to the RPC/WS server, this file allows you to curate the functionality of the node instead of writing out the flags manually

## Full reference

```toml
# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

# NOTE: Any path below can be absolute (e.g. "/var/gossamer/data") or
# relative to the home directory (e.g. "data"). The home directory is
# "$HOME/.local/share/gossamer" by default, but could be changed via
# $GSSMRHOME env variable or --home cmd flag.

#######################################################################
###                   Main Base Config Options                      ###
#######################################################################

# Name of the node
# Defaults to "Gossamer"
name = "Westend"

# Identifier of the node
# Defaults to a random value
id = "westend_dev"

# Path to the working directory of the node
# Defaults to "$HOME/.local/share/gossamer/<CHAIN>"
base-path = "/Users/user/.local/share/gossamer/alice"

# Path to the chain-spec raw JSON file
chain-spec = "/Users/user/.local/share/gossamer/alice/chain-spec.json"

# Global log level
# One of: crit, error, warn, info, debug, trace
# Defaults to "info"
log-level = "info"

# Listen address for the prometheus server
# Defaults to "localhost:9876"
prometheus-port = 9876

# Retain number of block from latest block while pruning
# Defaults to 512
retain-blocks = 512

# State trie online pruning mode
# Defaults to "archive"
pruning = "archive"

# Disable connecting to the Substrate telemetry server
# Defaults to false
no-telemetry = false

# List of telemetry server URLs to connect to
# Format for each entry:
# [[telemetry-urls]]
# endpoint = "wss://telemetry.polkadot.io/submit/"
# verbosity = 0


# Publish metrics to prometheus
# Defaults to false
prometheus-external = false

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
core = "info"

# Digest module log level
digest = "info"

# Sync module log level
sync = "info"

# Network module log level
network = "info"

# RPC module log level
rpc = "info"

# State module log level
state = "info"

# Runtime module log level
runtime = "info"

# BABE module log level
babe = "info"

# GRANDPA module log level
grandpa = "info"

# WASM module log level
wasmer = "info"


#######################################################
###          Account Configuration Options          ###
#######################################################
[account]

# Keyring to use for the node
key = "alice"

# Unlock an account. eg. --unlock=0 to unlock account 0
unlock = ""

#######################################################
###          Network Configuration Options          ###
#######################################################
[network]

# Network port to use
# Defaults to 7001
port = 7001

# Comma separated node URLs for network discovery bootstrap
bootnodes = ""

# Protocol ID to use
protocol-id = "dot"

# Disables network bootstrapping (mDNS still enabled)
# Defaults to false
no-bootstrap = true

# Disables network mDNS discovery
# Defaults to false
no-mdns = true

# Minimum number of peers to connect to
# Defaults to 25
min-peers = 0

# Maximum number of peers to connect to
# Defaults to 50
max-peers = 0

# Comma separated list of peers to always keep connected to
persistent-peers = ""

# Interval to perform peer discovery in duration
# Format: "10s", "1m", "1h"
discovery-interval = "1s"

# Overrides the public IP address used for peer to peer networking"
public-ip = ""

# Overrides the public DNS used for peer to peer networking"
public-dns = ""

# Overrides the secret Ed25519 key to use for libp2p networking
node-key = ""

# Multiaddress to listen on
listen-addr = ""

#######################################################
###             Core Configuration Options          ###
#######################################################
[core]

# Role of the gossamer node
# Represented as an integer
# One of: 1 (Full), 2 (Light), 4 (Authority)
role = 1

# Enable BABE authoring
# Defaults to true
babe-authority = true

# Enable GRANDPA authoring
# Defaults to true
grandpa-authority = true

# WASM interpreter
# Defaults to "wasmer"
wasm-interpreter = "wasmer"

# Grandpa interval
grandpa-interval = "1s"

#######################################################
###            State Configuration Options          ###
#######################################################
[state]
# Rewind head of chain to the given block number
# Defaults to 0
rewind = 0

#######################################################
###              RPC Configuration Options          ###
#######################################################
[rpc]

# Enable external HTTP-RPC connections
# Defaults to false
rpc-external = false

# Enable unsafe RPC methods
# Defaults to false
unsafe-rpc = false

# Enable external HTTP-RPC connections to unsafe procedures
# Defaults to false
unsafe-rpc-external = false

# HTTP-RPC server listening port
# Defaults to 8545
port = 8545

# HTTP-RPC server listening hostname
# Defaults to "localhost"
host = "localhost"

# API modules to enable via HTTP-RPC, comma separated list
# Defaults to "system, author, chain, state, rpc, grandpa, offchain, childstate, syncstate, payment"
modules = ["system", "author", "chain", "state", "rpc", "grandpa", "offchain", "childstate", "syncstate", "payment", ]

# Websockets server listening port
# Defaults to 8546
ws-port = 8546

# Enable external websocket connections
# Defaults to false
ws-external = false

# Enable external websocket connections to unsafe procedures
# Defaults to false
unsafe-ws-external = false

#######################################################
###            PPROF Configuration Options          ###
#######################################################
[pprof]

# Enable the pprof server
# Defaults to false
enabled = false

# Pprof server listening address
# Defaults to "localhost:6060"
listening-address = "localhost:6060"

# The frequency at which the Go runtime samples the state of goroutines to generate block profile information.
# Defaults to 0
block-profile-rate = 0

# The frequency at which the Go runtime samples the state of mutexes to generate mutex profile information.
# Defaults to 0
mutex-profile-rate = 0

```
