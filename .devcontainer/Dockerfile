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
