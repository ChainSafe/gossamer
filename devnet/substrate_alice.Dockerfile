FROM parity/polkadot:v0.9.10 as polkadot
FROM golang:1.17

ARG VERSION=v0.9.10
ARG DD_API_KEY=somekey
ENV DD_API_KEY=${DD_API_KEY}
ARG METRICS_NAMESPACE=substrate.local.devnet

RUN apt-get update

RUN DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"

COPY --from=polkadot /usr/bin/polkadot /usr/bin/polkadot
COPY devnet/chain/gssmr/genesis-raw.json genesis-spec.json

RUN go run cmd/update-dd-agent-confd/main.go -n=${METRICS_NAMESPACE} -t=key:alice > /etc/datadog-agent/conf.d/openmetrics.d/conf.yaml

ENTRYPOINT ./polkadot \
    --chain genesis-spec.json \
    --alice \
    --port 7001 \
    --rpc-port 8545 \
    --ws-port 8546 \
    --node-key 0000000000000000000000000000000000000000000000000000000000000000 \
    --tmp \
    --prometheus-external \
    --prometheus-port 9876 \
    --unsafe-rpc-external \
    --unsafe-ws-external


EXPOSE 7001 8545 8546 9876