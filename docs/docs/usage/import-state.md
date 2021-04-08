---
layout: default
title: Import Sate
permalink: /usage/import-state/
---

# Gossamer state import

## Importing state

Gossamer supports the ability to import state exported from gossamer or substrate. To retrieve the state from an existing node, you will need to run the node in **archive** mode. For example, with Kusama:
```
./target/release/polkadot --chain=kusama --pruning archive --rpc-methods unsafe --rpc-port 8545
```

Since we will be using the RPC method `state_getPairs` which is marked `unsafe`, you will need to use the `--rpc-methods unsafe` option.

Once the node has synced to the height you wish to export, you can export the state by first finding the block hash of the block you wish to export (can use polkascan.io) or RPC. For example, for block 1000:
```
curl -H "Content-Type: application/json" -d '{"id":1, "jsonrpc":"2.0", "method": "chain_getBlockHash", "params":[1000]}' http://localhost:8545 
{"jsonrpc":"2.0","result":"0xcf36a1e4a16fc579136137b8388f35490f09c5bdd7b9133835eba907a8b76c30","id":1}
```

For the following steps, you will need `jq` installed.

Then, you can get the state at that block and redirect the output to a file `state.json`:
```
curl -H "Content-Type: application/json" -d '{"id":1, "jsonrpc":"2.0", "method": "state_getPairs", "params":["0x", "0xcf36a1e4a16fc579136137b8388f35490f09c5bdd7b9133835eba907a8b76c30"]}' http://localhost:8545 | jq '.result' > state.json
```

Then, get the header of the block:
```
curl -H "Content-Type: application/json" -d '{"id":1, "jsonrpc":"2.0", "method": "chain_getHeader", "params":["0xcf36a1e4a16fc579136137b8388f35490f09c5bdd7b9133835eba907a8b76c30"]}' http://localhost:8545 | jq '.result' > header.json
```

Lastly, get the first slot of the network. This can be find on polkascan.io. First, go to the network which you are importing then search for `1` (ie. block 1). Then, navigate to `Logs -> PreRuntime -> Details`.  It will then show the `slotNumber`, for example, for Kusama, the first slot number is `262493679`: https://polkascan.io/kusama/log/1-0

Now you have all the required info to import the state into gossamer.

In the `gossamer` directory:
```
make gossamer 
./bin/gossamer --chain <chain-name> init --force
./bin/gossamer import-state --chain <chain-name> --state state.json  --header header.json --first-slot <first-slot>
```

If you don't want to use a specific chain, but instead a custom data directory, you can use `--basepath` instead of `--chain`.

If it is successful, you will see a `finished state import` log. Now, you can start the node as usual, and the node should begin from the imported state:
```
./bin/gossamer --chain <chain-name>
```