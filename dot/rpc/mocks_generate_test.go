// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . API,TransactionStateAPI
//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE . Telemetry
//go:generate mockgen -destination=mock_network_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/core Network
