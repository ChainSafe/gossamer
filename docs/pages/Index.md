---
layout: default
title: Index
permalink: /
---

<br/>
<div align="center">
  <img alt="Gossamer logo" src="/assets/img/gossamer_banner_white.png" width="500" />
</div>
<div align="center">
  <p><i><b>Gossamer Docs</b> - The Official Documentation for Gossamer</i></p>
</div>
<br/>

## Welcome

***Gossamer*** is an implementation of the [Polkadot Host](https://github.com/w3f/polkadot-spec) - a blockchain framework used to build and run node implementations for different blockchain protocols within the Polkadot ecosystem.

Gossamer includes official node implementations for major blockchains within the Polkadot ecosystem and makes building node implementations for other blockchains trivial; blockchains built with [Substrate](https://github.com/paritytech/substrate) can plug their compiled runtime into Gossamer to create a node implementation in Go.

***Gossamer Docs*** is an evolving set of documents and resources to help you understand Gossamer, the Polkadot ecosystem, and how to build and run nodes using Gossamer. 

- If you are new to Gossamer and the Polkadot ecosystem, we recommend starting with [this video](https://www.youtube.com/watch?v=nYkbYhM5Yfk) and then working your way through [General Resources](/general-resources/).

- If you are already familiar with Gossamer and the Polkadot ecosystem, or you just want to dive in, head over to [Get Started](Get-Started) to run your first node using Gossamer.

- If you are looking to build a node with Gossamer, learn how Gossamer can be used to build and run custom node implementations using Gossamer as a framework (see below).

## Framework

Gossamer is a ***modular blockchain framework*** used to build and run node implementations for different blockchain protocols within the Polkadot ecosystem.

- The ***simplest*** way to use the framework is using the base node implementation with a custom configuration file (see [Configuration](/configuration/)).

- The ***more advanced***  way to use the framework is using the base node implementation with a compiled runtime and custom runtime imports (see [Import a Runtime](/import-a-runtime/)). 

- The ***most advanced***  way to use the framework is building custom node services or a custom base node implementation (see [Custom Services](/custom-services/)).

## Table of Contents

- **[Run Nodes](/run-nodes/)**

    - [Get Started](/get-started/)
    - [Commands](/commands/)

- **[Build Nodes](/build-nodes/)**

    - [Configuration](/configuration/)
    - [Import a Runtime](/import-a-runtime/)
    - [Custom Services](/custom-services/)

- **[Implementation](/implementation/)**

    - [Package Overview](/package-overview/)
    - [Host Architecture](/host-architecture/)
    - [Integration Tests](/integration-tests/)

- **[Resources](/resources/)**

    - [General Resources](/general-resources/)
    - [Developer Resources](/developer-resources/)

- **[Appendix](/appendix/)**

    - [SCALE Examples](/scale-examples/)

## Contributing

Let us know if you have any feedback on how we can improve our documentation or if there are any resources you would like to see added. Also, contributions are welcome! If you would like to contribute, please read [Code of Conduct](https://github.com/ChainSafe/gossamer/blob/development/.github/CODE_OF_CONDUCT.md) and [Contributing Guidelines](https://github.com/ChainSafe/gossamer/blob/development/.github/CONTRIBUTING.md).
