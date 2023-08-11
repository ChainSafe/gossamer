# Quick start

## Install Gossamer from source

You'll need to have [Go](https://golang.org/doc/install) installed on your machine and the `GOPATH` environment variable set.

### Clone the repository

```bash
git clone https://github.com/ChainSafe/gossamer.git
cd gossamer
```

### Compile

To put the binary in ./bin, run:
```bash
make build
```

or to put the binary in the `$GOPATH/bin` directory, run:

```bash
make install
```

Verify the installation by running:

```bash
gossamer version
```

## Run Gossamer

### Initialize the node

```bash
gossamer init --chain westend --key alice --basepath /tmp/gossamer
```

This initialises the node with the default configuration for the `westend` chain with the `alice` keypair at the base-path `/tmp/gossamer`.

```
Supported flags:
--chain: The chain spec to initialise the node with. Supported chains are `polkadot`, `kusama`, `westend`, `westend-dev` and `westend_local`. It also accepts the chain-spec json path.
--key: The keypair to use for the node.
--basepath: The working directory for the node.
```

The init command will create the following files in the base-path:

```
├── config
│   ├── config.toml
├── chain-spec-raw.json
├── node-key.json
├── db
```

The node configuration can be modified in the `config.toml` file.

### Start the node

```bash
gossamer --basepath /tmp/gossamer --key alice 
```

**Note: The `init` command is optional. If the node is not initialised, it will be initialised with the default configuration.**

Here are the list of basic flags for the `gossamer` command:

```
--basepath: The working directory for the node.
--chain: The chain spec to initialise the node with. Supported chains are `polkadot`, `kusama`, `westend`, `westend-dev` and `westend_local`. It also accepts the chain-spec json path.
--key: The keypair to use for the node.
--name: The name of the node.
--id: The id of the node.
--log:  Set a logging filter.
	    Syntax is a list of 'module=logLevel' (comma separated)
	    e.g. --log sync=debug,core=trace
	    Modules are global, core, digest, sync, network, rpc, state, runtime, babe, grandpa, wasmer.
	    Log levels (least to most verbose) are error, warn, info, debug, and trace.
	    By default, all modules log 'info'.
	    The global log level can be set with --log global=debug
--prometheus-port: The port to expose prometheus metrics.
--retain-blocks: retain number of block from latest block while pruning
--pruning: The pruning strategy to use. Supported strategiey: `archive`
--no-telemetry: Disable telemetry.
--telemetry-urls: The telemetry endpoints to connect to.
--prometheus-external: Expose prometheus metrics externally.
```

To see all the available flags, run:

```bash
gossamer --help
```

## Other commands supported by Gossamer CLI

### Account Command

The `account` command provides the user with capabilities related to generating and using `ed25519`, `secp256k1`, and
`sr25519` [account keys](https://wiki.polkadot.network/docs/learn-keys), and managing the keys present in the
[Gossamer keystore](#keystore). The account command is defined in [account.go](./commands/account.go); it is an interface
to the capabilities defined in the [`lib/crypto`](../../lib/crypto) and [`lib/keystore`](../../lib/keystore) packages.
This subcommand provides capabilities that are similar to
[Parity's Subkey utility](https://docs.substrate.io/v3/tools/subkey).

The account command supports following arguments:
- `generate` - generates a new key pair; specify `--scheme ed25519`, `--scheme secp256k1`, or `--scheme sr25519` (default)
- `list` - lists the keys in the Gossamer keystore
- `import` - imports a key from a keystore file
- `import-raw` - imports a raw key from a keystore file

Supported flags:
- `keystore-path` - path to the Gossamer keystore
- `keystore-file` - path to the keystore file
- `chain` - path to the human-readable chain-spec file
- `--scheme` - `ed25519`, `secp256k1`, or `sr25519` (default)
- `--password` - allows the user to provide a password to either encrypt a generated key or unlock the Gossamer keystore

Examples:
- `gossamer account generate --scheme ed25519` - generates an `ed25519` key pair
- `gossamer account list` - lists the keys in the Gossamer keystore
- `gossamer account import --keystore-file keystore.json` - imports a key from a keystore file
- `gossamer account import-raw --keystore-file keystore.json` - imports a raw key from a keystore file

### Import Runtime Command

This subcommand takes a [Wasm runtime binary](https://wiki.polkadot.network/docs/learn-wasm) and appends it to a
[genesis](https://wiki.polkadot.network/docs/glossary#genesis) configuration file; it does not require any flags, but
expects both the path to a Wasm file and a genesis configuration file to be provided as a command-line parameter (example:
`./bin/gossamer import-runtime --wasm-file runtime.wasm --chain chain-spec.json > updated_chain-spec.json`).

### Build Spec Command

This subcommand allows the user to "compile" a human-readable Gossamer genesis configuration file into a format that the
Gossamer node can consume. If the `--chain` parameter is not provided, the generated genesis configuration will
represent the Gossamer default configuration.

- `--chain` - path to the human-readable chain-spec file that should be compiled into a format that Gossamer can
  consume
- `--raw` - when this flag is present, the output will be a raw genesis spec described as a JSON document
- `--output-path` - path to the file where the compiled chain-spec should be written

Examples:
- `gossamer build-spec --chain chain-spec.json --output-path compiled-chain-spec.json` - compiles a human-readable
  chain-spec into a format that Gossamer can consume
- `gossamer build-spec --chain chain-spec.json --raw --output-path compiled-chain-spec.json` - compiles a human-readable
  chain-spec into a format that Gossamer can consume, and outputs the raw genesis spec as a JSON document

### Import State Command

The `import-state` subcommand allows a user to seed [Gossamer storage](../../dot/state) with key-value pairs in the form
of a JSON file. The input for this subcommand can be retrieved from
[the `state_getPairs` RPC endpoint](https://github.com/w3f/PSPs/blob/master/PSPs/drafts/psp-6.md#1114-state_getpairs).

- `--first-slot` - the first [BABE](https://wiki.polkadot.network/docs/learn-consensus#block-production-babe) slot,
  which can be found by checking the
  [BABE pre-runtime digest](https://crates.parity.io/sp_runtime/enum.DigestItem.html#variant.PreRuntime) for a chain's
  first block _after_ its [genesis block](https://wiki.polkadot.network/docs/glossary#genesis) (e.g.
  [Polkadot on Polkascan](https://polkascan.io/polkadot/log/1-0))
- `--header` - path to a JSON file that describes the block header corresponding to the given state
- `--state` - path to a JSON file that contains the key-value pairs with which to seed Gossamer storage
- `--chain` - path to the human-readable chain-spec file

Examples:
- `gossamer import-state --first-slot 1 --header header.json --state state.json --chain chain-spec.json` - seeds Gossamer
  storage with key-value pairs from a JSON file

## Client Components

In its default method of execution, Gossamer orchestrates a number of modular services that run
[concurrently as goroutines](https://www.golang-book.com/books/intro/10) and work together to implement the protocols of
a blockchain network. Alongside these services, Gossamer manages [a keystore](#keystore), [a runtime](#runtime), and
[monitoring utilities](#monitoring), all of which are described in greater detail below. The entry point to the Gossamer
blockchain client capabilities is the `gossamerAction` function that is defined in [main.go](main.go), which in turn
invokes the `NewNode` function in [dot/node.go](../../dot/node.go). `NewNode` calls into functions that are defined in
[dot/services.go](../../dot/services.go) and starts the services that power a Gossamer node.

### Services & Capabilities

What follows is a list that describes the services and capabilities that inform a Gossamer blockchain client:

#### State

This service is a wrapper around an instance of [`pebble`](https://github.com/cockroachdb/pebble), a LevelDB/RocksDB inspired key-value. 
The state service provides storage capabilities for the other Gossamer services - each service is assigned a prefix that is added
to its storage keys. The state service is defined in [dot/state/service.go](../../dot/state/service.go).

#### Network

The network service, which is defined in [dot/network/service.go](../../dot/network/service.go), is built on top of
[the Go implementation](https://github.com/libp2p/go-libp2p) of [the `libp2p` protocol](https://libp2p.io/). This
service manages a `libp2p` "host", a peer-to-peer networking term for a network participant that is providing both
client _and_ server capabilities to a peer-to-peer network. Gossamer's network service manages the discovery of other
hosts as well as the connections with these hosts that allow Gossamer to communicate with its network peers.

#### Digest Handler

The digest handler ([dot/digest/digest.go](../../dot/digest/digest.go)) manages the verification of the
[digests](https://docs.substrate.io/v3/getting-started/glossary/#digest) that are present in block headers.

#### Consensus

The BABE and GRANDPA services work together to provide Gossamer with its
[hybrid consensus](https://wiki.polkadot.network/docs/learn-consensus#hybrid-consensus) capabilities. The term "hybrid
consensus" refers to the fact that block _production_ is decoupled from block _finalisation_. Block production is
handled by the BABE service, which is defined in [lib/babe/babe.go](../../lib/babe/babe.go); block finalisation is
handled by the GRANDPA service, which is defined in [lib/grandpa/grandpa.go](../../lib/grandpa/grandpa.go).

#### Sync

This service is concerned with keeping Gossamer in sync with a blockchain - it implements a "bootstrap" mode, to
download and verify blocks that are part of an existing chain's history, and a "tip-syncing" mode that manages the
multiple candidate forks that may exist at the head of a live chain. The sync service makes use of
[a block verification utility](../../lib/babe/verify.go) that implements BABE logic and is used by Gossamer to verify
blocks that were produced by other nodes in the network. The sync service is defined in
[dot/sync/syncer.go](../../dot/sync/syncer.go).

#### RPC

This service, which is defined in [dot/rpc/service.go](../../dot/rpc/service.go), exposes a JSON-RPC interface that is
used by client applications like [Polkadot JS Apps UI](https://polkadot.js.org/apps/). The RPC interface is used to
interact with Gossamer to perform administrative tasks such as key management, as well as for interacting with the
runtime by querying storage and submitting transactions, and inspecting the chain's history.

#### System

The system service is defined in [dot/system/service.go](../../dot/system/service.go) and exposes metadata about the
Gossamer system, such as the names and versions of the protocols that it implements.

#### Core

As its name implies, the core service ([dot/core/service.go](../../dot/core/service.go)) encapsulates a range of
capabilities that are central to the functioning of a Gossamer node. In general, the core service is a type of
dispatcher that coordinates interactions between services, e.g. writing blocks to the database, reloading
[the runtime](#runtime) when its definition is updated, etc.

### Keystore

The Gossamer keystore ([lib/keystore](../../lib/keystore)) is used for managing the public/private cryptographic key
pairs that are used for participating in a blockchain network. Public keys are used to identify network participants;
network participants use their private keys to sign messages in order to authorise privileged actions. In addition to
informing the Gossamer blockchain client capabilities, the Gossamer keystore is accessible by way of the `account`
subcommand. The Gossamer keystore manages a number of key types, some of which are listed below:

- `babe` - this key is used for signing messages related to the BABE block production algorithm
- `gran` - the GRANDPA key is used for participating in GRANDPA block finalisation
- `imon` - the name of this key is a reference to "ImOnline", which is an
  [online message](https://wiki.polkadot.network/docs/glossary#online-message) that Gossamer nodes use to report
  liveliness

### Runtime

In addition to the above-described services, Gossamer hosts a Wasm execution environment that is used to manage an
upgradeable blockchain runtime. The runtime must be implemented in Wasm, and must expose an interface that is specified
in [lib/runtime/interface.go](../../lib/runtime/interfaces.go). The runtime defines the blockchain's state transition
function, and the various Gossamer services consume this capability in order to author blocks, as well as to verify
blocks that were authored by network peers. The runtime is dependent on a
[Wasm host interface](https://docs.wasmer.io/integrations/examples/host-functions), which Gossamer implements and is
defined in [lib/runtime/wasmer/exports.go](../../lib/runtime/wasmer/exports.go).

### Monitoring

Gossamer publishes telemetry data and also includes an embedded Prometheus server that reports metrics. The metrics
capabilities are defined in the [dot/telemetry](../../dot/telemetry) package and build on
[the metrics library that is included with Go Ethereum](https://github.com/ethereum/go-ethereum/blob/master/metrics/README.md).
The default listening address for Prometheus metrics is `localhost:9876`, and Gossamer allows the user to configure this parameter with the
`--metrics-address` command-line parameter. The Gossamer telemetry server publishes telemetry data that is compatible with
[Polkadot Telemetry](https://github.com/paritytech/substrate-telemetry) and
[its helpful UI](https://telemetry.polkadot.io/).