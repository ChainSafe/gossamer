ARG DEBIAN_VERSION=buster-slim
ARG ALPINE_VERSION=3.14
ARG GO_VERSION=1.15-buster

FROM golang:${GO_VERSION} AS builder

RUN apt-get update && \
    apt-get install -y \
    gcc \
    cmake \
    wget \
    npm \
    # Install nodejs for polkadotjs tests
    nodejs

# Install node source for polkadotjs tests
RUN wget -qO- https://deb.nodesource.com/setup_14.x | bash -

# Install subkey
RUN wget -O /usr/local/bin/subkey https://chainbridge.ams3.digitaloceanspaces.com/subkey-v2.0.0 && \
    chmod +x /usr/local/bin/subkey

# Polkadot JS dependencies
WORKDIR /go/src/github.com/ChainSafe/gossamer/tests/polkadotjs_test
COPY tests/polkadotjs_test/package.json tests/polkadotjs_test/package-lock.json ./
RUN npm install

WORKDIR /go/src/github.com/ChainSafe/gossamer

# Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Prepare libwasmer.so for COPY
RUN cp /go/pkg/mod/github.com/wasmerio/go-ext-wasm@*/wasmer/libwasmer.so libwasmer.so

# Copy gossamer sources
COPY . .

# Build
RUN GOBIN=$GOPATH/src/github.com/ChainSafe/gossamer/bin go run scripts/ci.go install

# Final stage based on Alpine with glibc
FROM alpine:${ALPINE_VERSION}

# Install wget to have TLS validation
RUN apk add --update --no-cache wget

# Install (runtime) glibc
ARG GLIBC_VERSION=2.34-r0
RUN wget -qO /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
    wget -qO /tmp/glibc.apk https://github.com/sgerrand/alpine-pkg-glibc/releases/download/${GLIBC_VERSION}/glibc-${GLIBC_VERSION}.apk && \
    apk add /tmp/glibc.apk && \
    rm /tmp/glibc.apk

# Install C dependencies
RUN apk add libgcc musl

# Install libwasmer.so
ENV LD_LIBRARY_PATH=/lib:/usr/lib
COPY --from=builder /go/src/github.com/ChainSafe/gossamer/libwasmer.so /lib/libwasmer.so

# Install bash for retro-compatibility
RUN apk add --no-cache bash

EXPOSE 7001 8546 8540

ENTRYPOINT ["/gocode/src/github.com/ChainSafe/gossamer/scripts/docker-entrypoint.sh"]
CMD ["/usr/local/gossamer"]

WORKDIR /gocode/src/github.com/ChainSafe/gossamer
COPY chain chain
COPY scripts/docker-entrypoint.sh scripts/docker-entrypoint.sh
COPY --from=builder /go/src/github.com/ChainSafe/gossamer/bin/gossamer /gocode/src/github.com/ChainSafe/gossamer/bin/gossamer
RUN ln -s /gocode/src/github.com/ChainSafe/gossamer/bin/gossamer /usr/local/gossamer
