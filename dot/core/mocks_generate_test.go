// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

//go:generate mockgen -destination=mocks_test.go -package $GOPACKAGE . BlockState,StorageState,TransactionState,Network,CodeSubstitutedState,Telemetry,BlockImportDigestHandler
//go:generate mockgen -destination=mock_runtime_instance_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
