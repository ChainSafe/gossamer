#!/bin/bash

if ! command -v golangci-lint &> /dev/null
then
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.32.2
fi