# Gossamer Devnet

Docker container and Docker Compose for a Gossamer Devnet currently with three authority nodes running the `westend` chain with keys `alice`, `bob` and `charlie`.

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

>> optional: you can add the flag `-f` followed by the path to the docker-compose.yml file

> **_NOTE:_**  The devnet is not stateful, so subsequent runs will start from the genesis block.

## Prometheus Datadog Integration

All Prometheus metrics from the nodes are piped to Datadog. You can setup your own dashboard and add additional tags by modifying the Dockerfiles.  Currently the metrics are prefixed with `gossamer.local.devnet` and are tagged (Prometheus label) with a `key` tag for `alice`, `bob`, and `charlie`.

For metrics to be piped to Datadog, you will require a Datadog API key.  Please contact Elizabeth or myself (Tim) for access to datadog if you don't already have it.

The Datadog API key must be an environment variable on your own machine, which Docker Compose will pick up and inject when building the node images.

```
export $DD_API_KEY=YourKey
```

## Files

### Dockerfiles

There are four Docker files used in the devnet.  

- `alice.Dockerfile` is the lead node.  
- `bob.Dockerfile` is used for both `bob` and `charlie` and shares the same genesis as alice docker file.
- `substrate_alice.Dockerfile` is the alice node initiated with explicit node key to keep a deterministic peer id (the same used by gossamer alice node)
- `substrate_bob.Dockerfile` is used for `bob` and `charlie` and shares the same genesis as alice docker file.

> **_NOTE:_**: It is possible to use the substrate alice node with the bob and charlie gossamer nodes or any combination of these since the nodes in the network contain different keys

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

