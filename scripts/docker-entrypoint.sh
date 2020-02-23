#!/usr/bin/env bash
mkdir -p ~/gossamer-dev;
DATA_DIR=~/gossamer-dev

set -euxo pipefail

if [ ! -f $DATA_DIR/genesis_created ]; then
	/usr/local/gossamer init --genesis=/gocode/src/github.com/ChainSafe/gossamer/node/gssmr/genesis.json
	touch $DATA_DIR/genesis_created;
fi;

exec "$@"
