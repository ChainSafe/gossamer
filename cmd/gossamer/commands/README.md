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
├── chain-spec.json
├── node-key.json
├── db
```

The node configuration can be modified in the `config.toml` file.

### Start the node

```bash
gossamer --basepath /tmp/gossamer --key alice 
```