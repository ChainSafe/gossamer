---
layout: default
title: Debugging
permalink: /testing-and-debugging/debugging
---

## Logger Commands

The ```log``` command is used for setting the logging levels for the node.
## Config File
The logging level can be set using config.toml file, default log level will be set to info

```[log]
core = "trace | debug | info | warn | error | crit"
sync = "trace | debug | info | warn | error | crit"
network = "trace | debug | info | warn | error | crit"
rpc = "trace | debug | info | warn | error | crit"
state = "trace | debug | info | warn | error | crit"
runtime = "trace | debug | info | warn | error | crit"
babe = "trace | debug | info | warn | error | crit"
grandpa = "trace | debug | info | warn | error | crit"
```

## Logging Global Flags
```--log value        Supports levels crit (silent) to trce (trace) (default: "info")```

## Running node with log level as `DEBUG`
```./bin/gossamer --config chain/gssmr/config.toml --log debug```
