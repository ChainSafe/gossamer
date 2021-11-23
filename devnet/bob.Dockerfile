# Copyright 2021 ChainSafe Systems (ON)
# SPDX-License-Identifier: LGPL-3.0-only

FROM golang:1.17

ARG DD_API_KEY=somekey
RUN DD_API_KEY=${DD_API_KEY} DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"

WORKDIR /gossamer

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install -trimpath github.com/ChainSafe/gossamer/cmd/gossamer

# use modified genesis-spec.json with only 3 authority nodes
RUN cp -f devnet/chain/gssmr/genesis-spec.json chain/gssmr/genesis-spec.json

ARG key
RUN test -n "$key"
ARG pubip
RUN test -n "$pubip"

ENV key=${key}
ENV pubip=${pubip}

RUN gossamer --key=${key} init

RUN go run devnet/cmd/update-dd-agent-confd/main.go -n=gossamer.local.devnet -t=key:${key} > /etc/datadog-agent/conf.d/openmetrics.d/conf.yaml

ENTRYPOINT service datadog-agent start && gossamer --key=${key} --bootnodes=/ip4/10.5.0.2/tcp/7001/p2p/12D3KooWMER5iow67nScpWeVqEiRRx59PJ3xMMAYPTACYPRQbbWU --publish-metrics --rpc --pubip=${pubip}

EXPOSE 7001/tcp 8545/tcp 8546/tcp 8540/tcp 9876/tcp
