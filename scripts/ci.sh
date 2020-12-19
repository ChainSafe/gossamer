#!/bin/bash

# Copyright 2019 ChainSafe Systems (ON) Corp.
# This file is part of gossamer.
#
# The gossamer library is free software: you can redistribute it and/or modify
# it under the terms of the GNU Lesser General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# The gossamer library is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Lesser General Public License for more details.
#
# You should have received a copy of the GNU Lesser General Public License
# along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

set -e

echo ">> Running tests..."
# IPFS_LOGGING=debug go test -v -short -coverprofile c.out ./...

echo ">> Running DHT Test"
IPFS_LOGGING=debug go test -v -short -coverprofile c.out github.com/ChainSafe/gossamer/dot/network -run Test

./cc-test-reporter after-build --exit-code $?
# echo ">> Running race condition test on runtime"
# go test -short -race ./lib/runtime
echo ">> Running race condition test on priority queue"
go test -v -short -race ./lib/transaction/
