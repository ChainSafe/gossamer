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

Run the following command to build the Gossamer binary:
```
make gossamer
```

## Run a Gossamer Node

To run default Gossamer node, first initialize the node. This writes the genesis state to the database.
```
./bin/gossamer --chain gssmr init
```

The gossamer node runs as an authority by default. The built-in authorities are `alice`, `bob`, `charlie`, `dave`, `eve`, `ferdie`, `george`, and `ian`. To start the node as an authority, provide it with a built-in key:
```
./bin/gossamer --chain gssmr --key alice
```


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

If you wish to run the default node as a non-authority, you can specify `roles=1`:
```
./bin/gossamer --chain gssmr --roles 1
```

## Run Kusama Node

To run a Kusama node, first initialise the node:
```
./bin/gossamer --chain kusama init
```

Then run the node selecting the Kusama chain:
```
./bin/gossamer --chain kusama
```

The node may not appear to do anything for the first minute or so (it's bootstrapping to the network.) If you wish to see what is it doing in this time, you can turn on debug logs in `chain/kusama/config.toml`:

```
[log]
network = "debug"
```

After it's finished bootstrapping, the node should begin to sync. 

## Run Polkadot Node 

Initialize polkadot node:
```
./bin/gossamer --chain polkadot init
```

Start polkadot node:
```
./bin/gossamer --chain polkadot
```

## Run Gossamer Node with Docker

Gossamer can also be installed on GNU/Linux, MacOS systems with Docker. 

### Dependencies

- Install the latest release of [Docker](https://docs.docker.com/get-docker/)

Ensure you are running the most recent version of Docker by issuing the command: 

```
docker -v
```

Pull the latest Gossamer images from DockerHub Registry: 

```
docker pull chainsafe/gossamer:latest
```

The above command will install all required dependencies.  

Next, we need override the default entrypoint so we can run the node as an authority node

```
docker run -it --entrypoint /bin/bash chainsafe/gossamer:latest
```

The built-in authorities are `alice`, `bob`, `charlie`, `dave`, `eve`, `ferdie`, `george`, and `ian`. To start the node as an authority, provide it with a built-in key:
```
./bin/gossamer --chain gssmr --key alice
```
