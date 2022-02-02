# This is the build stage for Substrate. Here we create the binary.
FROM parity/polkadot:v0.9.10

ARG key
RUN test -n "$key"
ENV key=${key}

ARG DD_API_KEY=somekey
ENV DD_API_KEY=${DD_API_KEY}
RUN DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"

COPY devnet/chain/gssmr/genesis-raw.json genesis-spec.json

ENTRYPOINT /usr/bin/polkadot \
    --bootnodes=/dns/substrate-alice/tcp/30333/p2p/12D3KooWDpJ7As7BWAwRMfu1VU2WCqNjvq387JEYKDBj4kx6nXTN \
    --chain genesis-spec.json \
    --${key} \
    --tmp

EXPOSE 30333 9933 9944 9615