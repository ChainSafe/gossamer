---
layout: default
title: Installation
permalink: /getting-started/installation
---

# Get Started

## Prerequisites

Install <a target="_blank" rel="noopener noreferrer" href="https://golang.org/">Go</a> version `>=1.15`

## Installation

Get the <a target="_blank" rel="noopener noreferrer" href="https://github.com/ChainSafe/gossamer">ChainSafe/gossamer</a> repository:
```
git clone git@github.com:ChainSafe/gossamer
cd gossamer
```

Run the following command to build the Gossamer CLI:
```
make gossamer
```

## Run a Gossamer Node

To run default Gossamer node, first initialise the node, this establishes the settings for your node:
```
./bin/gossamer --chain gssmr init
```

To start the node, run the following command to use a built in key:
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

## Run Kusama Node (_in development_)

To run a Kusama node, first initialise the node:
```
./bin/gossamer --chain ksmcc init
```

Then run the node selecting the Kusama chain:
```
./bin/gossamer --chain ksmcc
```

The node may not appear to do anything for the first minute or so (it's bootstrapping to the network.) If you wish to see what is it doing in this time, you can turn on debug logs in `chain/ksmcc/config.toml`:

```
[log]
network = "debug"
```

After it's finished bootstrapping, the node should begin to sync. 

## Run Polkadot Node (_in development_)

NOTE: This is currently not supported.

initialize polkadot node:
```
./bin/gossamer --chain polkadot init
```

start polkadot node:
```
./bin/gossamer --chain polkadot
```