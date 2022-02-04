# This is the build stage for Substrate. Here we create the binary.
FROM parity/polkadot:v0.9.10
FROM docker.io/library/ubuntu:20.04

ARG key
ARG DD_API_KEY=somekey
ARG METRICS_NAMESPACE=substrate.local.devnet

ENV key=${key}
ENV DD_API_KEY=${DD_API_KEY}

RUN test -n "$key"
RUN DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"

COPY --from=polkadot /usr/bin/polkadot /usr/bin/polkadot
COPY devnet/chain/gssmr/genesis-raw.json genesis-spec.json

RUN go run cmd/update-dd-agent-confd/main.go -n=${METRICS_NAMESPACE} -t=key:${key} > /etc/datadog-agent/conf.d/openmetrics.d/conf.yaml

ENTRYPOINT service datadog-agent start && /usr/bin/polkadot \
    --bootnodes=/dns/substrate-alice/tcp/7001/p2p/12D3KooWDpJ7As7BWAwRMfu1VU2WCqNjvq387JEYKDBj4kx6nXTN \
    --chain genesis-spec.json \
    --port 7001 \
    --rpc-port 8545 \
    --ws-port 8546 \
    --${key} \
    --tmp \
    --prometheus-external \
    --prometheus-port 9876 \
    --unsafe-rpc-external \
    --unsafe-ws-external

EXPOSE 7001 8545 8546 9876