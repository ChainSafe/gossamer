# Copyright 2022 ChainSafe Systems (ON)
# SPDX-License-Identifier: LGPL-3.0-only

FROM golang:1.17 as openmetrics

ARG METRICS_NAMESPACE=substrate.local.devnet

WORKDIR /devnet

COPY ./devnet/go.mod ./devnet/go.sum ./
RUN go mod download

COPY ./devnet .
RUN go run cmd/update-dd-agent-confd/main.go -n=${METRICS_NAMESPACE} -t=key:alice > conf.yaml

FROM parity/polkadot:v0.9.17

ARG key
ARG CHAIN=cross-client
ARG DD_API_KEY=somekey

ENV DD_API_KEY=${DD_API_KEY}
ENV CHAIN=${CHAIN}
ENV key=${key}

USER root
RUN apt update && apt install -y curl && rm -r /var/cache/* /var/lib/apt/lists/*

WORKDIR /cross-client

RUN curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh --output install_script.sh && \
    chmod +x ./install_script.sh

RUN DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" ./install_script.sh
COPY --from=openmetrics /devnet/conf.yaml /etc/datadog-agent/conf.d/openmetrics.d/

USER polkadot

COPY ./devnet/chain ./chain/

ENTRYPOINT service datadog-agent start && /usr/bin/polkadot \
    --bootnodes /dns/alice/tcp/7001/p2p/12D3KooWMER5iow67nScpWeVqEiRRx59PJ3xMMAYPTACYPRQbbWU \
    --chain chain/$CHAIN/genesis-raw.json \
    --port 7001 \
    --rpc-port 8545 \
    --ws-port 8546 \
    --${key} \
    --tmp \
    --prometheus-external \
    --prometheus-port 9876 \
    --unsafe-rpc-external \
    --unsafe-ws-external

EXPOSE 7001/tcp 8545/tcp 8546/tcp 9876/tcp
