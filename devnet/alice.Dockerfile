# Copyright 2021 ChainSafe Systems (ON)
# SPDX-License-Identifier: LGPL-3.0-only

FROM golang:1.17

ARG DD_API_KEY=somekey
RUN DD_API_KEY=${DD_API_KEY}  DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"

WORKDIR /gossamer

COPY go.mod go.sum ./
RUN go mod download

COPY . . 

RUN go install -trimpath github.com/ChainSafe/gossamer/cmd/gossamer

# use modified genesis-spec.json with only 3 authority nodes
RUN cp -f devnet/chain/gssmr/genesis-spec.json chain/gssmr/genesis-spec.json

RUN gossamer --key=alice init

# use a hardcoded key for alice, so we can determine what the peerID is for subsequent nodes
RUN cp devnet/alice.node.key ~/.gossamer/gssmr/node.key

RUN go run devnet/cmd/update-dd-agent-confd/main.go -n=gossamer.local.devnet -t=key:alice > /etc/datadog-agent/conf.d/openmetrics.d/conf.yaml

ENTRYPOINT service datadog-agent start && gossamer --key=alice --babe-lead --publish-metrics --rpc --rpc-external=true --pubip=10.5.0.2
EXPOSE 7001/tcp 8545/tcp 8546/tcp 8540/tcp 9876/tcp
