FROM golang:1.17

ARG key
RUN test -n "$key"

# ARG DD_API_KEY

ENV GO111MODULE=on

ENV key=${key}
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

# use modified genesis-spec.json with only 3 authority nodes
RUN rm chain/gssmr/genesis-spec.json
RUN cp .devnet/chain/gssmr/genesis-spec.json chain/gssmr/genesis-spec.json

RUN ["sh", "-c", "gossamer --key=${key} init"]
ENTRYPOINT ["sh", "-c", "gossamer --key=${key} --bootnodes=/ip4/10.5.0.2/tcp/7001/p2p/12D3KooWMER5iow67nScpWeVqEiRRx59PJ3xMMAYPTACYPRQbbWU --rpc --rpc-external=true"]

EXPOSE 7001 8545 8546 8540 9876