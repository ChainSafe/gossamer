# This is the build stage for Substrate. Here we create the binary.
FROM parity/polkadot:v0.9.10

COPY devnet/chain/gssmr/genesis-raw.json genesis-spec.json

ENTRYPOINT /usr/bin/polkadot \
    --chain genesis-spec.json \
    --alice \
    --node-key 0000000000000000000000000000000000000000000000000000000000000000 \
    --tmp

EXPOSE 30333 9933 9944 9615