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