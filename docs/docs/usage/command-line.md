---
layout: default
title: Command-Line
permalink: /usage/command-line/
---

## Gossamer Command

The `gossamer` command is the root command for the `gossamer` package (`cmd/gossamer`). The root command starts the node (and initialises the node if the node has not already been initialised). 

### Flags

These are the flags that can be used with the `gossamer` command

```
--babe-authority  Enable BABE authorship
--base-path       Working directory for the node
--bootnodes       Comma separated enode URLs for network discovery bootstrap
--chain           chain-spec-raw.json used to load node configuration. It can also be a chain name (eg. kusama, polkadot, westend, westend-dev and westend-local)
--discovery-interval Interval between network discovery lookups (in duration format) 
--grandpa-authority Runs as a GRANDPA authority node
--grandpa-interval GRANDPA voting period in duration (default 10s)
--help help for gossamer
--id Identifier used to identify this node in the network
--key Key to use for the node
--listen-addr  Overrides the listen address used for peer to peer networking
--log:  Set a logging filter.
	    Syntax is a list of 'module=logLevel' (comma separated)
	    e.g. --log sync=debug,core=trace
	    Modules are global, core, digest, sync, network, rpc, state, runtime, babe, grandpa, wasmer.
	    Log levels (least to most verbose) are error, warn, info, debug, and trace.
	    By default, all modules log 'info'.
	    The global log level can be set with --log global=debug
--max-peers Maximum number of peers to connect to (default 50)
--min-peers Minimum number of peers to connect to (default 5)
--name Name of the node
--no-bootstrap Disables network bootstrapping (mdns still enabled)
--no-mdns Disables network mdns discovery
--no-telemetry Disables telemetry
--node-key Overrides the secret Ed25519 key to use for libp2p networking
--password Password used to encrypt the keystore
--persistent-peers Comma separated list of peers to always keep connected to
--port Network port to use (default 7001)
--pprof.block-profile-rate The frequency at which the Go runtime samples the state of goroutines to generate block profile information.
--pprof.enabled Enable the pprof profiler
--pprof.listening-address The address to listen on for pprof profiling
--pprof.mutex-profile-rate  The frequency at which the Go runtime samples the state of mutexes to generate mutex profile information.
--prometheus-external Publish prometheus metrics to external network
--prometheus-port Port to use for prometheus metrics (default 9876)
--protocol-id  Protocol ID to use (default "/gossamer/gssmr/0")
--public-dns Public DNS name of the node
--public-ip Public IP address of the node
--retain-blocks  Retain number of block from latest block while pruning (default 512)
--rewind Rewind head of chain to the given block number
--role Role of the node. Can be one of: full, light and authority
--rpc-external Enable external HTTP-RPC connections
--rpc-host HTTP-RPC server listening hostname
--rpc-methods API modules to enable via HTTP-RPC, comma separated list
--rpc-port HTTP-RPC server listening port (default 8545)
--state-pruning Pruning strategy to use. Supported strategy: archive
--telemetry-url URL of telemetry server to connect to
--unlock Unlock an account. eg. --unlock=0 to unlock account 0.
--unsafe-rpc Enable unsafe HTTP-RPC methods
--unsafe-rpc-external Enable external unsafe HTTP-RPC connections
--unsafe-ws-external Enable external unsafe WebSockets connections
--validator Run as a validator node
--wasm-interpreter WASM interpreter (default "wasmer")
--ws-external Enable external WebSockets connections
--ws-port WebSockets server listening port (default 8546)
```

## Gossamer Subcommands

List of available ***subcommands***:

```
SUBCOMMANDS:
    help, h           Shows a list of commands or help for one command
    account        Create and manage node keystore accounts
    export         Export configuration values to TOML configuration file
    init           Initialise node databases and load genesis data to state
    build-spec     Generates chain-spec JSON data, and can convert to raw chain-spec data
    import-runtime Imports a WASM runtime blob into the node's database
    import-state   Imports a state dump into the node's database
    prune-state    Prune state will prune the state trie
```

List of ***flags*** for `init` subcommand:

```
--force            Disable all confirm prompts (the same as answering "Y" to all)
--chain            Path to genesis JSON file
--base-path        Working directory for the node
```

List of ***flags*** for `account` subcommand:

```
--password      Password used to encrypt the keystore. Used with --generate or --unlock
--scheme        Keyring scheme (sr25519, ed25519, secp256k1
--keystore-path path to keystore
--keystore-file keystore file name
```

## Running Node Roles

Run an authority node:
```
./bin/gossamer --key alice --role authority
```

Run a non-authority node:
```
./bin/gossamer --key alice --role full
```

## Running Multiple Nodes

Two options for running another node at the same time...

(1) run `gossamer init` with two different `base-path` and manually update `port` in `base-path/config/config.toml`:
```
gossamer init --base-path ~/.gossamer/gssmr-alice --chain westend-local
gossamer init --base-path ~/.gossamer/gssmr-bob --chain westend-local
# open ~/.gossamer/gssmr-bob/config/config.toml, set port=7002
# set role=4 to also make bob an authority node, or role=1 to make bob a non-authority node
```

(2) run with `--base-path` flag:
```
./bin/gossamer --base-path ~/.gossamer/gssmr-alice --key alice --roles 4
./bin/gossamer --base-path ~/.gossamer/gssmr-bob --key bob --roles 4
```

or run with port, base-path flags:
```
./bin/gossamer --base-path ~/.gossamer/gssmr-alice --key alice --role 4 --port 7001
./bin/gossamer --base-path ~/.gossamer/gssmr-bob --key bob --role 4 --port 7002
```

To run more than two nodes, repeat steps for bob with a new `port` and `base-path` replacing `bob`.

Available built-in keys:
```
./bin/gossmer --key alice
./bin/gossmer --key bob
./bin/gossmer --key charlie
./bin/gossmer --key dave
./bin/gossmer --key eve
./bin/gossmer --key ferdie
./bin/gossmer --key george
./bin/gossmer --key heather
```

## Initialising Nodes

To initialise or re-initialise a node, use the init subcommand `init`:
```
./bin/gossamer init --base-path ~/.gossamer/gssmr-alice --chain westend-local
./bin/gossamer --base-path ~/.gossamer/gssmr-alice --key alice --roles 4
```

`init` can be used with the `--base-path` or `--chain` flag to re-initialise a custom node (ie, `bob` from the example above):
```
./bin/gossamer init --base-path ~/.gossamer/gssmr-bob --chain westend-local
```
