// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE . Telemetry
//go:generate mockgen -destination=mock_syncer_test.go -package $GOPACKAGE . Syncer
//go:generate mockgen -destination=mock_block_state_test.go -package $GOPACKAGE . BlockState
//go:generate mockgen -destination=mock_transaction_handler_test.go -package $GOPACKAGE . TransactionHandler
//go:generate mockgen -destination=mock_stream_test.go -package $GOPACKAGE github.com/libp2p/go-libp2p/core/network Stream
