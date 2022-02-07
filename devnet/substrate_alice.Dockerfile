FROM parity/polkadot:v0.9.10 as polkadot
FROM golang:1.17

ARG CHAIN=cross-client
ARG VERSION=v0.9.10
ARG DD_API_KEY=somekey
ARG METRICS_NAMESPACE=substrate.local.devnet
ARG NODE_KEY="93ce444331ced4d2f7bfb8296267544e20c2591dbf310c7ea3af672f2879cf8f"

ENV DD_API_KEY=${DD_API_KEY}

RUN DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"

COPY --from=polkadot /usr/bin/polkadot /usr/bin/polkadot

WORKDIR /gossamer

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /gossamer/devnet

RUN go run cmd/update-dd-agent-confd/main.go -n=${METRICS_NAMESPACE} -t=key:alice > /etc/datadog-agent/conf.d/openmetrics.d/conf.yaml

ENTRYPOINT service datadog-agent start && /usr/bin/polkadot \
    --chain chain/$CHAIN/genesis-raw.json \
    --alice \
    --port 7001 \
    --rpc-port 8545 \
    --ws-port 8546 \
    --node-key $NODE_KEY \ 
    --tmp \
    --prometheus-external \
    --prometheus-port 9876 \
    --unsafe-rpc-external \
    --unsafe-ws-external

EXPOSE 7001/tcp 8545/tcp 8546/tcp 9876/tcp