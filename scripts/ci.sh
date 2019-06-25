#!/bin/bash

set -e

echo ">> Running tests..."
go test -v -short -coverprofile c.out ./...

echo ">> Reporting test results..."
./cc-test-reporter after-build --exit-code $?

echo ">> Done!"
