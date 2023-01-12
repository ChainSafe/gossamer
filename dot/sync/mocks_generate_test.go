// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . BlockState,StorageState,TransactionState,BabeVerifier,FinalityGadget,BlockImportHandler,Network
//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE . Telemetry
//go:generate mockgen -destination=mock_runtime_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
//go:generate mockgen -source=interfaces_mock_source.go -destination=mocks_local_test.go -package=$GOPACKAGE
