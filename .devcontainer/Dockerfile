# Copyright 2021 ChainSafe Systems (ON)
# SPDX-License-Identifier: LGPL-3.0-only

FROM qmcgaw/godevcontainer:v0.12.0-debian
ENV CGO_ENABLED=1
RUN apt-get update && \
  apt-get install -y make htop
RUN wget -qO- https://deb.nodesource.com/setup_14.x | bash - && \
  apt-get install -y nodejs

# TODO: once Alpine is properly supported
# ARG GLIBC_VERSION=2.34-r0
# RUN wget -qO /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub && \
#   wget -qO /tmp/glibc.apk https://github.com/sgerrand/alpine-pkg-glibc/releases/download/${GLIBC_VERSION}/glibc-${GLIBC_VERSION}.apk && \
#   apk add /tmp/glibc.apk && \
#   rm /tmp/glibc.apk

RUN go install github.com/google/addlicense@v1.0.0

# TODO remove these once we can get the latest debian image
RUN echo "deb https://deb.debian.org/debian bookworm main" >> /etc/apt/sources.list && \
  apt-get update && \
  apt-get install -y -t bookworm git && \
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50 && \
  go install github.com/vektra/mockery/v2@v2.14
