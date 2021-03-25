---
layout: default
title: Configuration
permalink: /usage/configuration/
---

# Configuration

Gossamer consumes a `.toml` file containing predefined settings for the node from setting the genesis file, to the RPC/WS server, this file allows you to curated the functionality of the node instead of writing out the flags manually

## Full reference

```toml
[global]
basepath = "~/.gossamer/gssmr" // TODO: confirm
log = " | trace | debug | info | warn | error | crit"
cpuprof = "~/cpuprof.txt"  // TODO: Syntax? 
memprof = "~/memprof.txt" // TODO: Syntax? 
name = "gssmr"

[log]
core = " | trace | debug | info | warn | error | crit"
network = " | trace | debug | info | warn | error | crit"
rpc = " | trace | debug | info | warn | error | crit"
state = " | trace | debug | info | warn | error | crit"
runtime = " | trace | debug | info | warn | error | crit"
babe = " | trace | debug | info | warn | error | crit"
grandpa = " | trace | debug | info | warn | error | crit"
sync = " | trace | debug | info | warn | error | crit"

[init]
genesis-raw = "./chain/gssmr/genesis-raw.json"

[account]
key = ""
unlock = ""

[core]
roles = 4
babe-authority = true
grandpa-authority = true

[network]
port = 7001
nobootstrap = false
nomdns = false

[rpc]
enabled = true | false
external = true | false
port = 8545
host = "localhost"
modules = ["system", "author", "chain", "state", "rpc", "grandpa"]
ws = true | false
ws-external = true | false
ws-port = 8546
```