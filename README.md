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

Gossamer is an implementation of the [Polkadot Host](https://github.com/w3f/polkadot-spec): a framework used to build and run nodes for different blockchain protocols that are compatible with the Polkadot ecosystem.  The core of the Polkadot Host is the wasm runtime which handles the logic of the chain.

Gossamer includes node implementations for major blockchains within the Polkadot ecosystem and simplifies building node implementations for other blockchains. Runtimes built with [Substrate](https://github.com/paritytech/substrate) can plug their runtime into Gossamer to create a node implementation in Go.

For more information about Gossamer, the Polkadot ecosystem, and how to use Gossamer to build and run nodes for various blockchain protocols within the Polkadot ecosystem, check out the [Gossamer Docs](https://ChainSafe.github.io/gossamer).

## Get Started

### Prerequisites

install go version `>=1.15`

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

### Run Default Node

initialize default node:
```
./bin/gossamer --chain gssmr init
```

start default node:
```
./bin/gossamer --chain gssmr --key alice
```

The built-in keys available for the node are `alice`, `bob`, `charlie`, `dave`, `eve`, `ferdie`, `george`, and `ian`.

The node will not build blocks every slot by default; it will appear that the node is doing nothing, but it is actually waiting for a slot to build a block. If you wish to force it to build blocks every slot, you update the `[core]` section of `chain/gssmr/config.toml` to the following:

```
[core]
roles = 4
babe-authority = true
grandpa-authority = true
babe-threshold-numerator = 1
babe-threshold-denominator = 1
```

Then, re-run the above steps. NOTE: this feature is for testing only; if you wish to change the BABE block production parameters, you need to create a modified runtime.

### Run Kusama Node

initialize kusama node:
```
./bin/gossamer --chain kusama init
```

start kusama node:
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

initialize polkadot node:
```
./bin/gossamer --chain polkadot init
```

start polkadot node:
```
./bin/gossamer --chain polkadot
```

## Contribute

- Check out [Contributing Guidelines](.github/CONTRIBUTING.md)  
- Have questions? Say hi on [Discord](https://discord.gg/Xdc5xjE)!

## Donate

Our work on gossamer is funded by grants. If you'd like to donate, you can send us ETH or DAI at the following address:
`0x764001D60E69f0C3D0b41B0588866cFaE796972c`

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
	<img src="/docs/assets/img/chainsafe_gopher.png">
</p>

