FROM golang:1.17

ARG key
RUN test -n "$key"
ARG pubip
RUN test -n "$pubip"
ARG DD_API_KEY=somekey

ENV key=${key}
ENV pubip=${pubip}
ENV DD_API_KEY=${DD_API_KEY}

RUN DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"
WORKDIR /gossamer

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go run devnet/cmd/update-dd-agent-confd/main.go -n=gossamer.local.devnet -t=key:${key} > /etc/datadog-agent/conf.d/openmetrics.d/conf.yaml
RUN service datadog-agent start

# RUN go get ./...
RUN go install github.com/ChainSafe/gossamer/cmd/gossamer

# use modified genesis-spec.json with only 3 authority nodes
RUN cp -f devnet/chain/gssmr/genesis-spec.json chain/gssmr/genesis-spec.json

RUN gossamer --key=${key} init

ENTRYPOINT service datadog-agent restart && gossamer --key=${key} --bootnodes=/ip4/10.5.0.2/tcp/7001/p2p/12D3KooWMER5iow67nScpWeVqEiRRx59PJ3xMMAYPTACYPRQbbWU --publish-metrics --rpc --pubip=${pubip}
EXPOSE 7001/tcp 8545/tcp 8546/tcp 8540/tcp 9876/tcp