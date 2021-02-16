#!/usr/bin/env bash

PROJECTNAME=$(shell basename "$(PWD)")
COMPANY=chainsafe
NAME=gossamer
ifndef VERSION
VERSION=latest
endif
FULLDOCKERNAME=$(COMPANY)/$(NAME):$(VERSION)

.PHONY: help lint test install build clean start docker gossamer
all: help
help: Makefile
	@echo
	@echo " Choose a make command to run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

## lint: Lints project files, go gets golangci-lint if missing. Runs `golangci-lint` on project files.
.PHONY: lint
lint: 
	./scripts/install-lint.sh
	${GOPATH}/bin/golangci-lint run

clean:
	rm -fr ./bin

format:
	./scripts/goimports.sh

proto:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	protoc -I=./dot/network/proto --go_out=./dot/network/proto dot/network/proto/api.v1.proto

## test: Runs `go test` on project test files.
test:
	@echo "  >  \033[32mRunning tests...\033[0m "
	#GOBIN=$(PWD)/bin go run scripts/ci.go test
	go test -short -coverprofile c.out ./... -timeout=15m

## it-stable: Runs Integration Tests Stable mode
it-stable:
	@echo "  >  \033[32mRunning Integration Tests...\033[0m "
	@chmod +x scripts/integration-test-all.sh
	./scripts/integration-test-all.sh -q 3 -s 10

## it-stress: Runs Integration Tests stress mode
it-stress: build
	@echo "  >  \033[32mRunning stress tests...\033[0m "
	HOSTNAME=0.0.0.0 MODE=stress go test ./tests/stress/... -timeout=15m -v -short -run TestSync_

it-grandpa: build
	@echo "  >  \033[32mRunning GRANDPA stress tests...\033[0m "
	HOSTNAME=0.0.0.0 MODE=stress go test ./tests/stress/... -timeout=12m -v -short -run TestStress_Grandpa_

it-rpc: build
	@echo "  >  \033[32mRunning Integration Tests RPC Specs mode...\033[0m "
	HOSTNAME=0.0.0.0 MODE=rpc go test ./tests/rpc/... -timeout=10m -v

it-sync: build
	@echo "  >  \033[32mRunning Integration Tests sync mode...\033[0m "
	HOSTNAME=0.0.0.0 MODE=sync go test ./tests/sync/... -timeout=5m -v

it-polkadotjs: build
	@echo "  >  \033[32mRunning Integration Tests polkadot.js/api mode...\033[0m "
	HOSTNAME=0.0.0.0 MODE=polkadot go test ./tests/polkadotjs_test/... -timeout=5m -v

## test: Runs `go test -race` on project test files.
test-state-race:
	@echo "  >  \033[32mRunning race tests...\033[0m "
	go test ./dot/state/... -short -race -timeout=5m

## deps: Install missing dependencies. Runs `go mod download` internally.
deps:
	@echo "  >  \033[32mInstalling dependencies...\033[0m "
	go mod download

## build: Builds application binary and stores it in `./bin/gossamer`
build:
	@echo "  >  \033[32mBuilding binary...\033[0m "
	GOBIN=$(PWD)/bin go run scripts/ci.go install

## debug: Builds application binary with debug flags and stores it in `./bin/gossamer`
build-debug:
	@echo "  >  \033[32mBuilding binary...\033[0m "
	GOBIN=$(PWD)/bin go run scripts/ci.go install-debug

## init: Initialize gossamer using the default genesis and toml configuration files
init:
	./bin/gossamer --key alice init --genesis-raw chain/gssmr/genesis-raw.json --force

## init-repo: Set initial configuration for the repo
init-repo:
	git config core.hooksPath .githooks

## start: Starts application from binary executable in `./bin/gossamer` with built-in key alice
start:
	@echo "  >  \033[32mStarting node...\033[0m "
	./bin/gossamer --key alice

$(ADDLICENSE):
	go get -u github.com/google/addlicense

## license: Adds license header to missing files, go gets addLicense if missing. Runs `addlicense -c gossamer -f ./copyright.txt -y 2019 .` on project files.
.PHONY: license
license: $(ADDLICENSE)
	@echo "  >  \033[32mAdding license headers...\033[0m "
	addlicense -c gossamer -f ./copyright.txt -y 2019 .

docker: docker-build
	@echo "  >  \033[32mStarting Gossamer Container...\033[0m "
	docker run --rm $(FULLDOCKERNAME)

docker-version:
	@echo "  >  \033[32mStarting Gossamer Container...\033[0m "
	docker run -it $(FULLDOCKERNAME) /bin/bash -c "/usr/local/gossamer --version"

docker-build:
	@echo "  >  \033[32mBuilding Docker Container...\033[0m "
	docker build -t $(FULLDOCKERNAME) -f Dockerfile .

gossamer: clean
	cd cmd/gossamer && go build -o ../../bin/gossamer && cd ../..

## install: install the gossamer binary in $GOPATH/bin
install:
	GOBIN=$(GOPATH)/bin go run scripts/ci.go install