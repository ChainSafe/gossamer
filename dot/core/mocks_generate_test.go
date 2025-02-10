// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

//go:generate mockgen -destination=mocks_test.go -package $GOPACKAGE . StorageState,TransactionState,Network,CodeSubstitutedState,Telemetry,BlockImportDigestHandler,GrandpaState
//go:generate mockgen -destination=mock_runtime_instance_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/lib/runtime Instance
//go:generate mockgen -destination=mock_block_state_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/state BlockState
