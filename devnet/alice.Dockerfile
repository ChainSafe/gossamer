FROM golang:1.17

ARG DD_API_KEY=somekey

ENV DD_API_KEY=${DD_API_KEY}

RUN echo $DD_API_KEY

RUN ["sh", "-c", "DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_API_KEY=${DD_API_KEY} DD_SITE=\"datadoghq.com\" bash -c \"$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)\""]

WORKDIR /gossamer

COPY . . 

RUN go run devnet/cmd/update-dd-agent-confd/main.go -n="gossamer.local.devnet" -t="key:alice" > /etc/datadog-agent/conf.d/openmetrics.d/conf.yaml
RUN service datadog-agent start

RUN go get ./...
RUN go build github.com/ChainSafe/gossamer/cmd/gossamer

# use modified genesis-spec.json with only 3 authority nodes
RUN cp -f devnet/chain/gssmr/genesis-spec.json chain/gssmr/genesis-spec.json

RUN gossamer --key alice init

# use a hardcoded key for alice, so we can determine what the peerID is for subsequent nodes
RUN cp devnet/alice.node.key ~/.gossamer/gssmr/node.key

ENTRYPOINT service datadog-agent restart && gossamer --key=alice --babe-lead --publish-metrics --rpc --rpc-external=true --pubip=10.5.0.2
EXPOSE 7001 8545 8546 8540 9876