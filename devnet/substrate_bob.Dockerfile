# This is the build stage for Substrate. Here we create the binary.
FROM parity/polkadot:v0.9.10

ARG key
RUN test -n "$key"
ENV key=${key}

COPY devnet/chain/gssmr/genesis-raw.json genesis-spec.json

ENTRYPOINT /usr/bin/polkadot \
    --bootnodes=/dns/substrate-alice/tcp/30333/p2p/12D3KooWDpJ7As7BWAwRMfu1VU2WCqNjvq387JEYKDBj4kx6nXTN \
    --chain genesis-spec.json \
    --${key} \
    --tmp

EXPOSE 30333 9933 9944 9615