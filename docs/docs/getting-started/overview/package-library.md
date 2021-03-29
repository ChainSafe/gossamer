---
layout: default
title: Package Library
permalink: /welcome/package-library/
---

## Overview

Gossamer is a **modular blockchain framework**; it was designed with a package structure that makes it possible to reuse Gossamer packages to build and run custom nodes and node services.

This document provides an overview of the packages that make up the Gossamer framework - more detailed information about each package can be found at <a target="_blank" rel="noopener noreferrer" href="https://pkg.go.dev/github.com/ChainSafe/gossamer">pkg.go.dev/ChainSafe/gossamer</a>.

Gossamer packages can be categorized into **four package types**:

- **[cmd packages](#cmd-packages)**

    - `cmd/...` - command-line packages for running nodes and other services

- **[host packages](#host-packages)**

    - `host/...` - the host node package and host node service packages

- **[lib packages](#lib-packages)**

    - `lib/...` - modular packages for building nodes and other services

- **[test packages](#test-packages)**

    - `tests/...` - test packages for integration tests

## cmd packages

#### `cmd/gossamer`

- The **gossamer package** is used to run nodes built with Gossamer.

## dot packages

#### `dot`

- The **dot package** contains packages that implement the Polkadot Host spec. The **dot package** implements the [Host Node](/getting-started/overview/host-architecture#host-node); it is the base node implementation for all [Official Nodes](/getting-started/overview/host-architecture#official-nodes) and [Custom Services](/getting-started/overview/host-architecture#custom-services) built with Gossamer.

#### `dot/core`

- The **core package** implements the [Core Service](/getting-started/overview/host-architecture#core-service) -  responsible for block production and block finalization (consensus) and processing messages received from the [Network Service](/getting-started/overview/host-architecture/#network-service).

#### `dot/network`

- The **network package** implements the [Network Service](/getting-started/overview/host-architecture/#network-service) - responsible for coordinating network host and peer interactions. It manages peer connections, receives and parses messages from connected peers and handles each message based on its type.

#### `dot/state`

- The **state package** implements the [State Service](/getting-started/overview/host-architecture#state-service) - the source of truth for all chain and node state that is made accessible to all node services.

#### `dot/sync`

- The **sync package** implements handling of blocks received from the network layer.

#### `dot/rpc`

- The **rpc package** implements the [RPC Service](/getting-started/overview/host-architecture#rpc-service) - an HTTP server that handles state interactions.

#### `dot/types`

- The **types package** implements types defined within the Polkadot Host specification.

## lib packages

#### `lib/babe`

- the **babe package** implements the BABE block production algorithm.

#### `lib/blocktree`

- the **blocktree package** implements the blocktree, a data structure which tracks the chain and all its non-finalized forks.

#### `lib/common`

- the **common package** defines common types and functions.

#### `lib/crypto`

- the **crypto package** contains the key types used by the node (sr25519, ed25519, secp256k1).

#### `lib/grandpa`

- the **grandpa package** implements the GRANDPA finality gadget.

#### `lib/keystore`

- the **keystore package** manages the keystore and includes test keyrings.

#### `lib/runtime`

- the **runtime package** contains various wasm interpreters used to interpret the runtime. It currently contains `life`, `wasmer`, and `wasmtime`; however, `wasmer` is the only interpreter that is fully supported at the moment. In the future, all interpreters will be fully supported.

#### `lib/scale`

- the **scale package** implements the SCALE codec.

#### `lib/services`

- the **services package** implements a common interface for node services.

#### `lib/transaction`

- the **transaction package** is contains transaction types and the transaction queue data structure.

#### `lib/trie`

- the **trie package** implements a modified merkle-patricia trie.

#### `lib/utils`

- the **utils package** is used to manage node and test directories.

## test packages

#### `tests/polkadotjs_test`
- the **polkadotjs_test section** is a node.js implementation to test interaction with Gossamer's RPC.

#### `tests/rpc`
- the **rpc test collection** comprises tests to cover the RPC's various endpoints and return data.

#### `tests/stress`
- the **stress test collection** comprises tests to simulate the node running under various high stress scenarios

#### `tests/sync`
- the **sync test collection** comprises tests to verify the integrity of syncing operations and features

#### `tests/utils`
- the **utils test collection** comprises tests to validate the function of various utilities throughout the Gossamer stack
