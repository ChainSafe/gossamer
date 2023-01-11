// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE . Telemetry
//go:generate mockgen -destination=mock_state_test.go -package $GOPACKAGE . BlockState,ImportedBlockNotifierManager,StorageState,TransactionState,EpochState,BlockImportHandler
