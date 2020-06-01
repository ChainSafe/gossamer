---
layout: default
title: Get Started
permalink: /get-started/
---

## Prerequisites

install go version `1.13.7`

## Installation

get the [ChainSafe/gossamer](https://github.com/ChainSafe/gossamer) repository:
```
go get -u github.com/ChainSafe/gossamer
```

build gossamer command:
```
make gossamer
```

## Run Default Node

initialize default node:
```
./bin/gossamer --key alice init
```

start default node:
```
./bin/gossamer --key alice
```

## Run Gossamer Node

initialize gossamer node:
```
./bin/gossamer --chain gssmr --key alice init
```

start gossamer node:
```
./bin/gossamer --chain gssmr --key alice
```

## Run Kusama Node

initialize kusama node:
```
./bin/gossamer --chain ksmcc --key alice init
```

start kusama node:
```
./bin/gossamer --chain ksmcc --key alice
```

## Run Polkadot Node

initialize polkadot node:
```
./bin/gossamer --chain dotcc --key alice init
```

start polkadot node:
```
./bin/gossamer --chain dotcc --key alice
```
