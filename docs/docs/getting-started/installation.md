---
layout: default
title: Installation
permalink: /getting-started/installation
---

# Get Started

## Prerequisites

Install [Go](https://go.dev/doc/install) version [`>=1.20`](https://go.dev/dl/#go1.20)

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

Initialise polkadot node:

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
