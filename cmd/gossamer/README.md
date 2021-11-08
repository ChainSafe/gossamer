# Gossamer `cmd` Package

This package encapsulates the entry point to Gossamer - it uses the popular
[`cli` package from `urfave`](https://github.com/urfave/cli/blob/master/docs/v1/manual.md) to expose a command-line
interface (CLI). The Gossamer CLI accepts several subcommands, each of which is associated with an "action"; these
subcommands and their corresponding actions are defined in [`main.go`](main.go). When the Gossamer CLI is executed
without a subcommand, the `gossamerAction` is invoked.

## Actions & Subcommands

What follows is a list of the Gossamer subcommands, as well as an overview of some of the flags/parameters they accept.
The flags/parameters that the Gossamer CLI supports are defined in [`flags.go`](flags.go). For an exhaustive reference
of the Gossamer CLI capabilities, follow [the installation instructions](../../README.md#installation) and execute
`./bin/gossamer --help`.

### Default Command

This is the default Gossamer execution method, which invokes the `gossamerAction` function defined in
[`main.go`](main.go) - it will launch a Gossamer blockchain client. The details of how Gossamer orchestrates a
blockchain client are [described below in the Client Components section](#client-components).

- `--basepath` - the path to the directory where Gossamer will store its data
- `--chain` - specifies the [chain configuration](../../chain) that the
  [Gossamer host node](https://chainsafe.github.io/gossamer/getting-started/overview/host-architecture/) should load
- `--key` - specifies a test keyring account to use (e.g. `--key=alice`)
- `--log` - supports levels `crit` (silent), `error`, `warn`, `info`, `debug`, and `trce` (detailed), default is `info`
- `--name` - node name, as it will appear in, e.g., [telemetry](https://telemetry.polkadot.io/)

### Init Subcommand

This subcommand accepts a genesis configuration file and uses it to initialise the Gossamer node and its state. The
`init` subcommand invokes the `initAction` function defined in [`main.go`](main.go).

- `--genesis` - path to the "compiled" genesis configuration file that should be used to initialise the Gossamer node
  and its state

### Account Subcommand

The `account` subcommand provides the user with capabilities related to generating and using `ed25519`, `secp256k1`, and
`sr25519` [account keys](https://wiki.polkadot.network/docs/learn-keys), and managing the keys present in the
[Gossamer keystore](#keystore). The `accountAction` function is defined in [account.go](account.go); it is an interface
to the capabilities defined in the [`lib/crypto`](../../lib/crypto) and [`lib/keystore`](../../lib/keystore) packages.
This subcommand provides capabilities that are similar to
[Parity's Subkey utility](https://docs.substrate.io/v3/tools/subkey).

- `--generate` - creates a new key pair; specify `--ed25519`, `--secp256k1`, or `--sr25519` (default)
- `--list` - lists the keys in the Gossamer keystore
- `--password` - allows the user to provide a password to either encrypt a generated key or unlock the Gossamer keystore

### Import Runtime Subcommand

This subcommand takes a [Wasm runtime binary](https://wiki.polkadot.network/docs/learn-wasm) and uses it to generate a
[genesis](https://wiki.polkadot.network/docs/glossary#genesis) configuration file; it does not require any flags, but
expects the path to a Wasm file to be provided as a command-line parameter (example:
`./bin/gossamer import-runtime runtime.wasm > genesis.json`). The `import-runtime` subcommand invokes the
`importRuntimeAction` function defined in [`main.go`](main.go).

### Build Spec Subcommand

This subcommand allows the user to "compile" a human-readable Gossamer genesis configuration file into a format that the
Gossamer node can consume. If the `--genesis` parameter is not provided, the generated genesis configuration will
represent the Gossamer default configuration. The `build-spec` subcommand invokes the `buildSpecAction` function defined
in [`main.go`](main.go).

- `--genesis` - path to the human-readable configuration file that should be compiled into a format that Gossamer can
  consume
- `--raw` - when this flag is present, the output will be a raw genesis spec described as a JSON document

### Import State Subcommand

The `import-state` subcommand allows a user to seed [Gossamer storage](../../dot/state) with key-value pairs in the form
of a JSON file. The input for this subcommand can be retrieved from
[the `state_getPairs` RPC endpoint](https://github.com/w3f/PSPs/blob/master/PSPs/drafts/psp-6.md#1114-state_getpairs).
The `importStateAction` function is defined in [`main.go`](main.go).

- `--first-slot` - the first [BABE](https://wiki.polkadot.network/docs/learn-consensus#block-production-babe) slot,
  which can be found by checking the
  [BABE pre-runtime digest](https://crates.parity.io/sp_runtime/enum.DigestItem.html#variant.PreRuntime) for a chain's
  first block _after_ its [genesis block](https://wiki.polkadot.network/docs/glossary#genesis) (e.g.
  [Polkadot on Polkascan](https://polkascan.io/polkadot/log/1-0))
- `--header` - path to a JSON file that describes the block header corresponding to the given state
- `--state` - path to a JSON file that contains the key-value pairs with which to seed Gossamer storage

### Export Subcommand

The `export` subcommand transforms a genesis configuration and Gossamer state into a TOML configuration file. This
subcommand invokes the `exportAction` function defined in [`export.go`](export.go).

- `--config` - path to a TOML configuration file (e.g. those defined in [the `chain` directory](../../chain))
- `--basepath` - path to the Gossamer data directory that defines the state to export

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

This service is a wrapper around an instance of [`chaindb`](https://github.com/ChainSafe/chaindb), a key-value database
that is built on top of [BadgerDB](https://github.com/dgraph-io/badger) from [Dgraph](https://dgraph.io/). The state
service provides storage capabilities for the other Gossamer services - each service is assigned a prefix that is added
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
blocks that were produced by other other nodes in the network. The sync service is defined in
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
in [lib/runtime/interface.go](../../lib/runtime/interface.go). The runtime defines the blockchain's state transition
function, and the various Gossamer services consume this capability in order to author blocks, as well as to verify
blocks that were authored by network peers. The runtime is dependent on a
[Wasm host interface](https://docs.wasmer.io/integrations/examples/host-functions), which Gossamer implements and is
defined in [lib/runtime/wasmer/exports.go](../../lib/runtime/wasmer/exports.go).

### Monitoring

Gossamer publishes telemetry data and also includes an embedded Prometheus server that reports metrics. The metrics
capabilities are defined in the [dot/metrics](../../dot/metrics) package and build on
[the metrics library that is included with Go Ethereum](https://github.com/ethereum/go-ethereum/blob/master/metrics/README.md).
The default port for Prometheus metrics is 9090, and Gossamer allows the user to configure this parameter with the
`--metrics-port` command-line parameter. The Gossamer telemetry server publishes telemetry data that is compatible with
[Polkadot Telemetry](https://github.com/paritytech/substrate-telemetry) and
[its helpful UI](https://telemetry.polkadot.io/).
