# Copyright 2021 ChainSafe Systems (ON)
# SPDX-License-Identifier: LGPL-3.0-only

FROM golang:1.20

ARG DD_API_KEY=somekey
ARG CHAIN=westend-local

ENV DD_API_KEY=${DD_API_KEY}

RUN DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"

WORKDIR /gossamer

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install -trimpath github.com/ChainSafe/gossamer/cmd/gossamer

# use the 3 node westend local network
RUN cp -f chain/$CHAIN/$CHAIN-spec-raw.json chain/$CHAIN/$CHAIN-spec.json

RUN gossamer init --key=alice --chain chain/$CHAIN/$CHAIN-spec.json --base-path ~/.gossamer/gssmr

# use a hardcoded key for alice, so we can determine what the peerID is for subsequent nodes
RUN cp devnet/alice.node.key ~/.gossamer/gssmr/node.key

ARG METRICS_NAMESPACE=gossamer.local.devnet

WORKDIR /gossamer/devnet

RUN go run cmd/update-dd-agent-confd/main.go -n=${METRICS_NAMESPACE} -t=key:alice > /etc/datadog-agent/conf.d/openmetrics.d/conf.yaml

WORKDIR /gossamer

ENTRYPOINT service datadog-agent start && gossamer --base-path ~/.gossamer/gssmr --key=alice --prometheus-external  --prometheus-port=9876 --rpc-external=true --public-dns=alice --port 7001

EXPOSE 7001/tcp 8545/tcp 8546/tcp 8540/tcp 9876/tcp 6060/tcp
