<div align="center">
  <img alt="Gossamer logo" src="/docs/docs/assets/img/gossamer_banner.png" width="600" />
</div>
<div align="center">
  <a href="https://www.gnu.org/licenses/gpl-3.0">
    <img alt="License: GPL v3" src="https://img.shields.io/badge/License-GPLv3-blue.svg?style=for-the-badge&label=License" height="20"/>
  </a>
    <a href="https://github.com/ChainSafe/gossamer/actions">
    <img alt="build status" src="https://img.shields.io/github/workflow/status/ChainSafe/gossamer/build?branch=development&style=for-the-badge&logo=github&label=build" height="20"/>
  </a>
  <a href="https://godoc.org/github.com/ChainSafe/gossamer">
    <img alt="go doc" src="http://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge" height="20" />
  </a>
  <a href="https://goreportcard.com/report/github.com/ChainSafe/gossamer">
    <img alt="go report card" src="https://goreportcard.com/badge/github.com/ChainSafe/gossamer?style=for-the-badge" height="20" />
  </a>
</div>
<div align="center">
  <a href="https://app.codecov.io/gh/ChainSafe/gossamer">
    <img alt="Test Coverage" src="https://img.shields.io/codecov/c/github/ChainSafe/gossamer/development?style=for-the-badge" height="20" />
  </a>
    <a href="https://discord.gg/zy8eRF7FG2">
    <img alt="Discord" src="https://img.shields.io/discord/593655374469660673.svg?style=for-the-badge&label=Discord&logo=discord" height="20"/>
  </a>
  <a href="https://medium.com/chainsafe-systems/tagged/polkadot">
    <img alt="Gossamer Blog" src="https://img.shields.io/badge/Medium-grey?style=for-the-badge&logo=medium" height="20" />
  </a>
    <a href="https://medium.com/chainsafe-systems/tagged/polkadot">
    <img alt="Twitter" src="https://img.shields.io/twitter/follow/chainsafeth?color=blue&label=follow&logo=twitter&style=for-the-badge" height="20"/>
  </a>
</div>
<br />

## A Go Implementation of the Polkadot Host

> **Warning**
> 2022-11-01: Gossamer is pre-production software

Gossamer is an implementation of the [Polkadot Host](https://wiki.polkadot.network/docs/learn-polkadot-host): an execution environment for the Polkadot runtime, which is materialized as a Web Assembly (Wasm) blob.  In addition to running an embedded Wasm executor, a Polkadot Host must orchestrate a number of interrelated services, such as [networking](dot/network/README.md), block production, block finalization, a JSON-RPC server, [and more](cmd/gossamer/README.md).

For more information about Gossamer, check out the [Gossamer Docs](https://ChainSafe.github.io/gossamer).

## Get Started

### Prerequisites

Install Go version [`>=1.18`](https://go.dev/dl/#go1.18)

### Installation

get the [ChainSafe/gossamer](https://github.com/ChainSafe/gossamer) repository:

```
git clone git@github.com:ChainSafe/gossamer
cd gossamer
```

build gossamer command:

```
make gossamer
```

### Troubleshooting for Apple Silicon users

If you are facing the following problem with the `wasmer`:

```
undefined: cWasmerImportObjectT
undefined: cWasmerImportFuncT
undefined: cWasmerValueTag
```

Make sure you have the following Golang enviroment variables:

- GOARCH="amd64"
- CGO_ENABLED="1"

> use _go env_ to see all the Golang enviroment variables

> use _go env -w **ENV_NAME**=**ENV_VALUE**_ to set the new value

### Run Development Node

To initialise a development node:

```
./bin/gossamer --chain dev init
```

To start the development node:

```
./bin/gossamer --chain dev
```

The development node is configured to produce a block every slot and to finalise a block every round (as there is only one authority, `alice`.)

### Run Gossamer Node

The gossamer node runs by default as an authority with 9 authorites set at genesis. The built-in keys, corresponding to the authorities, that are available for the node are `alice`, `bob`, `charlie`, `dave`, `eve`, `ferdie`, `george`, and `ian`.

To initialise a gossamer node:

```
./bin/gossamer --chain gssmr init
```

To start the gossamer node:

```
./bin/gossamer --chain gssmr --key alice
```

Note: If you only run one gossamer node, the node will not build blocks every slot or finalize blocks; it will appear that the node is doing nothing, but it is actually waiting for a slot to build a block. This is because there are 9 authorities set, so at least 6 of the authorities should be run for a functional network. If you wish to reduce the number of authorities, you can modify the genesis file in `chain/gssmr/genesis-spec.json`.

### Run Kusama Node

Kusama is currently supported as a **full node**, ie. it can sync the chain but not act as an authority.

To initialise a kusama node:

```
./bin/gossamer --chain kusama init
```

To start the kusama node:

```
./bin/gossamer --chain kusama
```

The node may not appear to do anything for the first minute or so (it's bootstrapping to the network.) If you wish to see what is it doing in this time, you can turn on debug logs in `chain/gssmr/config.toml`:

```
[log]
network = "debug"
```

After it's finished bootstrapping, the node should begin to sync.

### Run Polkadot Node

Polkadot is currently supported as a **full node**, ie. it can sync the chain but not act as an authority.

To initialise a polkadot node:

```
./bin/gossamer --chain polkadot init
```

To start the polkadot node:

```
./bin/gossamer --chain polkadot
```

## Contribute

- Check out [Contributing Guidelines](.github/CONTRIBUTING.md) and our [code style](.github/CODE_STYLE.md) document
- Have questions? Say hi on [Discord](https://discord.gg/Xdc5xjE)!

## Donate

Our work on Gossamer is funded by the community. If you'd like to support us with a donation:
- DOT: [`14gaKBxYkbBh2SKGtRDdhuhtyGAs5XLh55bE5x4cDi5CmL75`](https://polkadot.subscan.io/account/14gaKBxYkbBh2SKGtRDdhuhtyGAs5XLh55bE5x4cDi5CmL75)
- KSM: [`FAjhFSFoM6X8CxeSp6JE2fPECauCA5NxyB1rAGNSkrVaMtf`](https://kusama.subscan.io/account/FAjhFSFoM6X8CxeSp6JE2fPECauCA5NxyB1rAGNSkrVaMtf)
- ETH/DAI: `0x764001D60E69f0C3D0b41B0588866cFaE796972c`

## ChainSafe Security Policy

### Reporting a Security Bug

We take all security issues seriously, if you believe you have found a security issue within a ChainSafe
project please notify us immediately. If an issue is confirmed, we will take all necessary precautions
to ensure a statement and patch release is made in a timely manner.

Please email us a description of the flaw and any related information (e.g. reproduction steps, version) to
[security at chainsafe dot io](mailto:security@chainsafe.io).

## License

_GNU Lesser General Public License v3.0_

<br />
<p align="center">
 <img src="/docs/docs/assets/img/chainsafe_gopher.png">
</p>
