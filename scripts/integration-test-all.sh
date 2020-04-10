#!/bin/bash

# "stable" mode tests assume data is static
# "live" mode tests assume data dynamic

SCRIPT=$(basename ${BASH_SOURCE[0]})
TEST=""
QTD=1
TIMEOUT=5

PORT="700"
RPC_PORT="854"
IP_ADDR="0.0.0.0"
MODE="stable"

KEY="alice"

usage() {
  echo "Usage: $SCRIPT"
  echo "Optional command line arguments"
  echo "-t <string>  -- Test to run. eg: rpc"
  echo "-q <number>  -- Quantity of nodes to run. eg: 3"
  echo "-o <number>  -- Timeout in secs to wait. eg: 5"
  exit 1
}

while getopts "h?t:q:o:" args; do
case $args in
    h|\?)
      usage;
      exit;;
    t ) TEST=${OPTARG};;
    q ) QTD=${OPTARG};;
    o ) TIMEOUT=${OPTARG};;
  esac
done

set -euxo pipefail

DATA_DIR=$(mktemp -d -t gossamer-datadir.XXXXX)

if [[ ! "$DATA_DIR" ]]; then
  echo "Could not create $DATA_DIR"
  exit 1
fi

# Compile gossamer
echo "compiling gossamer"
make build

# PID array declaration
arr=()

start_func() {
  echo "starting gossamer node $i in background ..."
  "$PWD"/bin/gossamer --port=$PORT"$i" --key=$KEY --datadir="$DATA_DIR$i" \
    --rpc --rpchost=$IP_ADDR --rpcport=$RPC_PORT"$i" --rpcmods=system,author >"$DATA_DIR"/node"$i".log 2>&1 & disown

  GOSSAMER_PID=$!
  echo "started gossamer node, pid=$GOSSAMER_PID"
  #add PID to array
  arr+=("$GOSSAMER_PID")
}

# Run node with static blockchain database
# For loop N times
for i in $(seq 1 "$QTD"); do
  start_func "$i" #&# Put a function in the background
  echo "sleeping $TIMEOUT seconds for startup"
  sleep "$TIMEOUT"
  echo "done sleeping"
done

set +e

# TODO: iterate and test all nodes ?
HOST_RPC=http://$IP_ADDR:$RPC_PORT"1"

if [[ -z $TEST || $TEST == "rpc" ]]; then

  GOSSAMER_INTEGRATION_TEST_MODE=$MODE GOSSAMER_NODE_HOST=$HOST_RPC go test ./tests/rpc/... -timeout=60s -v -count=1

  RPC_FAIL=$?

fi

stop_func() {
  GOSSAMER_PID=$i
  echo "shutting down gossamer node, pid=$GOSSAMER_PID"

  # Shutdown gossamer node
  kill -9 "$GOSSAMER_PID"
  wait "$GOSSAMER_PID"
}


for i in "${arr[@]}"; do
  stop_func "$i" #&
done

if [[ (-z $TEST || $TEST == "rpc") && $RPC_FAIL -ne 0 ]]; then
  exit $RPC_FAIL
else
  exit 0
fi
