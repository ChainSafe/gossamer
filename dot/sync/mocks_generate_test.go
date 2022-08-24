// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

//go:generate mockgen -destination=mocks_test.go -package=$GOPACKAGE . BlockState,StorageState,CodeSubstitutedState,TransactionState,BabeVerifier,FinalityGadget,BlockImportHandler,Network,RuntimeInstance
//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/telemetry Client
