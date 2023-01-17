# Copyright 2021 ChainSafe Systems (ON)
# SPDX-License-Identifier: LGPL-3.0-only

FROM qmcgaw/godevcontainer:debian
ENV CGO_ENABLED=1
RUN apt-get update && \
  apt-get install -y make htop
RUN wget -qO- https://deb.nodesource.com/setup_14.x | bash - && \
  apt-get install -y nodejs
RUN apt-get install -y procps cpulimit
# Use with: cpulimit -l 10 -p "$(pidof gossamer)"

# TODO: once Alpine is properly supported
# ARG GLIBC_VERSION=2.34-r0
# RUN wget -qO /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
#   wget -qO /tmp/glibc.apk https://github.com/sgerrand/alpine-pkg-glibc/releases/download/${GLIBC_VERSION}/glibc-${GLIBC_VERSION}.apk && \
#   apk add /tmp/glibc.apk && \
#   rm /tmp/glibc.apk

RUN go install github.com/google/addlicense@v1.0.0

ARG PROTOC_VERSION=21.10
RUN wget -qO protoc.zip "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip" && \
  apt-get install -y unzip && \
  unzip protoc.zip -d /usr/local && \
  apt-get remove -y unzip && \
  rm protoc.zip && \
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
