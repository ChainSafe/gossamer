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

This initialises the node with the default configuration for the `westend` chain with the `alice` keypair at the basepath `/tmp/gossamer`.

```
Supported flags:
--chain: The chain spec to initialise the node with. Supported chains are `polkadot`, `kusama`, `westend`, `westend-dev` and `westend_local`. It also accepts the chain-spec json path.
--key: The keypair to use for the node.
--basepath: The working directory for the node.
```

The init command will create the following files in the basepath:

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
--log-level: The global log level. Supported levels are `crit`, `error`, `warn`, `info`, `debug` and `trace`.
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
[Gossamer keystore](#keystore). The account command is defined in [account.go](account.go); it is an interface
to the capabilities defined in the [`lib/crypto`](../../lib/crypto) and [`lib/keystore`](../../lib/keystore) packages.
This subcommand provides capabilities that are similar to
[Parity's Subkey utility](https://docs.substrate.io/v3/tools/subkey).

The command can be invoked with the following subcommands:

- `gossamer account generate` - creates a new key pair; specify `--scheme ed25519`, `--scheme secp256k1`, or `--scheme sr25519` (default)
- `gossamer account list` - lists the keys in the Gossamer keystore
- `gossamer account import` - imports a key from a keystore file
- `gossamer account import --raw` - imports a raw key from a keystore file
- `gossamer account --password` - allows the user to provide a password to either encrypt a generated key or unlock the Gossamer keystore

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