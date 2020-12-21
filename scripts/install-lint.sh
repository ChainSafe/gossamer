#!/bin/bash

if ! command -v golangci-lint &> /dev/null
then
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.32.2
fi

export PATH=$PATH:$(go env GOPATH)/bin
echo $PATH
alias golangci-lint=$(go env GOPATH)/bin