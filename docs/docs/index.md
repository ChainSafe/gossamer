---
layout: default
permalink: /
---

<div align="center">
  <img alt="Gossamer logo"  src="./assets/Gossamer_Black_Name.svg" width="600" />
</div>
<br />
<br />
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
  <a href="https://codeclimate.com/github/ChainSafe/gossamer/badges">
    <img alt="maintainability" src="https://img.shields.io/codeclimate/maintainability/ChainSafe/gossamer?style=for-the-badge" height="20" />
  </a>
  <a href="https://codeclimate.com/github/ChainSafe/gossamer/test_coverage">
    <img alt="Test Coverage" src="https://img.shields.io/codeclimate/coverage/ChainSafe/gossamer?style=for-the-badge" height="20" />
  </a>
    <a href="https://discord.gg/zy8eRF7FG2">
    <img alt="Discord" src="https://img.shields.io/discord/593655374469660673.svg?style=for-the-badge&label=Discord&logo=discord" height="20"/>
  </a>
  <a href="https://medium.com/chainsafe-systems/tagged/polkadot">
    <img alt="Gossamer Blog" src="https://img.shields.io/badge/Medium-grey?style=for-the-badge&logo=medium" height="20" />
  </a>
    <a href="https://medium.com/chainsafe-systems/tagged/polkadot">
    <img alt="Gossamer Blog" src="https://img.shields.io/twitter/follow/chainsafeth?color=blue&label=follow&logo=twitter&style=for-the-badge" height="20"/>
  </a>
</div>
<br />

## A Go Implementation of the Polkadot Host

Gossamer is an implementation of the <a target="_blank" rel="noopener noreferrer"  href="https://github.com/w3f/polkadot-spec">Polkadot Host</a>: a framework used to build and run nodes for different blockchain protocols that are compatible with the Polkadot ecosystem.  The core of the Polkadot Host is the wasm runtime which handles the logic of the chain.

Gossamer includes node implementations for major blockchains within the Polkadot ecosystem and simplifies building node implementations for other blockchains. Runtimes built with <a target="_blank" rel="noopener noreferrer" href="https://github.com/paritytech/substrate">Substrate</a> can plug their runtime into Gossamer to create a node implementation in Go.

For more information about Gossamer, the Polkadot ecosystem, and how to use Gossamer to build and run nodes for various blockchain protocols within the Polkadot ecosystem, check out the [Gossamer Docs](https://ChainSafe.github.io/gossamer).

***Gossamer Docs*** is an evolving set of documents and resources to help you understand Gossamer, the Polkadot ecosystem, and how to build and run nodes using Gossamer. 

- If you are new to Gossamer and the Polkadot ecosystem, we recommend starting with <a target="_blank" rel="noopener noreferrer" href="https://www.youtube.com/watch?v=nYkbYhM5Yfk">this video</a>  and then working your way through [General Resources](./welcome/general-resources/).

- If you are already familiar with Gossamer and the Polkadot ecosystem, or you just want to dive in, head over to [Get Started](./welcome/get-started) to run your first node using Gossamer.

- If you are looking to build a node with Gossamer, learn how Gossamer can be used to build and run custom node implementations using Gossamer as a framework (keep reading).

## Framework

Gossamer is a ***modular blockchain framework*** used to build and run nodes for different blockchain protocols within the Polkadot ecosystem.

- The ***simplest*** way to use the framework is using the base node implementation with a custom configuration file (see [Configuration](./running-gossamer/configuration)).

- The ***more advanced***  way to use the framework is using the base node implementation with a compiled runtime and custom runtime imports (see [Import Runtime](./building-gossamer/import-runtime)). 

- The ***most advanced***  way to use the framework is building custom node services or a custom node implementation (see [Custom Services](./building-gossamer/custom-services)).

Our primary focus has been an initial implementation of the Polkadot Host. Once we feel confident our initial implementation is fully operational and secure, we will expand the Gossamer framework to include a runtime library and other tools and services that will enable Go developers to build, test, and run custom-built blockchain protocols within the Polkadot ecosystem.

## Table of Contents

- **Getting Started**
  - [Overview](./)
  - [Installation](/getting-started/installation) 
  - [Host Architecture](/getting-started/overview/host-architecture)
  - [Package lib](getting-started/overview/package-library) 
- **Resources**
  - [General Resources](/getting-started/resources/general-resources)
  - [Developer Resources](/getting-started/resources/developer-resources)
- **Usage**
  - [Command line](/usage/command-line)
  - [Configuration](/usage/configuration)
  - [Run official nodes](/usage/run-official-nodes)
  - [Import Runtime](/usage/import-runtime)
- **Integrate**
  - [Connect polkadot.js](/integrate/connect-to-polkadot-js)
- **Testing & Debugging**
  - [Running intergration tests](/testing-and-debugging/intergration-tests)
  - [Running unit tests](/testing-and-debugging/unit-tests)
  - [Running docker tests](/testing-and-debugging/docker-tests)
  - [Logger usage](/testing-and-debugging/logger-usage)
  - [Debugging](/testing-and-debugging/debugging)
- **Deployment**
  - [Docker usage](/deployment/docker-usage)
## Advanced
  - [SCALE](/advanced/scale-examples)
  - [Custom Services](/advanced/custom-servives)
## Contributing
  - [Overview](contibuting.md) - docs/docs/contibuting.md


## Connect

Let us know if you have any feedback or ideas that might help us improve our documentation or if you have any resources that you would like to see added. If you are planning to use Gossamer or any of the Gossamer packages, please say hello! You can find us on <a target="_blank" rel="noopener noreferrer" href="https://discord.gg/Xdc5xjE">Discord</a>.

## Contribute

Contributions to this site and it's contents are more than welcome. If you would like to contribute, please read <a target="_blank" rel="noopener noreferrer" href="https://github.com/ChainSafe/gossamer/blob/development/.github/CODE_OF_CONDUCT.md">Code of Conduct </a> and <a target="_blank" rel="noopener noreferrer" href="https://github.com/ChainSafe/gossamer/blob/development/.github/CONTRIBUTING.md">Contributing Guidelines</a> before getting started.
