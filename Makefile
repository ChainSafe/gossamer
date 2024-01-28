#!/usr/bin/env bash

PROJECTNAME=$(shell basename "$(PWD)")
COMPANY=chainsafe
NAME=gossamer
ifndef VERSION
VERSION=latest
endif
FULLDOCKERNAME=$(COMPANY)/$(NAME):$(VERSION)
OS:=$(shell uname)

.PHONY: help lint test install build clean start docker gossamer build-debug
all: help
help: Makefile
	@echo
	@echo " Choose a make command to run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

.PHONY: lint
lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54
	golangci-lint run

clean:
	rm -fr ./bin


proto:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	@echo "Protoc is required for this target, download it from https://github.com/protocolbuffers/protobuf/releases"
	go generate -run protoc ./...

## test: Runs `go test` on project test files.
test:
	@echo "  >  \033[32mRunning tests...\033[0m "
	git lfs pull
	go test -coverprofile c.out ./... -timeout=30m

## it-stress: Runs Integration Tests stress mode
it-stress: build
	@echo "  >  \033[32mRunning stress tests...\033[0m "
	MODE=stress go test ./tests/stress/... -timeout=15m -v -run TestSync_

it-grandpa: build
	@echo "  >  \033[32mRunning GRANDPA stress tests...\033[0m "
	MODE=stress go test ./tests/stress/... -timeout=12m -v -short -run TestStress_Grandpa_

it-rpc: build
	@echo "  >  \033[32mRunning Integration Tests RPC Specs mode...\033[0m "
	MODE=rpc go test ./tests/rpc/... -timeout=10m -v

it-sync: build
	@echo "  >  \033[32mRunning Integration Tests sync mode...\033[0m "
	MODE=sync go test ./tests/sync/... -timeout=5m -v

it-polkadotjs: build
	@echo "  >  \033[32mRunning Integration Tests polkadot.js/api mode...\033[0m "
	MODE=polkadot go test ./tests/polkadotjs_test/... -timeout=5m -v

## test: Runs `go test -race` on project test files.
test-using-race-detector:
	@echo "  >  \033[32mRunning race tests...\033[0m "
	go test ./dot/state/... -race -timeout=5m
	go test ./dot/sync/... -race -timeout=5m
	go test ./internal/log/... -race -timeout=5m
	go test ./lib/blocktree/... -race -timeout=5m
	go test ./lib/grandpa/... -race -timeout=5m


## deps: Install missing dependencies. Runs `go mod download` internally.
deps:
	@echo "  >  \033[32mInstalling dependencies...\033[0m "
	go mod download

## build: Builds application binary and stores it in `./bin/gossamer`
build: compile-erasure
	@echo "  >  \033[32mBuilding binary...\033[0m "
	go build -trimpath -o ./bin/gossamer -ldflags="-s -w" ./cmd/gossamer

## debug: Builds application binary with debug flags and stores it in `./bin/gossamer`
build-debug: clean
	go build -trimpath -gcflags=all="-N -l" -o ./bin/gossamer ./cmd/gossamer

## init: Initialise gossamer using the default genesis and toml configuration files
init:
	./bin/gossamer init --force

githooks:
	git config core.hooksPath .githooks

## start: Starts application from binary executable in `./bin/gossamer` with built-in key alice
start:
	@echo "  >  \033[32mStarting node...\033[0m "
	./bin/gossamer --key alice

## license: Adds license header to missing files, go install addlicense if it's missing.
.PHONY: license
license:
	@echo "  >  \033[32mAdding license headers...\033[0m "
	go install github.com/google/addlicense@v1.0
	addlicense -v \
		-s=only \
		-l="LGPL-3.0-only" \
		-f ./copyright.txt \
		-c "ChainSafe Systems (ON)" \
		-ignore "**/*.md" \
		-ignore "**/*.html" \
		-ignore "**/*.css" \
		-ignore "**/*.scss" \
		-ignore "**/*.yml" \
		-ignore "**/*.yaml" \
		-ignore "**/*.js" \
		-ignore "**/*.sh" \
		-ignore "*Dockerfile" \
		.

docker: docker-build
	@echo "  >  \033[32mStarting Gossamer Container...\033[0m "
	docker run --rm $(FULLDOCKERNAME)

docker-version:
	@echo "  >  \033[32mStarting Gossamer Container...\033[0m "
	docker run -it $(FULLDOCKERNAME) /bin/bash -c "/usr/local/gossamer --version"

docker-build:
	@echo "  >  \033[32mBuilding Docker Container...\033[0m "
	docker build -t $(FULLDOCKERNAME) -f Dockerfile .

gossamer: clean compile-erasure build

## install: install the gossamer binary in $GOPATH/bin
install: build
	mv ./bin/gossamer $(GOPATH)/bin/gossamer

install-zombienet:
ifeq ($(OS),Darwin)
	wget -O /usr/local/bin/zombienet https://github.com/paritytech/zombienet/releases/download/v1.3.41/zombienet-macos
else ifeq ($(OS),Linux)
		wget -O $(GOPATH)/bin/zombienet https://github.com/paritytech/zombienet/releases/download/v1.3.41/zombienet-linux-x64
else
		@echo "Zombienet for $(OS) is not supported"
		exit 1
endif
	chmod a+x $(GOPATH)/bin/zombienet

zombienet-test: install install-zombienet
	zombienet test -p native zombienet_tests/functional/0001-basic-network.zndsl

compile-erasure:
	cargo build --release --manifest-path=lib/erasure/rustlib/Cargo.toml
