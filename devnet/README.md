# Gossamer Devnet

Docker container and Docker Compose for a Gossamer Devnet currently with three authority nodes running the `gssmr` chain with keys `alice`, `bob` and `charlie`.

## Requirements

- [Docker Compose](https://docs.docker.com/compose/install/)

## Running the Devnet

From the Gossamer root run the following commands to run the devnet

```sh
# will rebuild the containers based on the current code
docker-compose up --abort-on-container-exit --build 

# will run the devnet without rebuilding
docker-compose up --abort-on-container-exit

# destroys the devnet
docker-compose down
```

> **_NOTE:_**  The devnet is not stateful, so subsequent runs will start from the genesis block.

## Running Cross Client Devnet

A cross-client devnet is network of gossamer node(s) and substrate node(s).

Steps to run a two node cross client devnet

- Create `genesis-spec.json`
```
./polkadot build-spec --disable-default-bootnode --dev > genesis-spec.json
```

- Edit `genesis-spec.json` as per your needs. Add extra authorities if you want to. In order to add extra authorities, you would want to edit "validatorCount", "minimumValidatorCount", "invulnerables", "stakers", and "session"."keys"

- Create `genesis.json` from `genesis-spec.json`.
```bash
./polkadot build-spec --chain genesis-spec.json --raw --disable-default-bootnode > genesis.json
```

- Initiate gossamer node
```
./bin/gossamer init --force --genesis ../polkadot-testing/genesis.json
```

- Run polkadot
```
./polkadot --alice --chain genesis.json
```

- Run gossamer
```
./bin/gossamer --key bob --bootnodes /ip4/127.0.0.1/tcp/30333/p2p/$PEERID_OF_POLKADOT_NODE
```

## Prometheus Datadog Integration

All Prometheus metrics from the nodes are piped to Datadog. You can setup your own dashboard and add additional tags by modifying the Dockerfiles.  Currently the metrics are prefixed with `gossamer.local.devnet` and are tagged (Prometheus label) with a `key` tag for `alice`, `bob`, and `charlie`.

For metrics to be piped to Datadog, you will require a Datadog API key.  Please contact Elizabeth or myself (Tim) for access to datadog if you don't already have it.

The Datadog API key must be an environment variable on your own machine, which Docker Compose will pick up and inject when building the node images.

```
export $DD_API_KEY=YourKey
```

## Files

### Dockerfiles

There are two Docker files used in the devnet.  `alice.Dockerfile` is the lead node and is intiated with the `babe-lead` flag to build the first block.  `bob.Dockerfile` is used for both `bob` and `charlie`.

### cmd/update-dd-agent-confd

A command line app to create a `confd.yml` file used by the Datadog agent when piping metrics to Datadog.  It's used in the both `alice.Dockerfile` and `bob.Dockerfile` to create specific `confd.yml` files.

### alice.node.key

This key is injected in `alice.Dockerfile` so it uses the same public key for the `bootnodes` param in `bob.Dockerfile`. 

### docker-compose.yml

The Docker Compose file.  Specifies the IP addresses of all the nodes. 


## ECS Fargate Deployment

The [`docker-compose.yml`](gssmr-ecs/docker-compose.yml) file within `devnet/gssmr-ecs` folder uses Docker Compose ECS plugin to deploy and update an existing AWS ECS Cluster using the Fargate launch type running a Gossamer devnet with 3 services corresponding to the 3 keys used `alice`, `bob`, and `charlie`.  

### Deployment

Currently deployment is handled via a github workflow.  Pushing the `devnet` branch will initiate the deploy process.  Steps are outlined in `/.github/workflows/devnet.yml`. 

At a high level, images for the `alice`, `bob` and `charlie` correspond to ECS services under the same name.  The docker images are built based on the latest commit on the `devnet` branch.  These images are pushed to ECR.  A specific type of Docker context is required to use the ECS plugin.  Deploying and updating is as simple as:

```
docker context create ecs gssmr-ecs --from-env
docker context use gssmr-ecs
docker compose up
```

### Prometheus to Datadog

Prometheus metrics are automatically piped to Datadog.  All metrics from the ECS devnet are prefixed with `gossamer.ecs.devnet`.  

