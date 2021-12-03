---
layout: default
title: Using Prometheus
permalink: /integrate/using-prometheus/
---

# Using Prometheus Locally

To get started with Prometheus locally make sure you have installed [Docker](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/install/).

The docker-compose.yml file has, currently, the Prometheus to collect metrics, so to start that service you can execute (in the project root folder):

```
docker-compose up (-d to disatach the terminal)

or

docker-compose up prometheus (-d to disatach the terminal)
```

the above command will starts the Prometheus service on `0.0.0.0:9090`.

### Prometheus

Actually the Prometheus service reads a file `prometheus.yml` placed in the root level project folder, this file contains the definitions that Prometheus needs to collect the metrics. 

Linux: In the **job_name == gossamer** the **targets** property should be `[localhost:9876]`

To publish metrics from the node use the flag **--publish-metrics**; i.e, `./bin/gossamer --chain {chain} --key {key} --publish-metrics`