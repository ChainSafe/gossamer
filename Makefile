PKGS := $(shell go list ./... | grep -v /vendor)

.PHONY: test
test: lint
	go test $(PKGS)

GOLANGCI := $(GOPATH)/bin/golangci-lint

$(GOLANGCI):
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s latest

.PHONY: lint
lint: $(GOLANGCI)
	golangci-lint run -v

run:
	@echo "  >  \033[32mStarting server...\033[0m "
	go run *.go

build:
	@echo "  >  \033[32mBuilding binary...\033[0m "
	go build -o ./bin/gossamer

start:
	@echo "  >  \033[32mStarting server...\033[0m "
	./bin/gossamer

install:
	@echo "  >  \033[32mInstalling dependencies...\033[0m "
	go mod vendor

docker:
	@echo "  >  \033[32mBuilding Docker Container...\033[0m "
	docker build -t chainsafe/gossamer -f Dockerfile.dev .
	@echo "  >  \033[32mRunning Docker Container...\033[0m "
	docker run chainsafe/gossamer

## clean: Clean build files. Runs `go clean` internally.
clean:
	@echo "  >  \033[32mCleaning build cache...\033[0m "
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean
