// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE github.com/ChainSafe/gossamer/devnet/cmd/scale-down-ecs-service/internal ECSAPI
