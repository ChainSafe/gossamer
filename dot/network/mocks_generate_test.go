// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE . Telemetry
//go:generate mockgen -destination=mock_syncer_test.go -package $GOPACKAGE . Syncer
//go:generate mockgen -destination=mock_block_state_test.go -package $GOPACKAGE . BlockState
