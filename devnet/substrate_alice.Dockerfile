# This is the build stage for Substrate. Here we create the binary.
FROM parity/polkadot:v0.9.10

ARG DD_API_KEY=somekey
ENV DD_API_KEY=${DD_API_KEY}
RUN DD_AGENT_MAJOR_VERSION=7 DD_INSTALL_ONLY=true DD_SITE="datadoghq.com" bash -c "$(curl -L https://s3.amazonaws.com/dd-agent/scripts/install_script.sh)"

COPY devnet/chain/gssmr/genesis-raw.json genesis-spec.json

ENTRYPOINT /usr/bin/polkadot \
    --chain genesis-spec.json \
    --alice \
    --node-key 0000000000000000000000000000000000000000000000000000000000000000 \
    --tmp

EXPOSE 30333 9933 9944 9615