---
layout: default
title: Using Prometheus
permalink: /integrate/using-prometheus/
---

1. Install [Docker](https://docs.docker.com/engine/install/)
1. Install [Docker Compose](https://docs.docker.com/compose/install/).
1. 📥 [Download the repository](https://github.com/ChainSafe/gossamer/archive/refs/heads/development.zip) or `git clone https://github.com/ChainSafe/gossamer.git` it.
1. 🏃 You can run the repository [docker-compose.yml](https://github.com/ChainSafe/gossamer/blob/development/docker-compose.yml) with `docker-compose up`.
By default, it will run a Gossamer node running on the Kusama chain, together with a Prometheus server and Grafana server. Both Prometheus and Grafana are pre-configured to show a nice dashboard of the metrics. All the relevant configuration lives in the `docker` directory of the repository.
1. 🖱️ Access the Grafana dashboard at [localhost:3000](http://localhost:3000/), there is no login required.

💁 You can modify the `docker` directory and the `docker-compose.yml` file to match the desired configuration.
