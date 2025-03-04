// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE . Telemetry
//go:generate mockgen -destination=mocks/runtime.go -package mocks github.com/ChainSafe/gossamer/lib/runtime Instance
//go:generate mockgen -destination=mocks/core.go -package mocks github.com/ChainSafe/gossamer/dot/core Network,BlockImportDigestHandler
//go:generate mockgen -destination=mock_state_test.go -package $GOPACKAGE . ImportedBlockNotifierManager,TransactionState,EpochState,BlockImportHandler,SlotState
//go:generate mockgen -destination=mock_block_state_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/state BlockState
//go:generate mockgen -destination=mock_storage_state_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/state StorageState
