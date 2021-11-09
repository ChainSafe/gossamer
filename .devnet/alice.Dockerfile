FROM golang:1.17

ARG chain="polkadot"
ARG basepath="~/.gossamer"
# ARG DD_API_KEY

ENV GO111MODULE=on

ENV chain=${chain}
ENV basepath=${basepath}
# ENV DD_API_KEY=${DD_API_KEY}

# RUN ["sh", "-c", "DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_API_KEY=${DD_API_KEY} DD_SITE=\"datadoghq.com\" bash -c \"$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)\""]

WORKDIR /gossamer

COPY . .

# RUN ["sh", "-c", "mv .github/workflows/staging/openmetrics.d/${chain}-conf.yaml /etc/datadog-agent/conf.d/openmetrics.d/conf.yaml"]
# RUN ls -la /etc/datadog-agent/conf.d/openmetrics.d/
# RUN cat /etc/hosts
# RUN service datadog-agent start

RUN go get ./...
RUN go build github.com/ChainSafe/gossamer/cmd/gossamer

RUN gossamer --key alice init

# use a hardcoded key for alice, so we can determine what the peerID is for subsequent nodes

RUN cp devnet/alice.node.key ~/.gossamer/gssmr/node.key

RUN ls -la ~/.gossamer/gssmr

RUN cat ~/.gossamer/gssmr/node.key

ENTRYPOINT gossamer --key=alice --babe-lead --publish-metrics --rpc --rpc-external=true
EXPOSE 7001 8545 8546 8540 9876