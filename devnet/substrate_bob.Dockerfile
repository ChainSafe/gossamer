# Copyright 2022 ChainSafe Systems (ON)
# SPDX-License-Identifier: LGPL-3.0-only

ARG POLKADOT_VERSION=v0.9.37

FROM golang:1.20 as openmetrics

ARG METRICS_NAMESPACE=substrate.local.devnet

WORKDIR /devnet

COPY ./devnet/go.mod ./devnet/go.sum ./
RUN go mod download

COPY ./devnet .
RUN go run cmd/update-dd-agent-confd/main.go -n=${METRICS_NAMESPACE} -t=key:alice > conf.yaml

FROM parity/polkadot:${POLKADOT_VERSION}

ARG CHAIN=westend-local
ARG DD_API_KEY=somekey
ARG key

ENV DD_API_KEY=${DD_API_KEY}
ENV CHAIN=${CHAIN}
ENV key=${key}

USER root
RUN gpg --recv-keys --keyserver hkps://keys.mailvelope.com 9D4B2B6EB8F97156D19669A9FF0812D491B96798
RUN gpg --export 9D4B2B6EB8F97156D19669A9FF0812D491B96798 > /usr/share/keyrings/parity.gpg

RUN apt update && apt install -y curl && rm -r /var/cache/* /var/lib/apt/lists/*

WORKDIR /cross-client

RUN curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh --output install_script.sh && \
    chmod +x ./install_script.sh

RUN DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" ./install_script.sh
COPY --from=openmetrics /devnet/conf.yaml /etc/datadog-agent/conf.d/openmetrics.d/

USER polkadot

COPY ./chain/ ./chain/

ENTRYPOINT  service datadog-agent start && /usr/bin/polkadot \
    --bootnodes /dns/alice/tcp/7001/p2p/12D3KooWMER5iow67nScpWeVqEiRRx59PJ3xMMAYPTACYPRQbbWU \
    --chain chain/$CHAIN/$CHAIN-spec-raw.json \
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
