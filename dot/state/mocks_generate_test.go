// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/telemetry Client
//go:generate mockgen -destination=mocks_chaindb_test.go -package $GOPACKAGE github.com/ChainSafe/chaindb Database
