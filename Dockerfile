FROM ubuntu:18.04 as builder

# Install GCC
RUN apt-get update && \
    apt-get install -y \
    gcc \
    cmake \
    wget \
    curl \
    npm 

# Install node source for polkadotjs tests
RUN curl -sL https://deb.nodesource.com/setup_14.x | bash -

# Install nodejs for polkadotjs tests
RUN apt-get update && \
    apt-get install -y \
    nodejs

# Install Go
RUN wget https://golang.org/dl/go1.16.7.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.16.7.linux-amd64.tar.gz

# Install subkey
RUN wget -P /usr/local/bin/ https://chainbridge.ams3.digitaloceanspaces.com/subkey-v2.0.0
RUN mv /usr/local/bin/subkey-v2.0.0 /usr/local/bin/subkey
RUN chmod +x /usr/local/bin/subkey

# Configure go env vars
ENV GO111MODULE=on
ENV GOPATH=/gocode
ENV GOROOT=/usr/local/go
ENV PATH=$PATH:$GOPATH/bin:$GOROOT/bin

# Prepare structure and change dir
RUN mkdir -p $GOPATH/src/github.com/ChainSafe/gossamer
WORKDIR $GOPATH/src/github.com/ChainSafe/gossamer

# Add go mod lock files and gossamer default config
COPY go.mod .
COPY go.sum .

# Get go mods
RUN go mod download

# Copy gossamer sources
COPY . $GOPATH/src/github.com/ChainSafe/gossamer

# Install js dependencies for polkadot.js tests
RUN cd $GOPATH/src/github.com/ChainSafe/gossamer/tests/polkadotjs_test && npm install

# Build
RUN GOBIN=$GOPATH/src/github.com/ChainSafe/gossamer/bin go run scripts/ci.go install

# Create symlink
RUN ln -s $GOPATH/src/github.com/ChainSafe/gossamer/bin/gossamer /usr/local/gossamer

# Give permissions
RUN chmod +x $GOPATH/src/github.com/ChainSafe/gossamer/scripts/docker-entrypoint.sh

# Expose gossamer command and port
ENTRYPOINT ["/gocode/src/github.com/ChainSafe/gossamer/scripts/docker-entrypoint.sh"]
CMD ["/usr/local/gossamer"]
EXPOSE 7001 8546 8540
