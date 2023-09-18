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

ENV DD_API_KEY=${DD_API_KEY}
ENV CHAIN=${CHAIN}

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

# The substrate node-key argument should be a 32 bytes long sr25519 secret key
# while gossamer nodes uses a 64 bytes long sr25519 key (32 bytes long to secret key + 32 bytes long to public key).
# Then to keep both substrate and gossamer alice nodes with the same libp2p node keys we just need to use
# the first 32 bytes from `alice.node.key` which means the 32 bytes long sr25519 secret key used here.
ENTRYPOINT service datadog-agent start && /usr/bin/polkadot \
    --chain chain/$CHAIN/$CHAIN-spec-raw.json \
    --alice \
    --port 7001 \
    --rpc-port 8545 \
    --ws-port 8546 \
    --node-key "93ce444331ced4d2f7bfb8296267544e20c2591dbf310c7ea3af672f2879cf8f" \
    --tmp \
    --prometheus-external \
    --prometheus-port 9876 \
    --unsafe-rpc-external \
    --unsafe-ws-external

EXPOSE 7001/tcp 8545/tcp 8546/tcp 9876/tcp
