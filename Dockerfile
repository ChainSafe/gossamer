FROM ubuntu:18.04 as builder

# Install GCC
RUN apt-get update && \
    apt-get install -y \
    gcc \
    cmake \
    wget

# Install Go
RUN wget https://dl.google.com/go/go1.15.5.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.15.5.linux-amd64.tar.gz

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

# Build
RUN GOBIN=$GOPATH/src/github.com/ChainSafe/gossamer/bin go run scripts/ci.go install

# Create symlink
RUN ln -s $GOPATH/src/github.com/ChainSafe/gossamer/bin/gossamer /usr/local/gossamer

# Give permissions
RUN chmod +x $GOPATH/src/github.com/ChainSafe/gossamer/scripts/docker-entrypoint.sh

# Expose gossamer command and port
ENTRYPOINT ["/gocode/src/github.com/ChainSafe/gossamer/scripts/docker-entrypoint.sh"]
CMD ["/usr/local/gossamer"]
EXPOSE 7001
