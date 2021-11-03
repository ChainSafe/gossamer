#!/bin/bash

if [[ -z "${GOPATH}" ]]; then 
	export GOPATH=~/go
fi

if ! command -v golangci-lint &> /dev/null
then
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.43.0
fi

export PATH=$PATH:$(go env GOPATH)/bin