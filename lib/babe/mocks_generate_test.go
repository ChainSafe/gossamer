// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE . Telemetry
//go:generate mockgen -destination=mocks/runtime.go -package mocks github.com/ChainSafe/gossamer/dot/core RuntimeInstance
//go:generate mockgen -destination=mocks/network.go -package mocks github.com/ChainSafe/gossamer/dot/core Network
