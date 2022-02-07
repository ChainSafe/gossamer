FROM parity/polkadot:v0.9.10 as polkadot
FROM golang:1.17

ARG key
ARG CHAIN=cross-client
ARG DD_API_KEY=somekey
ARG METRICS_NAMESPACE=substrate.local.devnet

ENV key=${key}
ENV CHAIN=${CHAIN}
ENV DD_API_KEY=${DD_API_KEY}

RUN test -n "$key"
RUN DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"

COPY --from=polkadot /usr/bin/polkadot /usr/bin/polkadot

WORKDIR /gossamer

COPY go.mod go.sum ./
RUN go mod download

COPY . . 

ARG METRICS_NAMESPACE=substrate.local.devnet

WORKDIR /gossamer/devnet

RUN go run cmd/update-dd-agent-confd/main.go -n=${METRICS_NAMESPACE} -t=key:${key} > /etc/datadog-agent/conf.d/openmetrics.d/conf.yaml

ENTRYPOINT service datadog-agent start && /usr/bin/polkadot \
    --bootnodes /dns/alice/tcp/7001/p2p/12D3KooWMER5iow67nScpWeVqEiRRx59PJ3xMMAYPTACYPRQbbWU \
    --chain chain/${CHAIN}/genesis-raw.json \
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