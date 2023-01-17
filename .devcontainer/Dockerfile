# Copyright 2021 ChainSafe Systems (ON)
# SPDX-License-Identifier: LGPL-3.0-only

FROM qmcgaw/godevcontainer:debian
ENV CGO_ENABLED=1
RUN apt-get update && \
  apt-get install -y make htop procps cpulimit
# Limit the Gossamer CPU usage with:
# cpulimit -l 10 -p "$(pidof gossamer)"
RUN wget -qO- https://deb.nodesource.com/setup_14.x | bash - && \
  apt-get install -y nodejs

RUN go install github.com/google/addlicense@v1.0.0

ARG PROTOC_VERSION=21.10
RUN wget -qO protoc.zip "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip" && \
  apt-get install -y unzip && \
  unzip protoc.zip -d /usr/local && \
  apt-get remove -y unzip && \
  rm protoc.zip && \
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
