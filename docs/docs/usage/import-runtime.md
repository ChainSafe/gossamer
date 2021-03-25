---
layout: default
title: Import Runtime
permalink: /usage/import-runtime/
---

# Launching a chain with a custom runtime

The main use-case of the Polkadot Host is to create a standalone chain that may be converted to a parachain later.  To do this, you need to have a compiled wasm runtime available for your chain. This can be created using [FRAME](https://substrate.dev/docs/en/knowledgebase/runtime/frame), a domain-specific language used for creating runtimes.

Once you have your runtime ready and compiled into a wasm binary, it is ready to be used with Gossamer.

### 1. Create genesis spec file with custom runtime

You can use the `gossamer import-runtime` subcommand to create a genesis file containin g your custom runtime. The rest of the file is based off the `gssmr` `genesis.json` file.

```
make gossamer
./bin/gossamer import-runtime <custom-runtime.wasm> > genesis-spec.json
```

This creates a genesis spec file `genesis-spec.json` with the contents of your given file as the `"system"` `"code"` field. 

By default, `genesis-spec.json` will contain the 9 built-in keys as authorities with some preset balance. You can edit the fields as you wish.

Note: the `import-runtime` subcommand does not validate that the runtime in the given file is valid. 

### 2. Create raw genesis file from genesis spec

To create the raw genesis file used by the node, you can use the `gossamer build-spec` subcommand.

```
./bin/gossamer build-spec --raw --genesis genesis-spec.json > genesis.json
```

This creates a genesis file `genesis.json` that is usable by the node.

### 3. Initialize the node with the genesis file

Next, you will need to write the state in `genesis.json` to the database by initializing the node.

```
./bin/gossamer init --genesis genesis.json
```

The default basepath is `~/.gossamer/gssmr`. If you wish you change this,  you can specify the basepath with `--basepath <path>`.

### 4. Start the node

The final step is to launch the node. This is the same as normal, providing a built-in authority key:
```
./bin/gossamer --key alice
```

You now have a chain running a custom runtime!