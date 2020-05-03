#!/usr/bin/env bash

PROJECTNAME=$(shell basename "$(PWD)")
GOLANGCI := $(GOPATH)/bin/golangci-lint
COMPANY=chainsafe
NAME=gossamer
VERSION=latest
FULLDOCKERNAME=$(COMPANY)/$(NAME):$(VERSION)

.PHONY: help lint test install build clean start docker gossamer
all: help
help: Makefile
	@echo
	@echo " Choose a make command to run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

$(GOLANGCI):
	wget -O - -q https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s latest

## lint: Lints project files, go gets golangci-lint if missing. Runs `golangci-lint` on project files.
.PHONY: lint
lint: $(GOLANGCI)
	GOBIN=$(PWD)/bin go run scripts/ci.go lint

clean:
	rm -fr ./bin

format:
	./scripts/goimports.sh

## test: Runs `go test` on project test files.
test:
	@echo "  >  \033[32mRunning tests...\033[0m "
	GOBIN=$(PWD)/bin go run scripts/ci.go test

## it-stable: Runs Integration Tests Stable mode
it-stable:
	@echo "  >  \033[32mRunning Integration Tests...\033[0m "
	@chmod +x scripts/integration-test-all.sh
	./scripts/integration-test-all.sh -q 3 -s 10

## it-stress: Runs Integration Tests stress mode
it-stress: build
	@echo "  >  \033[32mRunning Integration Tests stress mode...\033[0m "
	HOSTNAME=0.0.0.0 GOSSAMER_INTEGRATION_TEST_MODE=stress go test ./tests/stress/... -timeout=5m -p 1 -short -v

it-rpc: build
	@echo "  >  \033[32mRunning Integration Tests RPC Specs mode...\033[0m "
	HOSTNAME=0.0.0.0 GOSSAMER_INTEGRATION_TEST_MODE=rpc_suite go test ./tests/rpc/... -timeout=5m -p 1 -short -v

## test: Runs `go test -race` on project test files.
test-state-race:
	@echo "  >  \033[32mRunning race tests...\033[0m "
	go test ./dot/state/... -race -timeout=5m

## install: Install missing dependencies. Runs `go mod download` internally.
install:
	@echo "  >  \033[32mInstalling dependencies...\033[0m "
	go mod download

## build: Builds application binary and stores it in `./bin/gossamer`
build:
	@echo "  >  \033[32mBuilding binary...\033[0m "
	GOBIN=$(PWD)/bin go run scripts/ci.go install

# init: Initialize gossamer using the default genesis and toml configuration files
init:
	./bin/gossamer init --verbosity debug

## start: Starts application from binary executable in `./bin/gossamer`
start:
	@echo "  >  \033[32mStarting server...\033[0m "
	./bin/gossamer

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
	docker build -t $(FULLDOCKERNAME) -f Dockerfile.dev .

gossamer: clean
	GOBIN=$(PWD)/bin go run scripts/ci.go install
