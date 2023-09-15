<div align="center">
  <img alt="Gossamer logo" src="/docs/docs/assets/img/gossamer_banner.png" width="600" />
</div>
<div align="center">
  <a href="https://www.gnu.org/licenses/gpl-3.0">
    <img alt="License: GPL v3" src="https://img.shields.io/badge/License-GPLv3-blue.svg?style=for-the-badge&label=License" height="20"/>
  </a>
    <a href="https://github.com/ChainSafe/gossamer/actions">
    <img alt="build status" src="https://img.shields.io/github/actions/workflow/status/ChainSafe/gossamer/build.yml?branch=development&style=for-the-badge&logo=github&label=build" height="20"/>
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

> **Warning**
>
> The Gossamer Polkadot Host is pre-production software [2022-12-01]

Gossamer is a [Golang](https://go.dev/) implementation of the
[Polkadot Host](https://wiki.polkadot.network/docs/learn-polkadot-host): an
execution environment for the Polkadot runtime, which is materialized as a Web
Assembly (Wasm) blob. In addition to running an embedded Wasm executor, a
Polkadot Host must orchestrate a number of interrelated services, such as
[networking](dot/network/README.md), block production, block finalization, a
JSON-RPC server, [and more](cmd/gossamer/README.md#client-components).

## Getting Started

To get started with Gossamer, follow the steps below to build the source code
and start a development network.

### Prerequisites

[Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) is required
to acquire the Gossamer source code, and
[Make](https://tilburgsciencehub.com/building-blocks/configure-your-computer/automation-and-workflows/make/)
is used to build it. Building Gossamer requires version 1.20 or higher of
[Golang](https://go.dev/dl/).

### Installation

Clone the [Gossamer](https://github.com/ChainSafe/gossamer) repository and
checkout the `development` branch:

```sh
git clone git@github.com:ChainSafe/gossamer
cd gossamer
git checkout development
```

Build Gossamer:

```sh
make gossamer
```

Or build Gossamer _and_ move the resulting executable to `$GOPATH/bin`:

```sh
make install
```

To install Gossamer

## Use Gossamer

A comprehensive guide to
[Gossamer's end-user capabilities](cmd/gossamer/README.md) is located in the
`cmd/gossamer` directory. What follows is a guide to Gossamer's capabilities as
a Polkadot Host.

### Chain Specifications

A chain specification is a JSON document that defines the
[genesis](https://wiki.polkadot.network/docs/glossary#genesis) block of a
blockchain network, as well as network parameters and metadata (e.g. network
name, bootnodes,
[telemetry endpoints](https://wiki.polkadot.network/docs/build-node-management#monitoring-and-telemetry),
etc). It is necessary to provide Gossamer with a chain specification in order to
use it as a Polkadot Host. The Gossamer repository includes a number of chain
specifications, some of which will be used in this guide.

### Configuration Files

Gossamer exposes a number of configuration parameters, such as the location of a
chain specification file. Although it's possible to use command-line parameters,
this guide will focus on the usage of Gossamer TOML configuration files, which
define a set of configuration values in a declarative, portable, reusable
format. The chain specifications that are used in this guide are each
accompanied by one or more configuration files.

### Single-Node Development Network

The name of the Polkadot test network is "Westend", and the Gossamer repository
includes a chain specification and configuration file for a single-node, local
Westend test network.

First, initialize the directory that will be used by the Gossamer node to manage
its state:

```sh
./bin/gossamer init --force --chain westend-dev
```

Now, start Gossamer as a host for the local Westend development chain:

```sh
./bin/gossamer --chain westend-dev
```

### Multi-Node Development Network

The multi-node development network includes three participants: the Alice, Bob,
and Charlie test accounts. In three separate terminals, initialize the data
directories for the three Gossamer instances:

```sh
./bin/gossamer init --force --chain westend-local --alice
```

```sh
./bin/gossamer init --force --chain westend-local --bob
```

```sh
./bin/gossamer init --force --config westend-local --charlie
```

Then start the three hosts:

```sh
./bin/gossamer --chain westend-local --alice
```

```sh
./bin/gossamer --chain westend-local --bob
```

```sh
./bin/gossamer --chain westend-local --charlie
```

## Contribute

- Check out the [Contributing Guidelines](.github/CONTRIBUTING.md) and our
  [style guide](.github/CODE_STYLE.md).
- Have questions or just want to say hi? Join us on
  [Discord](https://discord.gg/Xdc5xjE)!

## Donate

Our work on Gossamer is funded by the community. If you'd like to support us
with a donation:

- DOT:
  [`14gaKBxYkbBh2SKGtRDdhuhtyGAs5XLh55bE5x4cDi5CmL75`](https://polkadot.subscan.io/account/14gaKBxYkbBh2SKGtRDdhuhtyGAs5XLh55bE5x4cDi5CmL75)
- KSM:
  [`FAjhFSFoM6X8CxeSp6JE2fPECauCA5NxyB1rAGNSkrVaMtf`](https://kusama.subscan.io/account/FAjhFSFoM6X8CxeSp6JE2fPECauCA5NxyB1rAGNSkrVaMtf)
- ETH/DAI: `0x764001D60E69f0C3D0b41B0588866cFaE796972c`

## ChainSafe Security Policy

We take all security issues seriously, if you believe you have found a security
issue within a ChainSafe project please notify us immediately. If an issue is
confirmed, we will take all necessary precautions to ensure a statement and
patch release is made in a timely manner.

### Reporting a Security Bug

Please email us a description of the flaw and any related information (e.g.
reproduction steps, version) to
[security at chainsafe dot io](mailto:security@chainsafe.io).

## License

_GNU Lesser General Public License v3.0_

<br />
<p align="center">
 <img src="/docs/docs/assets/img/chainsafe_gopher.png">
</p>
